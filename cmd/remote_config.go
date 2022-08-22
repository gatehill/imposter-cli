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
	"gatehill.io/imposter/remote"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var remoteConfigFlags = struct {
	remoteType string
	token      string
	server     string
}{}

// remoteConfigCmd represents the remoteConfig command
var remoteConfigCmd = &cobra.Command{
	Use:   "config [key=value]",
	Short: "Configure remote",
	Long:  `Configures the remote for the active workspace.`,
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		var dir string
		if remoteFlags.path != "" {
			dir = remoteFlags.path
		} else {
			dir, _ = os.Getwd()
		}

		configured := false
		if remoteConfigFlags.remoteType != "" {
			setRemoteConfigType(dir, remoteConfigFlags.remoteType)
			configured = true
		}

		// convenience flags
		token := cmd.Flag("token")
		if token != nil && token.Changed {
			args = append(args, "token="+remoteConfigFlags.token)
		}
		server := cmd.Flag("server")
		if server != nil && server.Changed {
			args = append(args, "server="+remoteConfigFlags.server)
		}

		if len(args) > 0 {
			for _, arg := range args {
				if !strings.Contains(arg, "=") {
					logger.Warnf("invalid config item: %s", arg)
					continue
				}
				splitArgs := strings.Split(arg, "=")
				setRemoteConfigItem(dir, splitArgs[0], splitArgs[1])
			}
			configured = true
		}

		if !configured {
			printRemoteConfigHelp(cmd, dir)
		}
	},
}

func printRemoteConfigHelp(cmd *cobra.Command, dir string) {
	supported := strings.Join(listSupportedKeys(dir), ", ")
	fmt.Fprintf(os.Stderr, "%v\nSupported config keys: %s\n", cmd.UsageString(), supported)
	os.Exit(1)
}

func init() {
	remoteConfigCmd.Flags().StringVar(&remoteConfigFlags.remoteType, "provider", "", "Set deployment provider")
	remoteConfigCmd.Flags().StringVarP(&remoteConfigFlags.token, "token", "t", "", "Set deployment token")
	remoteConfigCmd.Flags().StringVarP(&remoteConfigFlags.server, "server", "s", "", "Set deployment server URL")
	remoteCmd.AddCommand(remoteConfigCmd)
}

func setRemoteConfigType(dir string, remoteType string) {
	active, err := remote.SaveActiveRemoteType(dir, remoteType)
	if err != nil {
		logger.Fatalf("failed to set remote type: %s", err)
	}
	logger.Infof("set remote type to '%s' for remote: %s", remoteType, active.Name)
}

func setRemoteConfigItem(dir string, key string, value string) {
	active, r, err := remote.LoadActive(dir)
	if err != nil {
		logger.Fatalf("failed to load remote: %s", err)
	}
	err = (*r).SetConfigValue(key, value)
	if err != nil {
		logger.Fatalf("failed to set remote %s: %s", key, err)
	}
	logger.Infof("set %s for remote: %s", key, active.Name)
}

func listSupportedKeys(dir string) []string {
	_, r, err := remote.LoadActive(dir)
	if err != nil {
		logger.Fatalf("failed to load remote: %s", err)
	}
	return (*r).GetConfigKeys()
}
