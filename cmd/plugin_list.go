/*
Copyright © 2023 Pete Cornish <outofcoffee@gmail.com>

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
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"os"
)

var pluginListFlags = struct {
	engineVersion string
}{}

// pluginListCmd represents the pluginList command
var pluginListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List the plugins in the cache",
	Long:    `Lists all versions of plugins installed.`,
	Run: func(cmd *cobra.Command, args []string) {
		var version = engine.GetConfiguredVersion(pluginListFlags.engineVersion, true)
		listPlugins([]string{version})
	},
}

func listPlugins(versions []string) {
	logger.Tracef("listing plugins")
	var available []plugin.PluginMetadata

	for _, version := range versions {
		plugins, err := plugin.List(version)
		if err != nil {
			logger.Fatal(err)
		}
		available = append(available, plugins...)
	}

	var rows [][]string
	for _, metadata := range available {
		rows = append(rows, []string{metadata.Name, metadata.Version})
	}
	renderPlugins(rows)
}

func renderPlugins(rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Type", "Version"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk(rows)
	table.Render()
}

func init() {
	pluginListCmd.Flags().StringVarP(&pluginListFlags.engineVersion, "version", "v", "", "Imposter engine version (default \"latest\")")
	pluginCmd.AddCommand(pluginListCmd)
}
