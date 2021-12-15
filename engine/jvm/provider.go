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

package jvm

import (
	"gatehill.io/imposter/debounce"
	"gatehill.io/imposter/engine"
	"os/exec"
)

type JvmMockEngine struct {
	configDir string
	options   engine.StartOptions
	provider  *JvmProvider
	javaCmd   string
	command   *exec.Cmd
	debouncer debounce.Debouncer
}

type JvmProvider interface {
	engine.Provider
	GetStartCommand(jvmMockEngine *JvmMockEngine, args []string) *exec.Cmd
}

func buildEngine(configDir string, provider *JvmProvider, options engine.StartOptions) engine.MockEngine {
	return &JvmMockEngine{
		configDir: configDir,
		options:   options,
		provider:  provider,
		debouncer: debounce.Build(),
	}
}
