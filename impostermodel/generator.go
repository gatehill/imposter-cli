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

package impostermodel

import (
	"gatehill.io/imposter/openapi"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

type ResourceGenerationOptions struct {
	ScriptEngine   ScriptEngine
	ScriptFileName string
}

type ConfigGenerationOptions struct {
	ScriptEngine   ScriptEngine
	ScriptFileName string
}

func GenerateConfig(specFilePath string, resources []Resource, options ConfigGenerationOptions) []byte {
	pluginConfig := PluginConfig{
		Plugin:   "openapi",
		SpecFile: filepath.Base(specFilePath),
	}
	if len(resources) > 0 {
		pluginConfig.Resources = resources
	} else {
		if IsScriptEngineEnabled(options.ScriptEngine) {
			pluginConfig.Response = &ResponseConfig{
				ScriptFile: options.ScriptFileName,
			}
		}
	}

	config, err := yaml.Marshal(pluginConfig)
	if err != nil {
		logrus.Fatalf("unable to marshal imposter config: %v", err)
	}
	return config
}

func GenerateResourcesFromSpec(specFilePath string, options ResourceGenerationOptions) []Resource {
	var resources []Resource
	partialSpec, err := openapi.Parse(specFilePath)
	if err != nil {
		logrus.Fatalf("unable to parse openapi spec: %v: %v", specFilePath, err)
	}
	if partialSpec != nil {
		for path, pathDetail := range partialSpec.Paths {
			for verb := range pathDetail {
				resource := Resource{
					Path:   path,
					Method: strings.ToUpper(verb),
				}
				if IsScriptEngineEnabled(options.ScriptEngine) {
					resource.Response = &ResponseConfig{
						ScriptFile: options.ScriptFileName,
					}
				}
				resources = append(resources, resource)
			}
		}

	}
	return resources
}
