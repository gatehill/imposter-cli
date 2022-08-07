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
	"github.com/spf13/cobra"
)

type outputFormat string

const (
	outputFormatPlain outputFormat = "plain"
	outputFormatJson  outputFormat = "json"
)

var versionFlags = struct {
	engineType string
	format     string
	cliOnly    bool
}{}

// versionCmd represents the up command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints version information",
	Long:  `Prints the version of the CLI and engine, if available.`,
	Run: func(cmd *cobra.Command, args []string) {
		engineType := engine.GetConfiguredType(versionFlags.engineType)
		var format outputFormat
		if versionFlags.format != "" {
			format = outputFormat(versionFlags.format)
		} else {
			format = outputFormatPlain
		}
		println(describeVersions(engineType, format, versionFlags.cliOnly))
	},
}

func init() {
	versionCmd.Flags().StringVarP(&versionFlags.engineType, "engine-type", "t", "", "Imposter engine type (valid: docker,jvm - default \"docker\")")
	versionCmd.Flags().StringVarP(&versionFlags.format, "output-format", "o", "", "Output format (valid: plain,json - default \"plain\")")
	versionCmd.Flags().BoolVar(&versionFlags.cliOnly, "cli-only", false, "Only print the version of the CLI")
	rootCmd.AddCommand(versionCmd)
}

func describeVersions(engineType engine.EngineType, format outputFormat, cliOnly bool) string {
	var firstTrailer string
	if cliOnly {
		firstTrailer = ""
	} else {
		firstTrailer = ","
	}
	output := formatProperty(format, "imposter-cli", config.Config.Version, firstTrailer)

	if !cliOnly {
		engineConfigVersion := engine.GetConfiguredVersion("", true)
		output += formatProperty(format, "imposter-engine", engineConfigVersion, ",")
		output += formatProperty(format, "engine-output", getInstalledEngineVersion(engineType, engineConfigVersion), "")
	}

	switch format {
	case outputFormatPlain:
		return output
	case outputFormatJson:
		return fmt.Sprintf("{\n%s}", output)
	default:
		panic(fmt.Errorf("unsupported output format: %s", format))
	}
}

func formatProperty(format outputFormat, key string, value string, trailer string) string {
	var formatted string
	switch format {
	case outputFormatPlain:
		formatted = fmt.Sprintf("%s %s", key, value)
	case outputFormatJson:
		formatted = fmt.Sprintf(`  "%s": "%s"%s`, key, value, trailer)
	default:
		panic(fmt.Errorf("unsupported output format: %s", format))
	}
	return formatted + "\n"
}

func getInstalledEngineVersion(engineType engine.EngineType, version string) string {
	mockEngine := engine.BuildEngine(engineType, "", engine.StartOptions{
		Version:  version,
		LogLevel: "INFO",
	})
	versionString, err := mockEngine.GetVersionString()
	if err != nil {
		logger.Warn(err)
		return "error"
	}
	return versionString
}
