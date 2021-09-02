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
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

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
		createMockConfig(configDir)
	},
}

func init() {
	rootCmd.AddCommand(scaffoldCmd)
}

func createMockConfig(configDir string) {
	openApiSpecs := discoverOpenApiSpecs(configDir)
	logrus.Infof("found %d OpenAPI specs", len(openApiSpecs))

	for _, openApiSpec := range openApiSpecs {
		writeMockConfig(configDir, openApiSpec)
	}
}

func discoverOpenApiSpecs(configDir string) []string {
	var openApiSpecs []string

	for _, yamlFile := range append(findFilesWithExtension(configDir, ".yaml"), findFilesWithExtension(configDir, ".yml")...) {
		jsonContent, err := loadYamlAsJson(yamlFile)
		if err != nil {
			logrus.Fatal(err)
		}
		if isOpenApiSpec(jsonContent) {
			openApiSpecs = append(openApiSpecs, yamlFile)
		}
	}

	for _, jsonFile := range findFilesWithExtension(configDir, ".json") {
		jsonContent, err := ioutil.ReadFile(jsonFile)
		if err != nil {
			logrus.Fatal(err)
		}
		if isOpenApiSpec(jsonContent) {
			openApiSpecs = append(openApiSpecs, jsonFile)
		}
	}

	return openApiSpecs
}

func findFilesWithExtension(root, ext string) []string {
	var filesWithExtension []string
	infos, err := ioutil.ReadDir(root)
	if err != nil {
		logrus.Fatal(err)
	}
	for _, info := range infos {
		if !info.IsDir() && filepath.Ext(info.Name()) == ext {
			filesWithExtension = append(filesWithExtension, info.Name())
		}
	}
	return filesWithExtension
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

func writeMockConfig(dir string, specFilePath string) {
	specFileName := filepath.Base(specFilePath)
	configFileName := fmt.Sprintf("%v-config.yaml", strings.Replace(specFileName, filepath.Ext(specFileName), "", -1))
	configFilePath := filepath.Join(dir, configFileName)
	if _, err := os.Stat(configFilePath); err != nil {
		if !os.IsNotExist(err) {
			logrus.Fatal(err)
		}
	} else {
		logrus.Fatalf("config file already exists: %v - aborting", configFilePath)
	}

	configFile, err := os.Create(configFilePath)
	if err != nil {
		logrus.Fatal(err)
	}
	defer configFile.Close()

	config := fmt.Sprintf(`---
plugin: openapi
specFile: "%v"
`, specFileName)

	_, err = configFile.WriteString(config)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("wrote Imposter config: %v", configFile.Name())
}
