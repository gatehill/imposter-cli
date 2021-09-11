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

type StartOptions struct {
	Port            int
	ImageTag        string
	ImagePullPolicy ImagePullPolicy
	LogLevel        string
}

type ImagePullPolicy int

const (
	ImagePullSkip         ImagePullPolicy = iota
	ImagePullAlways       ImagePullPolicy = iota
	ImagePullIfNotPresent ImagePullPolicy = iota
)

type MockEngine interface {
	Start()
	Stop()
	Restart(stopCh chan string)
	TriggerRemovalAndNotify(stopCh chan string)
	NotifyOnStop(stopCh chan string)
	BlockUntilStopped()
}
