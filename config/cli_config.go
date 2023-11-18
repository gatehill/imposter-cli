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
	"os"
	"path"
	"path/filepath"
	"strings"

	"gatehill.io/imposter/logging"
	"github.com/spf13/viper"
)

type CliConfig struct {
	Version  string
	LogLevel string
}

type ConfigPair struct {
	Key   string
	Value string
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

func ParseConfig(args []string) []ConfigPair {
	var pairs []ConfigPair
	for _, arg := range args {
		if !strings.Contains(arg, "=") {
			logger.Warnf("invalid config item: %s", arg)
			continue
		}
		splitArgs := strings.Split(arg, "=")
		pairs = append(pairs, ConfigPair{
			Key:   splitArgs[0],
			Value: strings.Trim(splitArgs[1], `"`),
		})
	}
	return pairs
}

func WriteLocalConfigValue(configDir string, key string, value string) error {
	v := viper.New()

	localConfig := path.Join(configDir, LocalDirConfigFileName+".yaml")
	v.SetConfigFile(localConfig)

	// sink if does not exist
	_ = v.ReadInConfig()

	v.Set(key, value)
	err := v.WriteConfig()
	if err != nil {
		return fmt.Errorf("failed to write config file: %s: %v", localConfig, err)
	}

	logger.Tracef("wrote CLI config to: %s", localConfig)
	return nil
}
