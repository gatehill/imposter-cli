/*
Copyright © 2021 Pete Cornish <outofcoffee@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"context"
	"fmt"
	"gatehill.io/imposter/debounce"
	"gatehill.io/imposter/engine"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

const engineDockerImage = "outofcoffee/imposter"
const containerConfigDir = "/opt/imposter/config"
const removalTimeoutSec = 5

type DockerMockEngine struct {
	configDir   string
	options     engine.StartOptions
	containerId string
	debouncer   debounce.Debouncer
}

func BuildEngine(configDir string, options engine.StartOptions) engine.MockEngine {
	return &DockerMockEngine{
		configDir: configDir,
		options:   options,
		debouncer: debounce.Build(),
	}
}

func (d *DockerMockEngine) Start() {
	d.startWithOptions(d.options)
}

func (d *DockerMockEngine) startWithOptions(options engine.StartOptions) {
	logrus.Infof("starting mock engine on port %d", options.Port)
	ctx, cli := buildCliClient()

	imageAndTag, err := ensureContainerImage(cli, ctx, options.Version, options.PullPolicy)
	if err != nil {
		logrus.Fatal(err)
	}

	containerPort := nat.Port(fmt.Sprintf("%d/tcp", options.Port))
	hostPort := fmt.Sprintf("%d", options.Port)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageAndTag,
		Cmd: []string{
			"--configDir=" + containerConfigDir,
			fmt.Sprintf("--listenPort=%d", options.Port),
		},
		Env: []string{
			"IMPOSTER_LOG_LEVEL=" + strings.ToUpper(options.LogLevel),
		},
		ExposedPorts: nat.PortSet{
			containerPort: {},
		},
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: d.configDir,
				Target: containerConfigDir,
			},
		},
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: hostPort,
				},
			},
		},
	}, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	logrus.Info("mock engine started - press ctrl+c to stop")

	containerLogs, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
	})
	if err != nil {
		panic(err)
	}

	go func() {
		_, err := stdcopy.StdCopy(os.Stdout, os.Stderr, containerLogs)
		if err != nil {
			panic(err)
		}
	}()

	d.containerId = resp.ID
}

func buildCliClient() (context.Context, *client.Client) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return ctx, cli
}

func (d *DockerMockEngine) Stop() {
	stopCh := make(chan debounce.AtMostOnceEvent)
	d.TriggerRemovalAndNotify(stopCh)
	<-stopCh
}

func (d *DockerMockEngine) TriggerRemovalAndNotify(stopCh chan debounce.AtMostOnceEvent) {
	if len(d.containerId) == 0 {
		logrus.Tracef("no container ID to remove")
		stopCh <- debounce.AtMostOnceEvent{}
		return
	}

	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		logrus.Tracef("stopping mock engine container %v", d.containerId)
	} else {
		logrus.Info("stopping mock engine")
	}

	oldContainerId := d.containerId

	d.debouncer.Register(oldContainerId)

	// supervisor to work-around removal race
	go func() {
		time.Sleep(removalTimeoutSec * time.Second)
		logrus.Tracef("fired timeout supervisor for container %v removal", oldContainerId)
		d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: oldContainerId})
	}()

	d.removeAndNotify(oldContainerId, stopCh)
}

func (d *DockerMockEngine) removeAndNotify(containerId string, stopCh chan debounce.AtMostOnceEvent) {
	go func() {
		ctx, cli := buildCliClient()

		// check it exists
		_, err := cli.ContainerInspect(ctx, containerId)
		if err != nil {
			if !client.IsErrNotFound(err) {
				logrus.Warnf("failed to find mock engine container %v to remove: %v", containerId, err)
				d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: containerId, Err: err})
			} else {
				d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: containerId})
			}
			return
		}

		err = cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			if !client.IsErrNotFound(err) {
				logrus.Warnf("failed to remove mock engine container %v: %v", containerId, err)
				d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: containerId, Err: err})
			} else {
				d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: containerId})
			}
			return
		}

		statusCh, errCh := cli.ContainerWait(ctx, containerId, container.WaitConditionRemoved)
		select {
		case err := <-errCh:
			if err != nil && !client.IsErrNotFound(err) {
				logrus.Warnf("error waiting for removal of mock engine container %v: %v", containerId, err)
				d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: containerId, Err: err})
			} else {
				d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: containerId})
			}
			break
		case <-statusCh:
			logrus.Tracef("mock engine container %v removed", containerId)
			d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: containerId})
			break
		}
	}()
}

func (d *DockerMockEngine) NotifyOnStop(stopCh chan debounce.AtMostOnceEvent) {
	oldContainerId := d.containerId

	go func() {
		ctx, cli := buildCliClient()

		statusCh, errCh := cli.ContainerWait(ctx, oldContainerId, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil && !client.IsErrNotFound(err) {
				logrus.Warnf("failed to wait for mock engine container to stop: %v", err)
				d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: oldContainerId, Err: err})
			} else {
				d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: oldContainerId})
			}
			break
		case <-statusCh:
			d.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: oldContainerId})
			break
		}
	}()
}

func (d *DockerMockEngine) BlockUntilStopped() {
	stopCh := make(chan debounce.AtMostOnceEvent)
	d.NotifyOnStop(stopCh)
	<-stopCh
}

func (d *DockerMockEngine) Restart(stopCh chan debounce.AtMostOnceEvent) {
	d.TriggerRemovalAndNotify(stopCh)

	// don't pull again
	restartOptions := d.options
	restartOptions.PullPolicy = engine.PullSkip

	d.startWithOptions(restartOptions)
}
