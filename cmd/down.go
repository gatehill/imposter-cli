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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var downFlags = struct {
	flagEngineType string
}{}

// downCmd represents the down command
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop running mocks",
	Long:  `Stops running Imposter mocks for the current engine type.`,
	Run: func(cmd *cobra.Command, args []string) {
		stopAll(engine.GetConfiguredType(downFlags.flagEngineType))
	},
}

func init() {
	downCmd.Flags().StringVarP(&downFlags.flagEngineType, "engine-type", "t", "", "Imposter engine type (valid: docker,jvm - default \"docker\")")
	rootCmd.AddCommand(downCmd)
}

func stopAll(engineType engine.EngineType) {
	logrus.Info("stopping all managed mocks...")

	configDir := filepath.Join(os.TempDir(), "imposter-down")
	mockEngine := engine.BuildEngine(engineType, configDir, engine.StartOptions{})

	if stopped := mockEngine.StopAllManaged(); stopped > 0 {
		logrus.Infof("stopped %d managed mock(s)", stopped)
	} else {
		logrus.Info("no managed mocks were found")
	}
}
