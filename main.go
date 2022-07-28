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

package main

import (
	"gatehill.io/imposter/cmd"
	"gatehill.io/imposter/config"
	"gatehill.io/imposter/engine/docker"
	"gatehill.io/imposter/engine/jvm"
	"gatehill.io/imposter/logging"
	"gatehill.io/imposter/remote/cloudmocks"
	"gatehill.io/imposter/stringutil"
	"os"
)

const defaultLogLevel = "debug"

var version string

func main() {
	lvl := stringutil.GetFirstNonEmpty(os.Getenv("LOG_LEVEL"), os.Getenv("IMPOSTER_CLI_LOG_LEVEL"), defaultLogLevel)
	logging.SetLogLevel(lvl)

	if version == "" {
		version = "dev"
	}
	config.Config = config.CliConfig{
		LogLevel: lvl,
		Version:  version,
	}

	// engines
	docker.EnableEngine()
	jvm.EnableSingleJarEngine()
	jvm.EnableUnpackedDistroEngine()

	// backends
	cloudmocks.Register()

	cmd.Execute()
}
