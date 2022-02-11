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

package config

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

type CliConfig struct {
	Version  string
	LogLevel string
}

// The ConfigFileName is the file name without the file extension.
const ConfigFileName = "config"

var (
	Config  CliConfig
	DirPath string
)

func init() {
	Config = CliConfig{
		Version:  "dev",
		LogLevel: "DEBUG",
	}
}

func SetLogLevel(lvl string) {
	ll, err := logrus.ParseLevel(lvl)
	if err != nil {
		ll = logrus.DebugLevel
	}
	logrus.SetLevel(ll)
}

func GetConfigDir() (string, error) {
	if DirPath != "" {
		return DirPath, nil
	}
	return getDefaultConfigDir()
}

func getDefaultConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user home directory: %s", err)
	}
	configFile := filepath.Join(homeDir, ".imposter")
	return configFile, nil
}
