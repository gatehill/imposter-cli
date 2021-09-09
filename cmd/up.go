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
	"gatehill.io/imposter/cliconfig"
	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/fileutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var flagImageTag string
var flagPort int
var flagForcePull bool
var flagRestartOnChange bool

var containerId string
var stopCh chan string
var restartsPending int

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [CONFIG_DIR]",
	Short: "Start live mocks of APIs",
	Long: `Starts a live mock of your APIs, using their Imposter configuration.

If CONFIG_DIR is not specified, the current working directory is used.`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		trapExit()

		var configDir string
		if len(args) == 0 {
			configDir, _ = os.Getwd()
		} else {
			configDir, _ = filepath.Abs(args[0])
		}

		var imagePullPolicy engine.ImagePullPolicy
		if flagForcePull {
			imagePullPolicy = engine.ImagePullAlways
		} else {
			imagePullPolicy = engine.ImagePullIfNotPresent
		}
		startOptions := engine.StartOptions{
			Port:            flagPort,
			ImageTag:        flagImageTag,
			ImagePullPolicy: imagePullPolicy,
			LogLevel:        cliconfig.Config.LogLevel,
		}
		startControlLoop(configDir, startOptions)
	},
}

func init() {
	upCmd.Flags().StringVarP(&flagImageTag, "version", "v", "latest", "Imposter engine version")
	upCmd.Flags().IntVarP(&flagPort, "port", "p", 8080, "Port on which to listen")
	upCmd.Flags().BoolVar(&flagForcePull, "pull", false, "Force engine image pull")
	upCmd.Flags().BoolVar(&flagRestartOnChange, "auto-restart", true, "Automatically restart when config dir contents change")
	rootCmd.AddCommand(upCmd)
}

// listen for an interrupt from the OS, then attempt engine cleanup
func trapExit() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		println()
		if len(containerId) > 0 {
			engine.TriggerRemovalAndNotify(containerId, stopCh)
		} else {
			os.Exit(0)
		}
	}()
}

func startControlLoop(configDir string, startOptions engine.StartOptions) {
	stopCh = make(chan string)
	containerId = engine.StartMockEngine(configDir, startOptions)

	var dirUpdated chan bool
	if flagRestartOnChange {
		dirUpdated = fileutil.WatchDir(configDir)
	}

control:
	for {
		engine.NotifyOnStop(containerId, stopCh)

		select {
		case <-dirUpdated:
			logrus.Infof("detected change in: %v - triggering restart", configDir)
			containerId = restart(configDir, startOptions, containerId)
			break

		case stoppedContainerId := <-stopCh:
			// check ID to debounce events
			if stoppedContainerId == containerId {
				if restartsPending > 0 {
					restartsPending--
				} else {
					break control
				}
			}
			break
		}
	}

	logrus.Debug("shutting down")
}

func restart(configDir string, options engine.StartOptions, existingContainerId string) (containerId string) {
	restartsPending++
	engine.StopMockEngine(existingContainerId)

	// don't pull again
	options.ImagePullPolicy = engine.ImagePullSkip

	newContainerId := engine.StartMockEngine(configDir, options)
	return newContainerId
}
