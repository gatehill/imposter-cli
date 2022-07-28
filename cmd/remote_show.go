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
	"gatehill.io/imposter/remote"
	"github.com/spf13/cobra"
	"os"
)

// remoteShowCmd represents the remoteShow command
var remoteShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show remote",
	Long:  `Shows the remote for the active workspace.`,
	Run: func(cmd *cobra.Command, args []string) {
		var dir string
		if remoteFlags.path != "" {
			dir = remoteFlags.path
		} else {
			dir, _ = os.Getwd()
		}
		showRemote(dir)
	},
}

func init() {
	remoteCmd.AddCommand(remoteShowCmd)
}

func showRemote(dir string) {
	active, r, err := remote.LoadActive(dir)
	if err != nil {
		logger.Fatalf("failed to load remote: %s", err)
	}

	remoteType := (*r).GetType()
	url := (*r).GetUrl()
	token, err := (*r).GetObfuscatedToken()
	if err != nil {
		logger.Fatalf("failed to get remote token: %s", err)
	}

	logger.Infof(`Workspace '%s' remote:
  Type: %s
  URL: %s
  Token: %s`, active.Name, remoteType, url, token)
}
