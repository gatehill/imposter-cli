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
	"fmt"
	"gatehill.io/imposter/cliconfig"
	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/fileutil"
	"gatehill.io/imposter/impostermodel"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
)

var upFlags = struct {
	flagDeduplicate     string
	flagEngineType      string
	flagEngineVersion   string
	flagForcePull       bool
	flagPort            int
	flagRestartOnChange bool
	flagScaffoldMissing bool
}{}

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
		if err := validateConfigExists(configDir, upFlags.flagScaffoldMissing); err != nil {
			logrus.Fatal(err)
		}

		var pullPolicy engine.PullPolicy
		if upFlags.flagForcePull {
			pullPolicy = engine.PullAlways
		} else {
			pullPolicy = engine.PullIfNotPresent
		}
		startOptions := engine.StartOptions{
			Port:           upFlags.flagPort,
			Version:        engine.GetConfiguredVersion(upFlags.flagEngineVersion),
			PullPolicy:     pullPolicy,
			LogLevel:       cliconfig.Config.LogLevel,
			ReplaceRunning: true,
			Deduplicate:    upFlags.flagDeduplicate,
		}
		mockEngine := engine.BuildEngine(engine.GetConfiguredType(upFlags.flagEngineType), configDir, startOptions)

		wg := &sync.WaitGroup{}
		trapExit(mockEngine, wg)
		start(mockEngine, wg, configDir, upFlags.flagRestartOnChange)

		wg.Wait()
		logrus.Debug("shutting down")
	},
}

func init() {
	upCmd.Flags().StringVarP(&upFlags.flagEngineType, "engine", "e", "", "Imposter engine type (valid: docker,jvm - default \"docker\")")
	upCmd.Flags().StringVarP(&upFlags.flagEngineVersion, "version", "v", "", "Imposter engine version (default \"latest\")")
	upCmd.Flags().IntVarP(&upFlags.flagPort, "port", "p", 8080, "Port on which to listen")
	upCmd.Flags().BoolVar(&upFlags.flagForcePull, "pull", false, "Force engine pull")
	upCmd.Flags().BoolVar(&upFlags.flagRestartOnChange, "auto-restart", true, "Automatically restart when config dir contents change")
	upCmd.Flags().BoolVarP(&upFlags.flagScaffoldMissing, "scaffold", "s", false, "Scaffold Imposter configuration for all OpenAPI files")
	upCmd.Flags().StringVar(&upFlags.flagDeduplicate, "deduplicate", "", "Override deduplication ID for replacement of containers")
	rootCmd.AddCommand(upCmd)
}

func validateConfigExists(configDir string, scaffoldMissing bool) error {
	fileInfo, err := os.Stat(configDir)
	if err != nil {
		return fmt.Errorf("cannot find config dir: %v", err)
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("path is not a directory: %v", configDir)
	}
	files, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("unable to list directory contents: %v: %v", configDir, err)
	}

	configFileFound := false
	for _, file := range files {
		configFileFound = cliconfig.MatchesConfigFileFmt(file)
		if configFileFound {
			return nil
		}
	}
	if scaffoldMissing {
		logrus.Infof("scaffolding Imposter configuration files")
		impostermodel.CreateFromSpecs(configDir, false, false, impostermodel.ScriptEngineNone)
		return nil
	}
	return fmt.Errorf(`No Imposter configuration files found in: %v
Consider running 'imposter scaffold' first.`, configDir)
}

// listen for an interrupt from the OS, then attempt engine cleanup
func trapExit(mockEngine engine.MockEngine, wg *sync.WaitGroup) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		println()
		mockEngine.Stop(wg)
	}()
}

func start(mockEngine engine.MockEngine, wg *sync.WaitGroup, configDir string, restartOnChange bool) {
	mockEngine.Start(wg)

	if restartOnChange {
		dirUpdated := fileutil.WatchDir(configDir)
		go func() {
			for {
				<-dirUpdated
				logrus.Infof("detected change in: %v - triggering restart", configDir)
				mockEngine.Restart(wg)
			}
		}()
	}
}
