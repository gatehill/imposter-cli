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
	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/util"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var flagImageTag string
var flagPort int
var flagForcePull bool

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [CONFIG_DIR]",
	Short: "Start live mocks of APIs",
	Long: `Starts a live mock of your APIs, using their Imposter configuration.

If CONFIG_DIR is not specified, the current working directory is used.`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		var configDir string
		if len(args) == 0 {
			configDir, _ = os.Getwd()
		} else {
			configDir, _ = filepath.Abs(args[0])
		}
		containerId := engine.StartMockEngine(configDir, engine.EngineStartOptions{
			Port:           flagPort,
			ImageTag:       flagImageTag,
			ForceImagePull: flagForcePull,
			LogLevel:       util.Config.LogLevel,
		})
		trapExit(containerId)
		engine.BlockUntilStopped(containerId)
	},
}

func init() {
	upCmd.Flags().StringVarP(&flagImageTag, "version", "v", "latest", "Imposter engine version")
	upCmd.Flags().IntVarP(&flagPort, "port", "p", 8080, "Port on which to listen")
	upCmd.Flags().BoolVar(&flagForcePull, "pull", false, "Force engine image pull")
	rootCmd.AddCommand(upCmd)
}

// listen for an interrupt from the OS, then attempt engine cleanup
func trapExit(containerID string) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		engine.StopMockEngine(containerID)
		os.Exit(0)
	}()
}
