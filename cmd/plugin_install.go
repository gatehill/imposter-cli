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
	"gatehill.io/imposter/plugin"
	"github.com/spf13/cobra"
)

var pluginInstallFlags = struct {
	engineVersion string
	saveDefault   bool
}{}

// pluginInstallCmd represents the pluginInstall command
var pluginInstallCmd = &cobra.Command{
	Use:   "install [PLUGIN_NAME_1] [PLUGIN_NAME_N...]",
	Short: "Install plugins",
	Long: `Installs plugins for a specific engine version.

If version is not specified, it defaults to 'latest'.

Example 1: Install named plugin

	imposter plugin install store-redis

Example 2: Install all plugins in config file

	imposter plugin install`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		version := engine.GetConfiguredVersion(pluginInstallFlags.engineVersion, true)
		installPlugins(args, version, pluginInstallFlags.saveDefault)
	},
}

func init() {
	pluginInstallCmd.Flags().StringVarP(&pluginInstallFlags.engineVersion, "version", "v", "", "Imposter engine version (default \"latest\")")
	pluginInstallCmd.Flags().BoolVarP(&pluginInstallFlags.saveDefault, "save-default", "d", false, "Whether to save the plugin as a default")
	pluginCmd.AddCommand(pluginInstallCmd)
}

func installPlugins(plugins []string, version string, saveDefault bool) {
	var ensured int
	var err error
	if len(plugins) == 0 {
		ensured, err = plugin.EnsureDefaultPlugins(version)
	} else {
		ensured, err = plugin.EnsurePlugins(plugins, version, saveDefault)
	}
	if err != nil {
		logger.Fatal(err)
	}
	if ensured == 0 {
		logger.Infof("no plugins to install")
	} else {
		logger.Infof("%d plugin(s) installed", ensured)
	}
}
