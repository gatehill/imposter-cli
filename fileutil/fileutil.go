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
	"fmt"
	"gatehill.io/imposter/logging"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var logger = logging.GetLogger()

func FindFilesWithExtension(dir, ext string) []string {
	var filesWithExtension []string
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		logger.Fatal(err)
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
			logger.Fatal(err)
		}
	} else if !forceOverwrite {
		logger.Fatalf("file already exists: %v - aborting", destFilePath)
	}
	return destFilePath
}

func CopyDirShallow(src string, dest string) error {
	files, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error reading directory: %v: %v", src, err)
	}

	destInfo, err := os.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dest, 0700); err != nil {
				return fmt.Errorf("error creating directory: %v: %v", dest, err)
			}
		}
	}
	if !destInfo.IsDir() {
		return fmt.Errorf("destination is not a directory: %v", dest)
	}

	for _, file := range files {
		err := CopyFile(filepath.Join(src, file.Name()), filepath.Join(dest, file.Name()))
		if err != nil {
			return fmt.Errorf("error copying file: %v: %v", file.Name(), err)
		}
	}
	return nil
}

func CopyFile(src string, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error reading source: %s", err.Error())
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest) // creates if file doesn't exist
	if err != nil {
		return fmt.Errorf("error creating destination: %s", err.Error())
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile) // check first var for number of bytes copied
	if err != nil {
		return fmt.Errorf("error copying content: %s", err.Error())
	}

	return nil
}
