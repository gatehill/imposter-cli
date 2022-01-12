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
	"os"
	"strings"
)

type EngineType string

const (
	EngineTypeNone         EngineType = ""
	EngineTypeDocker       EngineType = "docker"
	EngineTypeJvmSingleJar EngineType = "jvm"
	EngineTypeJvmUnpacked  EngineType = "unpacked"
)
const defaultEngineType = EngineTypeDocker

var (
	libraries = make(map[EngineType]func() EngineLibrary)
	engines   = make(map[EngineType]func(configDir string, startOptions StartOptions) MockEngine)
)

func RegisterLibrary(engineType EngineType, b func() EngineLibrary) {
	libraries[engineType] = b
}

func RegisterEngine(engineType EngineType, b func(configDir string, startOptions StartOptions) MockEngine) {
	engines[engineType] = b
}

func EnumerateLibraries() []EngineType {
	var all []EngineType
	for key := range libraries {
		all = append(all, key)
	}
	return all
}

func GetLibrary(engineType EngineType) EngineLibrary {
	if err := validateEngineType(engineType); err != nil {
		logrus.Fatal(err)
	}
	library := libraries[engineType]
	if library == nil {
		logrus.Fatalf("unregistered engine type: %v", engineType)
	}
	return library()
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
	case EngineTypeDocker, EngineTypeJvmSingleJar, EngineTypeJvmUnpacked:
		return nil
	}
	return fmt.Errorf("unsupported engine type: %v", engineType)
}

func GetConfiguredType(override string) EngineType {
	return GetConfiguredTypeWithDefault(override, defaultEngineType)
}

func GetConfiguredTypeWithDefault(override string, defaultType EngineType) EngineType {
	return EngineType(cliconfig.GetFirstNonEmpty(
		override,
		viper.GetString("engine"),
		string(defaultType),
	))
}

func GetConfiguredVersion(override string, allowCached bool) string {
	version := cliconfig.GetFirstNonEmpty(
		override,
		viper.GetString("version"),
		"latest",
	)
	if version == "latest" {
		latest, err := ResolveLatestToVersion(allowCached)
		if err != nil {
			panic(err)
		}
		version = latest
	}
	return version
}

func SanitiseVersionOutput(s string) string {
	var remove = []string{
		"Version:",
		"WARNING: sun.reflect.Reflection.getCallerClass is not supported. This will impact performance.",
	}
	for _, r := range remove {
		s = strings.ReplaceAll(s, r, "")
	}
	return strings.TrimSpace(s)
}

func BuildEnv(options StartOptions, includeHome bool) []string {
	var env []string
	logLevelSet := false

	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "IMPOSTER_") ||
			strings.HasPrefix(e, "JAVA_TOOL_OPTIONS=") ||
			(includeHome && strings.HasPrefix(e, "HOME=")) {

			env = append(env, e)
		}
	}
	for _, e := range options.Environment {
		env = append(env, e)
		if strings.HasPrefix(e, "IMPOSTER_LOG_LEVEL=") {
			logLevelSet = true
		}
	}
	if !logLevelSet {
		env = append(env, "IMPOSTER_LOG_LEVEL="+strings.ToUpper(options.LogLevel))
	}

	return env
}
