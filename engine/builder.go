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

package engine

import (
	"gatehill.io/imposter/cliconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const defaultEngineType = "docker"

var (
	providers = make(map[string]func(version string) Provider)
	engines   = make(map[string]func(configDir string, startOptions StartOptions) MockEngine)
)

func RegisterProvider(engineType string, b func(version string) Provider) {
	providers[engineType] = b
}

func RegisterEngine(engineType string, b func(configDir string, startOptions StartOptions) MockEngine) {
	engines[engineType] = b
}

func GetProvider(engineType string, version string) Provider {
	et := getConfiguredEngineType(engineType)
	provider := providers[et]
	if provider == nil {
		logrus.Fatalf("unsupported engine type: %v", et)
	}
	return provider(version)
}

func BuildEngine(engineType string, configDir string, startOptions StartOptions) MockEngine {
	et := getConfiguredEngineType(engineType)
	eng := engines[et]
	if eng == nil {
		logrus.Fatalf("unsupported engine type: %v", et)
	}
	return eng(configDir, startOptions)
}

func getConfiguredEngineType(engineType string) string {
	return cliconfig.GetFirstNonEmpty(engineType, viper.GetString("engine"), defaultEngineType)
}
