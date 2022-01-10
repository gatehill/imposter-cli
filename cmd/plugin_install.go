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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var pluginInstallFlags = struct {
	flagEngineVersion string
}{}

// pluginInstallCmd represents the pluginInstall command
var pluginInstallCmd = &cobra.Command{
	Use:   "install [PLUGIN_NAME]",
	Short: "Install plugin",
	Long: `Installs the plugin for use with a given engine version.

If version is not specified, it defaults to 'latest'.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version := engine.GetConfiguredVersion(pluginInstallFlags.flagEngineVersion)
		pluginName := args[0]
		installPlugin(pluginName, version)
	},
}

func installPlugin(pluginName string, version string) {
	err := plugin.DownloadPlugin(pluginName, version)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("installed plugin '%s' version %s", pluginName, version)
}

func init() {
	pluginInstallCmd.Flags().StringVarP(&pluginInstallFlags.flagEngineVersion, "version", "v", "", "Imposter engine version (default \"latest\")")
	pluginCmd.AddCommand(pluginInstallCmd)
}
