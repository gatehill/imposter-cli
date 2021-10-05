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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

var downFlags = struct {
	flagDeduplicate string
	flagEngineType  string
}{}

// downCmd represents the down command
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop running mocks",
	Long:  `Stops running Imposter mocks for the current engine type.`,
	Run: func(cmd *cobra.Command, args []string) {
		version := cliconfig.GetFirstNonEmpty(viper.GetString("version"), "latest")
		stopAll(engine.EngineType(downFlags.flagEngineType), version, downFlags.flagDeduplicate)
	},
}

func init() {
	downCmd.Flags().StringVarP(&downFlags.flagEngineType, "engine", "e", "", "Imposter engine type (valid: docker - default \"docker\")")
	downCmd.Flags().StringVar(&downFlags.flagDeduplicate, "deduplicate", "", "Override deduplication ID for replacement of containers")
	rootCmd.AddCommand(downCmd)
}

func stopAll(engineType engine.EngineType, version string, deduplicate string) {
	configDir := filepath.Join(os.TempDir(), "imposter-tidy")
	mockEngine := engine.BuildEngine(engineType, configDir, engine.StartOptions{
		Version:     version,
		LogLevel:    cliconfig.Config.LogLevel,
		Deduplicate: deduplicate,
	})
	mockEngine.StopAllManaged()
}
