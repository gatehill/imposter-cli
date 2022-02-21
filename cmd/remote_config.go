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

var remoteConfigFlags = struct {
	remoteType string
	token      string
	url        string
}{}

// remoteConfigCmd represents the remoteConfig command
var remoteConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure remote",
	Long:  `Configures the remote for the active workspace.`,
	Run: func(cmd *cobra.Command, args []string) {
		var dir string
		if remoteFlags.path != "" {
			dir = remoteFlags.path
		} else {
			dir, _ = os.Getwd()
		}

		configured := false
		token := cmd.Flag("token")
		if token != nil && token.Changed {
			setRemoteConfigToken(dir, remoteConfigFlags.token)
			configured = true
		}
		if remoteConfigFlags.remoteType != "" {
			setRemoteConfigType(dir, remoteConfigFlags.remoteType)
			configured = true
		}
		url := cmd.Flag("server")
		if url != nil && url.Changed {
			setRemoteConfigUrl(dir, remoteConfigFlags.url)
			configured = true
		}

		if !configured {
			cobra.CheckErr(cmd.Help())
		}
	},
}

func init() {
	remoteConfigCmd.Flags().StringVar(&remoteConfigFlags.remoteType, "provider", "", "Set deployment provider")
	remoteConfigCmd.Flags().StringVarP(&remoteConfigFlags.token, "token", "t", "", "Set deployment token")
	remoteConfigCmd.Flags().StringVarP(&remoteConfigFlags.url, "server", "s", "", "Set deployment server URL")
	remoteCmd.AddCommand(remoteConfigCmd)
}

func setRemoteConfigType(dir string, remoteType string) {
	active, err := remote.SaveActiveRemoteType(dir, remoteType)
	if err != nil {
		logrus.Fatalf("failed to set remote type: %s", err)
	}
	logrus.Infof("set remote type to '%s' for remote: %s", remoteType, active.Name)
}

func setRemoteConfigUrl(dir string, url string) {
	active, r, err := remote.LoadActive(dir)
	if err != nil {
		logrus.Fatalf("failed to load remote: %s", err)
	}
	err = (*r).SetUrl(url)
	if err != nil {
		logrus.Fatalf("failed to set remote URL: %s", err)
	}
	logrus.Infof("set remote URL to '%s' for remote: %s", url, active.Name)
}

func setRemoteConfigToken(dir string, token string) {
	active, r, err := remote.LoadActive(dir)
	if err != nil {
		logrus.Fatalf("failed to load remote: %s", err)
	}
	err = (*r).SetToken(token)
	if err != nil {
		logrus.Fatalf("failed to set remote token: %s", err)
	}
	logrus.Infof("set remote token for remote: %s", active.Name)
}
