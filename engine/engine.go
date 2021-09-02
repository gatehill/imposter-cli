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
package engine

import (
	"context"
	"fmt"
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
)

type EngineStartOptions struct {
	Port           int
	ImageTag       string
	ForceImagePull bool
	LogLevel       string
}

const engineDockerImage = "outofcoffee/imposter"
const containerConfigDir = "/opt/imposter/config"

func StartMockEngine(configDir string, options EngineStartOptions) (containerId string) {
	logrus.Infof("starting mock engine on port %d", options.Port)
	ctx, cli := buildCliClient()

	imageAndTag, err := ensureContainerImage(cli, ctx, options.ImageTag, options.ForceImagePull)

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
				Source: configDir,
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

	return resp.ID
}

func buildCliClient() (context.Context, *client.Client) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	return ctx, cli
}

func ensureContainerImage(cli *client.Client, ctx context.Context, imageTag string, forcePull bool) (imageAndTag string, e error) {
	imageAndTag = engineDockerImage + ":" + imageTag

	if !forcePull {
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

func StopMockEngine(containerId string) {
	logrus.Infof("\rstopping mock engine...\n")
	ctx, cli := buildCliClient()

	err := cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		logrus.Warnf("failed to remove mock engine container: %v", err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, containerId, container.WaitConditionRemoved)
	select {
	case err := <-errCh:
		if err != nil {
			logrus.Warnf("failed to remove mock engine container: %v", err)
		}
	case <-statusCh:
	}
	logrus.Trace("mock engine container removed")
}

func BlockUntilStopped(containerId string) {
	ctx, cli := buildCliClient()

	statusCh, errCh := cli.ContainerWait(ctx, containerId, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logrus.Warnf("failed to wait for mock engine container to stop: %v", err)
		}
	case <-statusCh:
	}
}
