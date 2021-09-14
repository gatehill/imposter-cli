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

package cmd

import (
	"gatehill.io/imposter/fileutil"
	"gatehill.io/imposter/impostermodel"
	"gatehill.io/imposter/openapi"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var flagForceOverwrite bool
var flagGenerateResources bool
var flagScriptEngine string

// scaffoldCmd represents the up command
var scaffoldCmd = &cobra.Command{
	Use:   "scaffold [DIR]",
	Short: "Create Imposter configuration from OpenAPI specs",
	Long: `Creates Imposter configuration from one or more OpenAPI/Swagger specification files
in a directory.

If DIR is not specified, the current working directory is used.`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		var configDir string
		if len(args) == 0 {
			configDir, _ = os.Getwd()
		} else {
			configDir, _ = filepath.Abs(args[0])
		}
		scriptEngine := impostermodel.ParseScriptEngine(flagScriptEngine)
		createMockConfig(configDir, flagGenerateResources, flagForceOverwrite, scriptEngine)
	},
}

func init() {
	scaffoldCmd.Flags().BoolVarP(&flagForceOverwrite, "force-overwrite", "f", false, "Force overwrite of destination file(s) if already exist")
	scaffoldCmd.Flags().BoolVar(&flagGenerateResources, "generate-resources", true, "Generate Imposter resources from OpenAPI paths")
	scaffoldCmd.Flags().StringVarP(&flagScriptEngine, "script-engine", "s", "none", "Generate placeholder Imposter script (none|groovy|js)")
	rootCmd.AddCommand(scaffoldCmd)
}

func createMockConfig(configDir string, generateResources bool, forceOverwrite bool, scriptEngine impostermodel.ScriptEngine) {
	openApiSpecs := openapi.DiscoverOpenApiSpecs(configDir)
	logrus.Infof("found %d OpenAPI spec(s)", len(openApiSpecs))

	for _, openApiSpec := range openApiSpecs {
		writeMockConfig(openApiSpec, generateResources, forceOverwrite, scriptEngine)
	}
}

func writeMockConfig(specFilePath string, generateResources bool, forceOverwrite bool, scriptEngine impostermodel.ScriptEngine) {
	var scriptFileName string
	if impostermodel.IsScriptEngineEnabled(scriptEngine) {
		scriptFilePath := writeScriptFile(specFilePath, scriptEngine, forceOverwrite)
		scriptFileName = filepath.Base(scriptFilePath)
	}

	var resources []impostermodel.Resource
	if generateResources {
		resources = buildResources(specFilePath, scriptEngine, scriptFileName)
	} else {
		logrus.Debug("skipping resource generation")
	}

	config := impostermodel.GenerateConfig(specFilePath, resources, impostermodel.ConfigGenerationOptions{
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

func writeScriptFile(specFilePath string, engine impostermodel.ScriptEngine, forceOverwrite bool) string {
	scriptFilePath := impostermodel.BuildScriptFilePath(specFilePath, engine, forceOverwrite)
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

func buildResources(specFilePath string, scriptEngine impostermodel.ScriptEngine, scriptFileName string) []impostermodel.Resource {
	resources := impostermodel.GenerateResourcesFromSpec(specFilePath, impostermodel.ResourceGenerationOptions{
		ScriptEngine:   scriptEngine,
		ScriptFileName: scriptFileName,
	})
	logrus.Debugf("generated %d resources from spec", len(resources))
	return resources
}
