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
	"os"
	"path/filepath"
)

func CreateFromSpecs(configDir string, generateResources bool, forceOverwrite bool, scriptEngine ScriptEngine) {
	openApiSpecs := openapi.DiscoverOpenApiSpecs(configDir)
	logrus.Infof("found %d OpenAPI spec(s)", len(openApiSpecs))

	for _, openApiSpec := range openApiSpecs {
		writeMockConfig(openApiSpec, generateResources, forceOverwrite, scriptEngine)
	}
}

func writeMockConfig(specFilePath string, generateResources bool, forceOverwrite bool, scriptEngine ScriptEngine) {
	var scriptFileName string
	if IsScriptEngineEnabled(scriptEngine) {
		scriptFilePath := writeScriptFile(specFilePath, scriptEngine, forceOverwrite)
		scriptFileName = filepath.Base(scriptFilePath)
	}

	var resources []Resource
	if generateResources {
		resources = buildResources(specFilePath, scriptEngine, scriptFileName)
	} else {
		logrus.Debug("skipping resource generation")
	}

	config := GenerateConfig(specFilePath, resources, ConfigGenerationOptions{
		ScriptEngine:   scriptEngine,
		ScriptFileName: scriptFileName,
	})

	configFilePath := fileutil.GenerateFilePathAdjacentToFile(specFilePath, "-config.yaml", forceOverwrite)
	configFile, err := os.Create(configFilePath)
	if err != nil {
		logrus.Fatal(err)
	}
	defer configFile.Close()
	_, err = configFile.Write(config)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("wrote Imposter config: %v", configFilePath)
}

func writeScriptFile(specFilePath string, engine ScriptEngine, forceOverwrite bool) string {
	scriptFilePath := BuildScriptFilePath(specFilePath, engine, forceOverwrite)
	scriptFile, err := os.Create(scriptFilePath)
	if err != nil {
		logrus.Fatalf("error writing script file: %v: %v", scriptFilePath, err)
	}
	defer scriptFile.Close()

	_, err = scriptFile.WriteString(`
// TODO add your custom logic here
logger.debug('method: ' + context.request.method);
logger.debug('path: ' + context.request.path);
logger.debug('pathParams: ' + context.request.pathParams);
logger.debug('queryParams: ' + context.request.queryParams);
logger.debug('headers: ' + context.request.headers);
`)
	if err != nil {
		logrus.Fatalf("error writing script file: %v: %v", scriptFilePath, err)
	}

	logrus.Infof("wrote script file: %v", scriptFilePath)
	return scriptFilePath
}

func buildResources(specFilePath string, scriptEngine ScriptEngine, scriptFileName string) []Resource {
	resources := GenerateResourcesFromSpec(specFilePath, ResourceGenerationOptions{
		ScriptEngine:   scriptEngine,
		ScriptFileName: scriptFileName,
	})
	logrus.Debugf("generated %d resources from spec", len(resources))
	return resources
}
