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

package jvm

import (
	"fmt"
	"gatehill.io/imposter/engine"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const binCacheDir = ".imposter/cache/"
const downloadUrlTemplate = "https://github.com/outofcoffee/imposter/releases/download/v%[1]v/imposter-%[1]v.jar"
const fallbackVersion = "1.22.0"

func GetJavaCmdPath() (string, error) {
	var binaryPathSuffix string
	if runtime.GOOS == "Windows" {
		binaryPathSuffix = ".exe"
	} else {
		binaryPathSuffix = ""
	}

	// prefer JAVA_HOME environment variable
	if javaHomeEnv, found := os.LookupEnv("JAVA_HOME"); found {
		return filepath.Join(javaHomeEnv, "/bin/java"+binaryPathSuffix), nil
	}

	if runtime.GOOS == "darwin" {
		command, stdout := exec.Command("/usr/libexec/java_home"), new(strings.Builder)
		command.Stdout = stdout
		err := command.Run()
		if err != nil {
			return "", fmt.Errorf("error determining JAVA_HOME: %v", err)
		}
		if command.ProcessState.Success() {
			return filepath.Join(strings.TrimSpace(stdout.String()), "/bin/java"+binaryPathSuffix), nil
		} else {
			return "", fmt.Errorf("failed to determine JAVA_HOME using libexec")
		}
	}

	// search for 'java' in the PATH
	javaPath, err := exec.LookPath("java")
	if err != nil {
		return "", fmt.Errorf("could not find 'java' in PATH: %v", err)
	}
	return javaPath, nil
}

func findImposterJar(version string, pullPolicy engine.PullPolicy) string {
	if version == "latest" {
		version = fallbackVersion
	}

	err, binCachePath := ensureBinCache()
	if err != nil {
		logrus.Fatal(err)
	}

	binFilePath := filepath.Join(binCachePath, fmt.Sprintf("imposter-%v.jar", version))
	if pullPolicy == engine.PullSkip {
		return binFilePath
	}

	if pullPolicy == engine.PullIfNotPresent {
		if _, err = os.Stat(binFilePath); err != nil {
			if !os.IsNotExist(err) {
				logrus.Fatalf("failed to stat: %v: %v", binFilePath, err)
			}
		} else {
			logrus.Debugf("engine version '%v' already present", version)
			logrus.Tracef("binary for version %v found at: %v", version, binFilePath)
			return binFilePath
		}
	}

	if err := downloadBinary(binFilePath, version); err != nil {
		logrus.Fatalf("failed to fetch binary: %v", err)
	}
	return binFilePath
}

func ensureBinCache() (error, string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err), ""
	}

	binCachePath := filepath.Join(homeDir, binCacheDir)
	if _, err = os.Stat(binCachePath); err != nil {
		if os.IsNotExist(err) {
			logrus.Tracef("creating cache directory: %v", binCachePath)
			err := os.MkdirAll(binCachePath, 0700)
			if err != nil {
				return fmt.Errorf("failed to create cache directory: %v: %v", binCachePath, err), ""
			}
		} else {
			return fmt.Errorf("failed to stat: %v: %v", binCachePath, err), ""
		}
	}

	logrus.Tracef("ensured binary cache directory: %v", binCachePath)
	return nil, binCachePath
}

func downloadBinary(localPath string, version string) error {
	url := fmt.Sprintf(downloadUrlTemplate, version)
	logrus.Debugf("downloading %v", url)

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("error creating file: %v: %v", localPath, err)
	}
	defer file.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error dwnloading from: %v: %v", url, err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
