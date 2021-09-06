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
	"fmt"
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
	Use:   "scaffold [CONFIG_DIR]",
	Short: "Create Imposter configuration from OpenAPI specs",
	Long: `Creates Imposter configuration from one or more OpenAPI/Swagger specification files.

If CONFIG_DIR is not specified, the current working directory is used.`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		var configDir string
		if len(args) == 0 {
			configDir, _ = os.Getwd()
		} else {
			configDir, _ = filepath.Abs(args[0])
		}
		scriptEngine := parseScriptEngine(flagScriptEngine)
		createMockConfig(configDir, flagGenerateResources, flagForceOverwrite, scriptEngine)
	},
}

func init() {
	scaffoldCmd.Flags().BoolVar(&flagForceOverwrite, "force-overwrite", false, "Force overwrite destination file(s) if already exist")
	scaffoldCmd.Flags().BoolVar(&flagGenerateResources, "generate-resources", true, "Generate Imposter resources from OpenAPI paths")
	scaffoldCmd.Flags().StringVar(&flagScriptEngine, "script-engine", "none", "Generate placeholder Imposter script (none|groovy|js)")
	rootCmd.AddCommand(scaffoldCmd)
}

func parseScriptEngine(scriptEngine string) impostermodel.ScriptEngine {
	engine := impostermodel.ScriptEngine(scriptEngine)
	switch engine {
	case impostermodel.ScriptEngineNone, impostermodel.ScriptEngineGroovy, impostermodel.ScriptEngineJavaScript:
		return engine
	default:
		panic(fmt.Errorf("unsupported script engine: %v", flagScriptEngine))
	}
}

func createMockConfig(configDir string, generateResources bool, forceOverwrite bool, scriptEngine impostermodel.ScriptEngine) {
	openApiSpecs := openapi.DiscoverOpenApiSpecs(configDir)
	logrus.Infof("found %d OpenAPI spec(s)", len(openApiSpecs))

	for _, openApiSpec := range openApiSpecs {
		writeMockConfig(configDir, openApiSpec, generateResources, forceOverwrite, scriptEngine)
	}
}

func writeMockConfig(dir string, specFilePath string, generateResources bool, forceOverwrite bool, scriptEngine impostermodel.ScriptEngine) {
	configFileName := fileutil.GenerateFilenameAdjacentToFile(dir, specFilePath, "-config.yaml", forceOverwrite)
	scriptFileName := writeScriptFile(dir, specFilePath, scriptEngine, forceOverwrite)
	config := impostermodel.GenerateConfig(specFilePath, generateResources, scriptEngine, scriptFileName)

	configFile, err := os.Create(filepath.Join(dir, configFileName))
	if err != nil {
		logrus.Fatal(err)
	}
	defer configFile.Close()
	_, err = configFile.Write(config)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("wrote Imposter config: %v", configFile.Name())
}

func writeScriptFile(dir string, spec string, engine impostermodel.ScriptEngine, forceOverwrite bool) (scriptFileName string) {
	if engine == impostermodel.ScriptEngineNone {
		return ""
	}
	scriptFileName = impostermodel.BuildScriptFileName(dir, spec, engine, forceOverwrite)
	scriptFilePath := filepath.Join(dir, scriptFileName)
	file, err := os.Create(scriptFilePath)
	if err != nil {
		logrus.Fatalf("error writing script file: %v: %v", scriptFilePath, err)
	}
	defer file.Close()

	_, err = file.WriteString(`
// TODO add your custom logic here
logger.debug(context.request);
`)
	if err != nil {
		logrus.Fatalf("error writing script file: %v: %v", scriptFilePath, err)
	}

	logrus.Infof("wrote script file: %v", scriptFilePath)
	return scriptFileName
}
