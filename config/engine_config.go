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

package config

import (
	"os"
	"path/filepath"
	"strings"
)

// ContainsConfigFile determines if the specified configDir
// contains a file match the expected naming format
func ContainsConfigFile(configDir string, recursive bool) bool {
	files, err := os.ReadDir(configDir)
	if err != nil {
		logger.Errorf("unable to list directory contents: %v: %v", configDir, err)
		return false
	}
	for _, file := range files {
		if file.IsDir() && recursive {
			if ContainsConfigFile(filepath.Join(configDir, file.Name()), recursive) {
				return true
			}
		} else if matchesConfigFileFmt(file) {
			return true
		}
	}
	return false
}

func matchesConfigFileFmt(file os.DirEntry) bool {
	for _, configFileSuffix := range getConfigFileSuffixes() {
		if strings.HasSuffix(file.Name(), configFileSuffix) {
			return true
		}
	}
	return false
}

func getConfigFileSuffixes() []string {
	return []string{
		"-config.yaml",
		"-config.yml",
		"-config.json",
	}
}
