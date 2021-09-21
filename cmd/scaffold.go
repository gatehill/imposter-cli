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
	"gatehill.io/imposter/impostermodel"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var flagForceOverwrite bool
var flagGenerateResources bool
var flagScriptEngine string

// scaffoldCmd represents the up command
var scaffoldCmd = &cobra.Command{
	Use:   "scaffold [DIR]",
	Short: "Create Imposter configuration from OpenAPI specs",
	Long: `Creates Imposter configuration from one or more OpenAPI/Swagger specification files
in a directory.

If DIR is not specified, the current working directory is used.`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		var configDir string
		if len(args) == 0 {
			configDir, _ = os.Getwd()
		} else {
			configDir, _ = filepath.Abs(args[0])
		}
		scriptEngine := impostermodel.ParseScriptEngine(flagScriptEngine)
		impostermodel.CreateFromSpecs(configDir, flagGenerateResources, flagForceOverwrite, scriptEngine)
	},
}

func init() {
	scaffoldCmd.Flags().BoolVarP(&flagForceOverwrite, "force-overwrite", "f", false, "Force overwrite of destination file(s) if already exist")
	scaffoldCmd.Flags().BoolVar(&flagGenerateResources, "generate-resources", true, "Generate Imposter resources from OpenAPI paths")
	scaffoldCmd.Flags().StringVarP(&flagScriptEngine, "script-engine", "s", "none", "Generate placeholder Imposter script (none|groovy|js)")
	rootCmd.AddCommand(scaffoldCmd)
}
