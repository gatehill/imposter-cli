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
	"path/filepath"
	"strconv"
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
	provider    *EngineImageProvider
	containerId string
	debouncer   debounce.Debouncer
}

func init() {
	engine.RegisterProvider("docker", func(version string) engine.Provider {
		return GetProvider(version)
	})
	engine.RegisterEngine("docker", func(configDir string, startOptions engine.StartOptions) engine.MockEngine {
		return BuildEngine(configDir, startOptions)
	})
}

func BuildEngine(configDir string, options engine.StartOptions) engine.MockEngine {
	return &DockerMockEngine{
		configDir: configDir,
		options:   options,
		provider:  GetProvider(options.Version),
		debouncer: debounce.Build(),
	}
}

func (d *DockerMockEngine) Start(wg *sync.WaitGroup) {
	d.startWithOptions(wg, d.options)
}

func (d *DockerMockEngine) startWithOptions(wg *sync.WaitGroup, options engine.StartOptions) {
	logrus.Infof("starting mock engine on port %d", options.Port)
	ctx, cli, err := BuildCliClient()
	if err != nil {
		logrus.Fatal(err)
	}

	if !d.provider.Satisfied() {
		err := d.provider.Provide(options.PullPolicy)
		if err != nil {
			logrus.Fatal(err)
		}
	}

	mockHash, containerLabels := generateMetadata(d, options)

	if options.ReplaceRunning {
		stopDuplicateContainers(d, cli, ctx, mockHash)
	}

	containerPort := nat.Port(fmt.Sprintf("%d/tcp", options.Port))
	hostPort := fmt.Sprintf("%d", options.Port)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: d.provider.imageAndTag,
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
		Labels: containerLabels,
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

	containerId := resp.ID
	d.debouncer.Register(wg, containerId)
	if err := cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		logrus.Fatalf("error starting mock engine container: %v", err)
	}
	logrus.Info("mock engine started - press ctrl+c to stop")

	d.containerId = containerId
	streamLogs(err, cli, ctx, containerId)
	engine.WaitUntilUp(options.Port)

	// watch in case container stops
	go func() {
		notifyOnStopBlocking(d, wg, containerId, cli, ctx)
	}()
}

func generateMetadata(d *DockerMockEngine, options engine.StartOptions) (string, map[string]string) {
	absoluteConfigDir, _ := filepath.Abs(d.configDir)

	var mockHash string
	if options.Deduplicate != "" {
		mockHash = sha1hash(options.Deduplicate)
	} else {
		mockHash = genDefaultHash(absoluteConfigDir, options.Port)
	}

	containerLabels := map[string]string{
		labelKeyManaged: "true",
		labelKeyDir:     absoluteConfigDir,
		labelKeyPort:    strconv.Itoa(options.Port),
		labelKeyHash:    mockHash,
	}
	return mockHash, containerLabels
}

func streamLogs(err error, cli *client.Client, ctx context.Context, containerId string) {
	containerLogs, err := cli.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{
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
}

func BuildCliClient() (context.Context, *client.Client, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, nil, err
	}
	return ctx, cli, nil
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

	removeContainer(d, wg, oldContainerId)
}

func (d *DockerMockEngine) Restart(wg *sync.WaitGroup) {
	wg.Add(1)
	d.Stop(wg)

	// don't pull again
	restartOptions := d.options
	restartOptions.PullPolicy = engine.PullSkip

	d.startWithOptions(wg, restartOptions)
	wg.Done()
}

func (d *DockerMockEngine) StopAllManaged() {
	logrus.Info("stopping all managed mocks...")
	cli, ctx, err := BuildCliClient()
	if err != nil {
		logrus.Fatal(err)
	}

	labels := map[string]string{
		labelKeyManaged: "true",
	}
	if stopped := stopContainersWithLabels(d, ctx, cli, labels); stopped > 0 {
		logrus.Infof("stopped %d managed mocks", stopped)
	} else {
		logrus.Info("no managed mocks were found")
	}
}
