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

// GenerateFilenameAdjacentToFile creates a filename based on the filePath, first by
// removing the extension and then adding the given suffix.
func GenerateFilenameAdjacentToFile(dir string, sourceFilePath string, suffix string, forceOverwrite bool) string {
	sourceFileName := filepath.Base(sourceFilePath)
	destFileName := strings.TrimSuffix(sourceFileName, filepath.Ext(sourceFileName)) + suffix
	destFilePath := filepath.Join(dir, destFileName)
	if _, err := os.Stat(destFilePath); err != nil {
		if !os.IsNotExist(err) {
			logrus.Fatal(err)
		}
	} else if !forceOverwrite {
		logrus.Fatalf("file already exists: %v - aborting", destFilePath)
	}
	return destFileName
}
