package jvm

import (
	"fmt"
	"gatehill.io/imposter/engine"
	"os"
	"os/exec"
	"strings"
)

type JvmEngineLibrary struct{}

func getSingleJarLibrary() *JvmEngineLibrary {
	return &JvmEngineLibrary{}
}

func (JvmEngineLibrary) CheckPrereqs() (bool, []string) {
	var msgs []string
	javaCmdPath, err := GetJavaCmdPath()
	if err != nil {
		msgs = append(msgs, fmt.Sprintf("❌ Failed to find JVM installation: %v", err))
		return false, msgs
	}
	msgs = append(msgs, fmt.Sprintf("✅ Found JVM installation: %v", javaCmdPath))

	cmd := exec.Command(javaCmdPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		msgs = append(msgs, fmt.Sprintf("❌ Failed to determine java version: %v", err))
		return false, msgs
	}
	msgs = append(msgs, fmt.Sprintf("✅ Java version installed: %v", string(output)))

	return true, msgs
}

func (JvmEngineLibrary) List() ([]engine.EngineMetadata, error) {
	binCachePath, err := ensureBinCache()
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(binCachePath)
	if err != nil {
		return nil, fmt.Errorf("error reading binary cache directory: %v: %v", binCachePath, err)
	}
	var available []engine.EngineMetadata
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".jar") || file.IsDir() {
			continue
		}
		fileVersion := strings.Split(strings.TrimSuffix(file.Name(), ".jar"), "-")[1]
		available = append(available, engine.EngineMetadata{
			EngineType: engine.EngineTypeJvmSingleJar,
			Version:    fileVersion,
		})
	}
	return available, nil
}

func (JvmEngineLibrary) GetProvider(version string) engine.Provider {
	return newSingleJarProvider(version)
}
