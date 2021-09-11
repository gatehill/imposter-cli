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
	"gatehill.io/imposter/fileutil"
	"gatehill.io/imposter/openapi"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

type ScriptEngine string

const (
	ScriptEngineNone       ScriptEngine = "none"
	ScriptEngineGroovy     ScriptEngine = "groovy"
	ScriptEngineJavaScript ScriptEngine = "javascript"
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
		if options.ScriptEngine != ScriptEngineNone {
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

func BuildScriptFileName(specFilePath string, scriptEngine ScriptEngine, forceOverwrite bool) string {
	var scriptFileName string
	if scriptEngine != ScriptEngineNone {
		var scriptEngineExt string
		switch scriptEngine {
		case ScriptEngineJavaScript:
			scriptEngineExt = ".js"
			break
		case ScriptEngineGroovy:
			scriptEngineExt = ".groovy"
			break
		default:
			logrus.Fatal("script engine is disabled")
		}
		scriptFileName = fileutil.GenerateFilenameAdjacentToFile(specFilePath, scriptEngineExt, forceOverwrite)
	}
	return scriptFileName
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
				if options.ScriptEngine != ScriptEngineNone {
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
