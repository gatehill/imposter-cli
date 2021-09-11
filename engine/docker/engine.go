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
	"gatehill.io/imposter/engine"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

const engineDockerImage = "outofcoffee/imposter"
const containerConfigDir = "/opt/imposter/config"
const removalTimeoutSec = 5

type DockerEngine struct {
	configDir   string
	options     engine.StartOptions
	containerId string
	stopMutex   *sync.Mutex
	stopping    map[string]bool
}

func BuildEngine(configDir string, options engine.StartOptions) engine.MockEngine {
	return &DockerEngine{
		configDir: configDir,
		options:   options,
		stopMutex: &sync.Mutex{},
		stopping:  make(map[string]bool),
	}
}

func (d *DockerEngine) Start() {
	d.startWithOptions(d.options)
}

func (d *DockerEngine) startWithOptions(options engine.StartOptions) {
	logrus.Infof("starting mock engine on port %d", options.Port)
	ctx, cli := buildCliClient()

	imageAndTag, err := ensureContainerImage(cli, ctx, options.ImageTag, options.ImagePullPolicy)

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

func ensureContainerImage(cli *client.Client, ctx context.Context, imageTag string, imagePullPolicy engine.ImagePullPolicy) (imageAndTag string, e error) {
	imageAndTag = engineDockerImage + ":" + imageTag

	if imagePullPolicy == engine.ImagePullSkip {
		return imageAndTag, nil
	}

	if imagePullPolicy == engine.ImagePullIfNotPresent {
		var hasImage = true
		_, _, err := cli.ImageInspectWithRaw(ctx, imageAndTag)
		if err != nil {
			if client.IsErrNotFound(err) {
				hasImage = false
			} else {
				panic(err)
			}
		}
		if hasImage {
			logrus.Debugf("engine image '%v' already present", imageTag)
			return imageAndTag, nil
		}
	}

	err := pullImage(cli, ctx, imageTag, imageAndTag)
	return imageAndTag, err
}

func pullImage(cli *client.Client, ctx context.Context, imageTag string, imageAndTag string) error {
	logrus.Infof("pulling '%v' engine image", imageTag)
	reader, err := cli.ImagePull(ctx, "docker.io/"+imageAndTag, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	var pullLogDestination io.Writer
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		pullLogDestination = os.Stdout
	} else {
		pullLogDestination = ioutil.Discard
	}
	_, err = io.Copy(pullLogDestination, reader)
	if err != nil {
		panic(err)
	}
	return err
}

func (d *DockerEngine) Stop() {
	stopCh := make(chan string)
	d.TriggerRemovalAndNotify(stopCh)
	<-stopCh
}

func (d *DockerEngine) TriggerRemovalAndNotify(stopCh chan string) {
	if len(d.containerId) == 0 {
		logrus.Tracef("no container ID to remove")
		stopCh <- ""
		return
	}

	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		logrus.Tracef("stopping mock engine container %v", d.containerId)
	} else {
		logrus.Info("stopping mock engine")
	}

	oldContainerId := d.containerId

	d.pushStoppingContainer(oldContainerId)

	// supervisor to work-around removal race
	go func() {
		time.Sleep(removalTimeoutSec * time.Second)
		logrus.Tracef("fired timeout supervisor for container %v removal", oldContainerId)
		d.popStoppingContainer(stopCh, oldContainerId)
	}()

	d.notifyOnRemoval(oldContainerId, stopCh)
}

func (d *DockerEngine) pushStoppingContainer(oldContainerId string) {
	d.stopMutex.Lock()
	d.stopping[oldContainerId] = true
	d.stopMutex.Unlock()
}

// popStoppingContainer debounces container stop events
func (d *DockerEngine) popStoppingContainer(stopCh chan string, containerId string) {
	if d.stopping[containerId] {
		d.stopMutex.Lock()
		if d.stopping[containerId] { // double-guard
			delete(d.stopping, containerId)
		}
		d.stopMutex.Unlock()
		stopCh <- containerId
	}
}

func (d *DockerEngine) notifyOnRemoval(containerId string, stopCh chan string) {
	go func() {
		ctx, cli := buildCliClient()

		// check it exists
		_, err := cli.ContainerInspect(ctx, containerId)
		if err != nil {
			if !client.IsErrNotFound(err) {
				logrus.Warnf("failed to find mock engine container %v to remove: %v", containerId, err)
			}
			d.popStoppingContainer(stopCh, containerId)
			return
		}

		err = cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			if !client.IsErrNotFound(err) {
				logrus.Warnf("failed to remove mock engine container %v: %v", containerId, err)
			}
			d.popStoppingContainer(stopCh, containerId)
			return
		}

		statusCh, errCh := cli.ContainerWait(ctx, containerId, container.WaitConditionRemoved)
		select {
		case err := <-errCh:
			if err != nil && !client.IsErrNotFound(err) {
				logrus.Warnf("error waiting for removal of mock engine container %v: %v", containerId, err)
			}
			d.popStoppingContainer(stopCh, containerId)
			break
		case <-statusCh:
			logrus.Tracef("mock engine container %v removed", containerId)
			d.popStoppingContainer(stopCh, containerId)
			break
		}
	}()
}

func (d *DockerEngine) NotifyOnStop(stopCh chan string) {
	oldContainerId := d.containerId

	go func() {
		ctx, cli := buildCliClient()

		statusCh, errCh := cli.ContainerWait(ctx, oldContainerId, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				if !client.IsErrNotFound(err) {
					logrus.Warnf("failed to wait for mock engine container to stop: %v", err)
				}
			}
			d.popStoppingContainer(stopCh, oldContainerId)
			break
		case <-statusCh:
			d.popStoppingContainer(stopCh, oldContainerId)
			break
		}
	}()
}

func (d *DockerEngine) BlockUntilStopped() {
	stopCh := make(chan string)
	d.NotifyOnStop(stopCh)
	<-stopCh
}

func (d *DockerEngine) Restart(stopCh chan string) {
	d.TriggerRemovalAndNotify(stopCh)

	// don't pull again
	restartOptions := d.options
	restartOptions.ImagePullPolicy = engine.ImagePullSkip

	d.startWithOptions(restartOptions)
}
