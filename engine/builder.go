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
	"fmt"
	"gatehill.io/imposter/cliconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type EngineType string

const (
	EngineTypeDocker EngineType = "docker"
	EngineTypeJvm    EngineType = "jvm"
)
const defaultEngineType = EngineTypeDocker

var (
	providers = make(map[EngineType]func(version string) Provider)
	engines   = make(map[EngineType]func(configDir string, startOptions StartOptions) MockEngine)
)

func RegisterProvider(engineType EngineType, b func(version string) Provider) {
	providers[engineType] = b
}

func RegisterEngine(engineType EngineType, b func(configDir string, startOptions StartOptions) MockEngine) {
	engines[engineType] = b
}

func GetProvider(engineType EngineType, version string) Provider {
	if err := validateEngineType(engineType); err != nil {
		logrus.Fatal(err)
	}
	provider := providers[engineType]
	if provider == nil {
		logrus.Fatalf("unregistered engine type: %v", engineType)
	}
	return provider(version)
}

func BuildEngine(engineType EngineType, configDir string, startOptions StartOptions) MockEngine {
	if err := validateEngineType(engineType); err != nil {
		logrus.Fatal(err)
	}
	eng := engines[engineType]
	if eng == nil {
		logrus.Fatalf("unregistered engine type: %v", engineType)
	}
	return eng(configDir, startOptions)
}

func validateEngineType(engineType EngineType) error {
	switch engineType {
	case EngineTypeDocker, EngineTypeJvm:
		return nil
	}
	return fmt.Errorf("unsupported engine type: %v", engineType)
}

func GetConfiguredType(engineType string) EngineType {
	return EngineType(cliconfig.GetFirstNonEmpty(
		engineType,
		viper.GetString("engine"),
		string(defaultEngineType),
	))
}
