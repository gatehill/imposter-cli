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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// versionCmd represents the up command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CLI version",
	Long:  `Prints the version of the CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		println(describeVersions())
	},
}

func describeVersions() string {
	return fmt.Sprintf(`imposter-cli %v
imposter-engine %v`,
		cliconfig.Config.Version,
		cliconfig.GetOrDefaultString(viper.GetString("version"), "latest"),
	)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
