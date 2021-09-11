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

package openapi

import (
	"encoding/json"
	"fmt"
	"gatehill.io/imposter/fileutil"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

// DiscoverOpenApiSpecs finds JSON and YAML OpenAPI specification files
// within the given directory. It returns fully qualified paths
// to the files discovered.
func DiscoverOpenApiSpecs(configDir string) []string {
	var openApiSpecs []string

	for _, yamlFile := range append(fileutil.FindFilesWithExtension(configDir, ".yaml"), fileutil.FindFilesWithExtension(configDir, ".yml")...) {
		fullyQualifiedPath := filepath.Join(configDir, yamlFile)
		jsonContent, err := loadYamlAsJson(fullyQualifiedPath)
		if err != nil {
			logrus.Fatal(err)
		}
		if isOpenApiSpec(jsonContent) {
			openApiSpecs = append(openApiSpecs, fullyQualifiedPath)
		}
	}

	for _, jsonFile := range fileutil.FindFilesWithExtension(configDir, ".json") {
		fullyQualifiedPath := filepath.Join(configDir, jsonFile)
		jsonContent, err := ioutil.ReadFile(fullyQualifiedPath)
		if err != nil {
			logrus.Fatal(err)
		}
		if isOpenApiSpec(jsonContent) {
			openApiSpecs = append(openApiSpecs, fullyQualifiedPath)
		}
	}

	return openApiSpecs
}

func loadYamlAsJson(yamlFile string) ([]byte, error) {
	y, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		return nil, err
	}

	j, err := yaml.YAMLToJSON(y)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML at %v: %v\n", yamlFile, err)
	}
	return j, nil
}

func isOpenApiSpec(jsonContent []byte) bool {
	var spec map[string]interface{}
	if err := json.Unmarshal(jsonContent, &spec); err != nil {
		panic(err)
	}
	return spec["openapi"] != nil || spec["swagger"] != nil
}
