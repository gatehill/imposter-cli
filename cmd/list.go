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
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var listFlags = struct {
	engineType string
}{}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List running mocks",
	Long:  `Lists running Imposter mocks for the current engine type.`,
	Run: func(cmd *cobra.Command, args []string) {
		listMocks(engine.GetConfiguredType(listFlags.engineType))
	},
}

func init() {
	listCmd.Flags().StringVarP(&listFlags.engineType, "engine-type", "t", "", "Imposter engine type (valid: docker,jvm - default \"docker\")")
	rootCmd.AddCommand(listCmd)
}

func listMocks(engineType engine.EngineType) {
	logger.Info("listing all managed mocks...")

	configDir := filepath.Join(os.TempDir(), "imposter-list")
	mockEngine := engine.BuildEngine(engineType, configDir, engine.StartOptions{})

	mocks, err := mockEngine.ListAllManaged()
	if err != nil {
		logger.Fatalf("failed to list mocks: %s", err)
	}

	var rows [][]string
	for _, mock := range mocks {
		rows = append(rows, []string{mock.Name, mock.ID})
	}
	renderMocks(rows)
}

func renderMocks(rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "ID"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.AppendBulk(rows)
	table.Render()
}
