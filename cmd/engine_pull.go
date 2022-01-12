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
)

var pullFlags = struct {
	flagEngineType    string
	flagEngineVersion string
	flagForcePull     bool
}{}

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull the engine into the cache",
	Long: `Pulls a specified version of the engine binary/image into the cache.

If version is not specified, it defaults to 'latest'.`,
	Run: func(cmd *cobra.Command, args []string) {
		var pullPolicy engine.PullPolicy
		if pullFlags.flagForcePull {
			pullPolicy = engine.PullAlways
		} else {
			pullPolicy = engine.PullIfNotPresent
		}
		version := engine.GetConfiguredVersion(pullFlags.flagEngineVersion, pullPolicy != engine.PullAlways)
		engineType := engine.GetConfiguredType(pullFlags.flagEngineType)
		pull(version, engineType, pullPolicy)
	},
}

func pull(version string, engineType engine.EngineType, pullPolicy engine.PullPolicy) {
	downloader := engine.GetLibrary(engineType).GetProvider(version)
	err := downloader.Provide(pullPolicy)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("pulled engine version %v", version)
}

func init() {
	pullCmd.Flags().StringVarP(&pullFlags.flagEngineType, "engine-type", "t", "", "Imposter engine type (valid: docker,jvm - default \"docker\")")
	pullCmd.Flags().StringVarP(&pullFlags.flagEngineVersion, "version", "v", "", "Imposter engine version (default \"latest\")")
	pullCmd.Flags().BoolVarP(&pullFlags.flagForcePull, "force", "f", false, "Force engine pull")
	engineCmd.AddCommand(pullCmd)
}
