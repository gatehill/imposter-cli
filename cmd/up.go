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
	"gatehill.io/imposter/config"
	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/fileutil"
	"gatehill.io/imposter/impostermodel"
	"gatehill.io/imposter/plugin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	flagEnablePlugins   bool
	flagEnsurePlugins   bool
	flagEnableFileCache bool
	flagEnvironment     []string
	flagMountDirs       []string
}{}

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [CONFIG_DIR]",
	Short: "Start live mocks of APIs",
	Long: `Starts a live mock of your APIs, using their Imposter configuration.

If CONFIG_DIR is not specified, the current working directory is used.`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		injectExplicitEnvironment()

		var pullPolicy engine.PullPolicy
		if upFlags.flagForcePull {
			pullPolicy = engine.PullAlways
		} else {
			pullPolicy = engine.PullIfNotPresent
		}
		version := engine.GetConfiguredVersion(upFlags.flagEngineVersion, pullPolicy != engine.PullAlways)

		if upFlags.flagEnsurePlugins {
			_, err := plugin.EnsureDefaultPlugins(version)
			if err != nil {
				logrus.Fatal(err)
			}
		}

		var configDir string
		if len(args) == 0 {
			configDir, _ = os.Getwd()
		} else {
			configDir, _ = filepath.Abs(args[0])
		}
		if err := validateConfigExists(configDir, upFlags.flagScaffoldMissing); err != nil {
			logrus.Fatal(err)
		}

		startOptions := engine.StartOptions{
			Port:            upFlags.flagPort,
			Version:         version,
			PullPolicy:      pullPolicy,
			LogLevel:        config.Config.LogLevel,
			ReplaceRunning:  true,
			Deduplicate:     upFlags.flagDeduplicate,
			EnablePlugins:   upFlags.flagEnablePlugins,
			EnableFileCache: upFlags.flagEnableFileCache,
			Environment:     upFlags.flagEnvironment,
			MountDirs:       upFlags.flagMountDirs,
		}
		start(upFlags.flagEngineType, startOptions, configDir, upFlags.flagRestartOnChange)
	},
}

func injectExplicitEnvironment() {
	for _, env := range upFlags.flagEnvironment {
		envParts := strings.Split(env, "=")
		_ = os.Setenv(envParts[0], envParts[1])
	}
}

func init() {
	upCmd.Flags().StringVarP(&upFlags.flagEngineType, "engine-type", "t", "", "Imposter engine type (valid: docker,jvm - default \"docker\")")
	upCmd.Flags().StringVarP(&upFlags.flagEngineVersion, "version", "v", "", "Imposter engine version (default \"latest\")")
	upCmd.Flags().IntVarP(&upFlags.flagPort, "port", "p", 8080, "Port on which to listen")
	upCmd.Flags().BoolVar(&upFlags.flagForcePull, "pull", false, "Force engine pull")
	upCmd.Flags().BoolVar(&upFlags.flagRestartOnChange, "auto-restart", true, "Automatically restart when config dir contents change")
	upCmd.Flags().BoolVarP(&upFlags.flagScaffoldMissing, "scaffold", "s", false, "Scaffold Imposter configuration for all OpenAPI files")
	upCmd.Flags().StringVar(&upFlags.flagDeduplicate, "deduplicate", "", "Override deduplication ID for replacement of containers")
	upCmd.Flags().BoolVar(&upFlags.flagEnablePlugins, "enable-plugins", true, "Whether to enable plugins")
	upCmd.Flags().BoolVar(&upFlags.flagEnsurePlugins, "install-default-plugins", true, "Whether to install missing default plugins")
	upCmd.Flags().BoolVar(&upFlags.flagEnableFileCache, "enable-file-cache", true, "Whether to enable file cache")
	upCmd.Flags().StringArrayVarP(&upFlags.flagEnvironment, "env", "e", []string{}, "Explicit environment variables to set")
	upCmd.Flags().StringArrayVar(&upFlags.flagMountDirs, "mount-dir", []string{}, "(Docker engine type only) Extra directory bind-mounts in the form HOST_PATH:CONTAINER_PATH (e.g. $HOME/somedir:/opt/imposter/somedir) or simply HOST_PATH, which will mount the directory at /opt/imposter/<dir>")
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

	// check for IMPOSTER_CONFIG_SCAN_RECURSIVE
	recursive := viper.GetBool("config.scan.recursive")
	if config.ContainsConfigFile(configDir, recursive) {
		return nil
	}

	if scaffoldMissing {
		logrus.Infof("scaffolding Imposter configuration files")
		impostermodel.CreateFromSpecs(configDir, false, false, impostermodel.ScriptEngineNone)
		return nil
	}
	return fmt.Errorf(`No Imposter configuration files found in: %v
Consider running 'imposter scaffold' first.`, configDir)
}

func start(engineTypeRaw string, startOptions engine.StartOptions, configDir string, restartOnChange bool) {
	engineType := engine.GetConfiguredType(engineTypeRaw)
	mockEngine := engine.BuildEngine(engineType, configDir, startOptions)

	wg := &sync.WaitGroup{}
	trapExit(mockEngine, wg)
	success := mockEngine.Start(wg)

	if success && restartOnChange {
		dirUpdated := fileutil.WatchDir(configDir)
		go func() {
			for {
				<-dirUpdated
				logrus.Infof("detected change in: %v - triggering restart", configDir)
				mockEngine.Restart(wg)
			}
		}()
	}

	wg.Wait()
	logrus.Debug("shutting down")
}

// listen for an interrupt from the OS, then attempt engine cleanup
func trapExit(mockEngine engine.MockEngine, wg *sync.WaitGroup) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		println()
		mockEngine.StopImmediately(wg)
	}()
}
