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
package cmd

import (
	"context"
	"fmt"
	"gatehill.io/imposter/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const EngineDockerImage = "outofcoffee/imposter"
const ContainerConfigDir = "/opt/imposter/config"

var ImageTag string
var Port string

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [CONFIG_DIR]",
	Short: "Start live mocks of APIs",
	Long:  `Starts a live mock of your APIs, using their Imposter configuration.`,
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		var configDir string
		if len(args) == 0 {
			configDir, _ = os.Getwd()
		} else {
			configDir = args[0]
		}
		port, err := strconv.Atoi(Port)
		if err != nil {
			panic(fmt.Errorf("invalid port: %v", Port))
		}
		startMockEngine(configDir, port)
	},
}

func init() {
	upCmd.Flags().StringVarP(&ImageTag, "version", "v", "latest", "Imposter engine version")
	upCmd.Flags().StringVarP(&Port, "port", "p", "8080", "Port on which to listen")
	rootCmd.AddCommand(upCmd)
}

func startMockEngine(configDir string, port int) {
	logrus.Infof("starting mock engine on port %d", port)

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	imageAndTag := EngineDockerImage + ":" + ImageTag
	logrus.Infof("checking '%v' image", ImageTag)
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

	containerPort := nat.Port(fmt.Sprintf("%d/tcp", port))
	hostPort := fmt.Sprintf("%d", port)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageAndTag,
		Cmd: []string{
			"--configDir=" + ContainerConfigDir,
			fmt.Sprintf("--listenPort=%d", port),
		},
		Env: []string{
			"IMPOSTER_LOG_LEVEL=" + strings.ToUpper(util.LogLevel),
		},
		ExposedPorts: nat.PortSet{
			containerPort: {},
		},
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: configDir,
				Target: ContainerConfigDir,
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

	trapExit(cli, ctx, resp.ID)
	logrus.Info("mock engine started - press ctrl+c to stop")

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
	})
	if err != nil {
		panic(err)
	}

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	if err != nil {
		panic(err)
	}
}

func stopMockEngine(cli *client.Client, ctx context.Context, containerID string) {
	logrus.Info("\rstopping mock engine...\n")
	err := cli.ContainerStop(ctx, containerID, nil)
	if err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logrus.Warnf("failed to stop mock engine: %v", err)
		}
	case <-statusCh:
	}
	logrus.Trace("mock engine stopped")

	err = cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
	if err != nil {
		logrus.Warnf("failed to remove mock engine container: %v", err)
	}
	logrus.Trace("mock engine container removed")
}

// listen for an interrupt from the OS, then attempt engine cleanup
func trapExit(cli *client.Client, ctx context.Context, containerID string) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		stopMockEngine(cli, ctx, containerID)
		os.Exit(0)
	}()
}
