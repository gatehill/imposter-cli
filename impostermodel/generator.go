package impostermodel

import (
	"fmt"
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
	ScriptEngineGroovy                  = "groovy"
	ScriptEngineJavaScript              = "javascript"
)

func GenerateConfig(specFilePath string, generateResources bool, scriptEngine ScriptEngine, scriptFileName string) []byte {
	pluginConfig := PluginConfig{
		Plugin:   "openapi",
		SpecFile: filepath.Base(specFilePath),
	}
	if generateResources {
		logrus.Debug("generating resources from spec")
		pluginConfig.Resources = generateResourcesFromSpec(specFilePath, scriptEngine, scriptFileName)
	} else {
		logrus.Debug("skipping resource generation")
		if scriptEngine != ScriptEngineNone {
			pluginConfig.Response = &ResponseConfig{
				ScriptFile: scriptFileName,
			}
		}
	}

	config, err := yaml.Marshal(pluginConfig)
	if err != nil {
		logrus.Fatalf("unable to marshal imposter config: %v", err)
	}
	return config
}

func BuildScriptFileName(dir string, specFilePath string, scriptEngine ScriptEngine, forceOverwrite bool) string {
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
			panic(fmt.Errorf("script engine is disabled"))
		}
		scriptFileName = fileutil.GenerateFilenameAdjacentToFile(dir, specFilePath, scriptEngineExt, forceOverwrite)
	}
	return scriptFileName
}

func generateResourcesFromSpec(specFilePath string, scriptEngine ScriptEngine, scriptFileName string) []Resource {
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
				if scriptEngine != ScriptEngineNone {
					resource.Response = &ResponseConfig{
						ScriptFile: scriptFileName,
					}
				}
				resources = append(resources, resource)
			}
		}

	}
	return resources
}
