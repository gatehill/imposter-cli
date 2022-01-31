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
	"gatehill.io/imposter/plugin"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const containerConfigDir = "/opt/imposter/config"
const containerPluginDir = "/opt/imposter/plugins"
const containerFileCacheDir = "/tmp/imposter-cache"
const removalTimeoutSec = 5

func (d *DockerMockEngine) Start(wg *sync.WaitGroup) bool {
	return d.startWithOptions(wg, d.options)
}

func (d *DockerMockEngine) startWithOptions(wg *sync.WaitGroup, options engine.StartOptions) (success bool) {
	logrus.Infof("starting mock engine on port %d - press ctrl+c to stop", options.Port)
	ctx, cli, err := BuildCliClient()
	if err != nil {
		logrus.Fatal(err)
	}

	if !d.provider.Satisfied() {
		if err := d.provider.Provide(engine.PullIfNotPresent); err != nil {
			logrus.Fatal(err)
		}
	}

	mockHash, containerLabels := generateMetadata(d, options)

	if options.ReplaceRunning {
		stopDuplicateContainers(d, cli, ctx, mockHash)
	}

	containerPort := nat.Port(fmt.Sprintf("%d/tcp", options.Port))
	hostPort := fmt.Sprintf("%d", options.Port)

	// if not specified, falls back to default in container image
	containerUser := viper.GetString("docker.containerUser")
	logrus.Tracef("container user: %s", containerUser)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: d.provider.imageAndTag,
		Cmd: []string{
			"--configDir=" + containerConfigDir,
			fmt.Sprintf("--listenPort=%d", options.Port),
		},
		Env: buildEnv(options),
		ExposedPorts: nat.PortSet{
			containerPort: {},
		},
		Labels: containerLabels,
		User:   containerUser,
	}, &container.HostConfig{
		Binds: buildBinds(d, options),
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
		logrus.Fatal(err)
	}

	containerId := resp.ID
	d.debouncer.Register(wg, containerId)
	if err := cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		logrus.Fatalf("error starting mock engine container: %v", err)
	}
	logrus.Trace("starting Docker mock engine")

	d.containerId = containerId
	if err = streamLogsToStdIo(cli, ctx, containerId); err != nil {
		logrus.Warn(err)
	}
	up := engine.WaitUntilUp(options.Port, d.shutDownC)

	// watch in case container stops
	go func() {
		notifyOnStopBlocking(d, wg, containerId, cli, ctx)
	}()

	return up
}

func buildEnv(options engine.StartOptions) []string {
	env := engine.BuildEnv(options, false)
	if options.EnableFileCache {
		env = append(env, "IMPOSTER_CACHE_DIR=/tmp/imposter-cache", "IMPOSTER_OPENAPI_REMOTE_FILE_CACHE=true")
	}
	logrus.Tracef("engine environment: %v", env)
	return env
}

func buildBinds(d *DockerMockEngine, options engine.StartOptions) []string {
	binds := []string{
		d.configDir + ":" + containerConfigDir + viper.GetString("docker.bindFlags"),
	}
	if options.EnablePlugins {
		logrus.Tracef("plugins are enabled")
		pluginDir, err := plugin.EnsurePluginDir(options.Version)
		if err != nil {
			logrus.Fatal(err)
		}
		binds = append(binds, pluginDir+":"+containerPluginDir)
	}
	if options.EnableFileCache {
		logrus.Tracef("file cache enabled")
		fileCacheDir, err := engine.EnsureFileCacheDir()
		if err != nil {
			logrus.Fatal(err)
		}
		binds = append(binds, fileCacheDir+":"+containerFileCacheDir)
	}
	binds = append(binds, options.BindMounts...)
	logrus.Tracef("using binds: %v", binds)
	return binds
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

func streamLogsToStdIo(cli *client.Client, ctx context.Context, containerId string) error {
	return streamLogs(cli, ctx, containerId, os.Stdout, os.Stderr)
}

func streamLogs(cli *client.Client, ctx context.Context, containerId string, outStream io.Writer, errStream io.Writer) error {
	containerLogs, err := cli.ContainerLogs(ctx, containerId, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
	})
	if err != nil {
		return fmt.Errorf("error streaming container logs for container with ID: %v: %v", containerId, err)
	}
	go func() {
		_, err := stdcopy.StdCopy(outStream, errStream, containerLogs)
		if err != nil {
			logrus.Warnf("error streaming container logs for container with ID: %v: %v", containerId, err)
		}
	}()
	return nil
}

func BuildCliClient() (context.Context, *client.Client, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, nil, err
	}
	return ctx, cli, nil
}

func (d *DockerMockEngine) StopImmediately(wg *sync.WaitGroup) {
	go func() { d.shutDownC <- true }()
	d.Stop(wg)
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

func (d *DockerMockEngine) StopAllManaged() int {
	cli, ctx, err := BuildCliClient()
	if err != nil {
		logrus.Fatal(err)
	}

	labels := map[string]string{
		labelKeyManaged: "true",
	}
	return stopContainersWithLabels(d, ctx, cli, labels)
}

func (d *DockerMockEngine) GetVersionString() (string, error) {
	if !d.provider.Satisfied() {
		if err := d.provider.Provide(engine.PullIfNotPresent); err != nil {
			return "", err
		}
	}

	output := new(strings.Builder)
	errOutput := new(strings.Builder)

	ctx, cli, err := BuildCliClient()
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: d.provider.imageAndTag,
		Cmd: []string{
			"--version",
		},
	}, &container.HostConfig{}, nil, nil, "")
	if err != nil {
		return "", err
	}
	containerId := resp.ID

	wg := &sync.WaitGroup{}
	d.debouncer.Register(wg, containerId)
	if err := cli.ContainerStart(ctx, containerId, types.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("error starting mock engine container: %v", err)
	}
	if err = streamLogs(cli, ctx, containerId, output, errOutput); err != nil {
		return "", fmt.Errorf("error getting mock engine output: %v", err)
	}
	notifyOnStopBlocking(d, wg, containerId, cli, ctx)
	return engine.SanitiseVersionOutput(output.String()), nil
}
