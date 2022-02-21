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
	"gatehill.io/imposter/workspace"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

// workspaceDeleteCmd represents the workspaceDelete command
var workspaceDeleteCmd = &cobra.Command{
	Use:   "delete [WORKSPACE_NAME]",
	Short: "Delete a workspace",
	Long:  `Deletes a workspace, if it exists.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		deleteWorkspace(name)
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceDeleteCmd)
}

func deleteWorkspace(name string) {
	wd, _ := os.Getwd()
	err := workspace.Delete(wd, name)
	if err != nil {
		logrus.Fatalf("failed to delete workspace: %s", err)
	}
	logrus.Infof("deleted workspace '%s'", name)
}
