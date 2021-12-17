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
	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

var rootFlags = struct {
	cfgFile          string
	flagPrintVersion bool
}{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "imposter",
	Short: "Imposter mock engine CLI",
	Long: `Imposter is a scriptable, multipurpose mock server.
Use Imposter to:

* run standalone mocks in place of real systems
* turn an OpenAPI/Swagger file into a mock API for testing or QA
* quickly set up a temporary API for your mobile/web client teams whilst the real API is being built
* decouple your integration tests from the cloud/various back-end systems and take control of your dependencies
* validate your API requests against an OpenAPI specification

Provide mock responses using static files or customise behaviour based on characteristics of the request.
Capture data and use response templates to provide conditional responses.

Power users can control mock responses with JavaScript or Java/Groovy script engines.
Advanced users can write their own plugins in a JVM language of their choice.

Learn more at www.imposter.sh`,
	Run: func(cmd *cobra.Command, args []string) {
		if rootFlags.flagPrintVersion {
			engineType := engine.GetConfiguredType(versionFlags.flagEngineType)
			println(describeVersions(engineType))
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// syntactic sugar to support common `<app> --version` usage
	rootCmd.Flags().BoolVar(&rootFlags.flagPrintVersion, "version", false, "Print version information")

	// Global flags.
	rootCmd.PersistentFlags().StringVar(&rootFlags.cfgFile, "config", "", "config file (default is $HOME/.imposter/config.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if rootFlags.cfgFile != "" {
		viper.SetConfigFile(rootFlags.cfgFile)
	} else {
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		configDir := filepath.Join(home, ".imposter")
		if _, err := os.Stat(configDir); err == nil {
			// Search files in config directory with name "config" (without extension).
			viper.AddConfigPath(configDir)
			viper.SetConfigName("config")
		}
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("IMPOSTER")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logrus.Tracef("using CLI config file: %v", viper.ConfigFileUsed())
	}
}
