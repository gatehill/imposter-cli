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
	"gatehill.io/imposter/logging"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

type CliConfig struct {
	Version  string
	LogLevel string
}

// The GlobalConfigFileName is the file name without the file extension.
const GlobalConfigFileName = "config"

// The LocalDirConfigFileName is the file name without the file extension.
const LocalDirConfigFileName = ".imposter"

var logger = logging.GetLogger()

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

func GetGlobalConfigDir() (string, error) {
	if DirPath != "" {
		return DirPath, nil
	}
	return getDefaultGlobalConfigDir()
}

func getDefaultGlobalConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user home directory: %s", err)
	}
	configFile := filepath.Join(homeDir, ".imposter")
	return configFile, nil
}

func MergeCliConfigIfExists(configDir string) {
	viper.AddConfigPath(configDir)
	viper.SetConfigName(LocalDirConfigFileName)

	// If a local CLI config file is found, read it in.
	if err := viper.MergeInConfig(); err == nil {
		logger.Tracef("using local CLI config file: %v", viper.ConfigFileUsed())
	}
}
