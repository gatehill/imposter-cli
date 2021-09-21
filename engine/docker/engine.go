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
	"sync"
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

func (d *DockerMockEngine) Start(wg *sync.WaitGroup) {
	d.startWithOptions(wg, d.options)
}

func (d *DockerMockEngine) startWithOptions(wg *sync.WaitGroup, options engine.StartOptions) {
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

	d.debouncer.Register(wg, resp.ID)
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
			logrus.Warnf("error streaming container logs: %v", err)
		}
	}()

	d.containerId = resp.ID

	engine.WaitUntilUp(options.Port)
}

func buildCliClient() (context.Context, *client.Client) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return ctx, cli
}

func (d *DockerMockEngine) Stop(wg *sync.WaitGroup) {
	if len(d.containerId) == 0 {
		logrus.Tracef("no container ID to remove")
		wg.Done()
		return
	}
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		logrus.Tracef("stopping mock engine container %v", d.containerId)
	} else {
		logrus.Info("stopping mock engine")
	}

	oldContainerId := d.containerId

	// supervisor to work-around removal race
	go func() {
		time.Sleep(removalTimeoutSec * time.Second)
		logrus.Tracef("fired timeout supervisor for container %v removal", oldContainerId)
		d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: oldContainerId})
	}()

	d.removeAndNotify(wg, oldContainerId)
}

func (d *DockerMockEngine) removeAndNotify(wg *sync.WaitGroup, containerId string) {
	go func() {
		ctx, cli := buildCliClient()

		// check it exists
		_, err := cli.ContainerInspect(ctx, containerId)
		if err != nil {
			if !client.IsErrNotFound(err) {
				logrus.Warnf("failed to find mock engine container %v to remove: %v", containerId, err)
				d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId, Err: err})
			} else {
				d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId})
			}
			return
		}

		err = cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			if !client.IsErrNotFound(err) {
				logrus.Warnf("failed to remove mock engine container %v: %v", containerId, err)
				d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId, Err: err})
			} else {
				d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId})
			}
			return
		}

		d.notifyOnStopSync(wg, containerId)
	}()
}

func (d *DockerMockEngine) Restart(wg *sync.WaitGroup) {
	innerWg := &sync.WaitGroup{}
	innerWg.Add(1)

	d.Stop(innerWg)
	innerWg.Wait()

	// don't pull again
	restartOptions := d.options
	restartOptions.PullPolicy = engine.PullSkip

	d.startWithOptions(wg, restartOptions)
	wg.Done()
}

func (d *DockerMockEngine) NotifyOnStop(wg *sync.WaitGroup) {
	oldContainerId := d.containerId
	go func() { d.notifyOnStopSync(wg, oldContainerId) }()
}

func (d *DockerMockEngine) notifyOnStopSync(wg *sync.WaitGroup, oldContainerId string) {
	ctx, cli := buildCliClient()
	statusCh, errCh := cli.ContainerWait(ctx, oldContainerId, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil && !client.IsErrNotFound(err) {
			logrus.Warnf("failed to wait for mock engine container to stop: %v", err)
			d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: oldContainerId, Err: err})
		} else {
			d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: oldContainerId})
		}
		break
	case <-statusCh:
		logrus.Tracef("mock engine container %v stopped", oldContainerId)
		d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: oldContainerId})
		break
	}
}

func (d *DockerMockEngine) BlockUntilStopped() {
	wg := &sync.WaitGroup{}
	d.NotifyOnStop(wg)
	wg.Wait()
}
