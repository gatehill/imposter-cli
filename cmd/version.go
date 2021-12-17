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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var versionFlags = struct {
	flagEngineType string
}{}

// versionCmd represents the up command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints version information",
	Long:  `Prints the version of the CLI and engine, if available.`,
	Run: func(cmd *cobra.Command, args []string) {
		engineType := engine.GetConfiguredType(versionFlags.flagEngineType)
		println(describeVersions(engineType))
	},
}

func init() {
	versionCmd.Flags().StringVarP(&versionFlags.flagEngineType, "engine", "e", "", "Imposter engine type (valid: docker,jvm - default \"docker\")")
	rootCmd.AddCommand(versionCmd)
}

func describeVersions(engineType engine.EngineType) string {
	engineConfigVersion := engine.GetConfiguredVersion("")
	engineVersionOutput := getInstalledEngineVersion(engineType)

	return fmt.Sprintf(`imposter-cli %[1]v
imposter-engine %[2]v
engine-output %[3]v`,
		cliconfig.Config.Version,
		engineConfigVersion,
		engineVersionOutput,
	)
}

func getInstalledEngineVersion(engineType engine.EngineType) string {
	mockEngine := engine.BuildEngine(engineType, "", engine.StartOptions{
		Version:  engine.GetConfiguredVersion(""),
		LogLevel: "INFO",
	})
	versionString, err := mockEngine.GetVersionString()
	if err != nil {
		logrus.Warn(err)
		return "error"
	}
	return versionString
}
