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
	"github.com/spf13/cobra"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

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

Learn more at github.com/outofcoffee/imposter`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.imposter.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".imposter" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".imposter")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
