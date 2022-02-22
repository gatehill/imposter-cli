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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

// remoteDeployCmd represents the remoteDeploy command
var remoteDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy active workspace",
	Long:  `Deploys the active workspace to the remote.`,
	Run: func(cmd *cobra.Command, args []string) {
		var dir string
		if remoteFlags.path != "" {
			dir = remoteFlags.path
		} else {
			dir, _ = os.Getwd()
		}
		remoteDeploy(dir)
	},
}

func init() {
	remoteCmd.AddCommand(remoteDeployCmd)
}

func remoteDeploy(dir string) {
	active, r, err := remote.LoadActive(dir)
	if err != nil {
		logrus.Fatalf("failed to load remote: %s", err)
	}
	logrus.Infof("deploying workspace '%s' to remote: %s", active.Name, (*r).GetUrl())

	endpoint, err := (*r).Deploy()
	if err != nil {
		logrus.Fatalf("failed to deploy workspace: %s", err)
	}
	logrus.Infof("deployed workspace '%s'\nBase URL: %s\nSpec: %s\nStatus: %s", active.Name, endpoint.BaseUrl, endpoint.SpecUrl, endpoint.StatusUrl)
}
