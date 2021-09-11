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
package fileutil

import (
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func FindFilesWithExtension(dir, ext string) []string {
	var filesWithExtension []string
	infos, err := ioutil.ReadDir(dir)
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

// GenerateFilePathAdjacentToFile creates a filename based on the sourceFilePath, first by
// removing the extension and then adding the given suffix. The full path is returned.
func GenerateFilePathAdjacentToFile(sourceFilePath string, suffix string, forceOverwrite bool) string {
	destFilePath := strings.TrimSuffix(sourceFilePath, filepath.Ext(sourceFilePath)) + suffix
	if _, err := os.Stat(destFilePath); err != nil {
		if !os.IsNotExist(err) {
			logrus.Fatal(err)
		}
	} else if !forceOverwrite {
		logrus.Fatalf("file already exists: %v - aborting", destFilePath)
	}
	return destFilePath
}
