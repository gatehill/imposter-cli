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

package docker

import (
	"gatehill.io/imposter/debounce"
	"gatehill.io/imposter/engine"
)

type DockerMockEngine struct {
	configDir   string
	options     engine.StartOptions
	provider    *EngineImageProvider
	containerId string
	debouncer   debounce.Debouncer
}

var initialised = false

func EnableEngine() engine.EngineType {
	if !initialised {
		initialised = true

		engine.RegisterProvider(engine.EngineTypeDocker, func(version string) engine.Provider {
			return getProvider(version)
		})
		engine.RegisterEngine(engine.EngineTypeDocker, func(configDir string, startOptions engine.StartOptions) engine.MockEngine {
			return buildEngine(configDir, startOptions)
		})
	}
	return engine.EngineTypeDocker
}

func buildEngine(configDir string, options engine.StartOptions) engine.MockEngine {
	return &DockerMockEngine{
		configDir: configDir,
		options:   options,
		provider:  getProvider(options.Version),
		debouncer: debounce.Build(),
	}
}
