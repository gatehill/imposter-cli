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

import "sync"

type StartOptions struct {
	Port           int
	Version        string
	PullPolicy     PullPolicy
	LogLevel       string
	ReplaceRunning bool
	Deduplicate    string
}

type PullPolicy int

const (
	PullSkip         PullPolicy = iota
	PullAlways       PullPolicy = iota
	PullIfNotPresent PullPolicy = iota
)

type MockEngine interface {
	Start(wg *sync.WaitGroup)
	Stop(wg *sync.WaitGroup)
	Restart(wg *sync.WaitGroup)
	StopAllManaged() int
	GetVersionString() (string, error)
}

type ProviderOptions struct {
	EngineType EngineType
	Version    string
}

type Provider interface {
	Satisfied() bool
	Provide(policy PullPolicy) error
	GetEngineType() EngineType
}
