package jvm

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// GetJavaCmdPath finds the best candidate for the 'java' command, searching
// the environment as well as using well-known OS-specific mechanisms.
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
	logrus.Tracef("using java: %v", javaPath)
	return javaPath, nil
}
