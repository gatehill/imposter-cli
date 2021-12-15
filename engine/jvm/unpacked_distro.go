package jvm

import (
	"fmt"
	"gatehill.io/imposter/engine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"path/filepath"
)

type UnpackedDistroProvider struct {
	engine.ProviderOptions
	distroPath string
}

func init() {
	engine.RegisterProvider(engine.EngineTypeJvmUnpacked, func(version string) engine.Provider {
		return newUnpackedDistroProvider(version)
	})
	engine.RegisterEngine(engine.EngineTypeJvmUnpacked, func(configDir string, startOptions engine.StartOptions) engine.MockEngine {
		provider := newUnpackedDistroProvider(startOptions.Version)
		return buildEngine(configDir, &provider, startOptions)
	})
}

func newUnpackedDistroProvider(version string) JvmProvider {
	return &UnpackedDistroProvider{
		ProviderOptions: engine.ProviderOptions{
			EngineType: engine.EngineTypeJvmUnpacked,
			Version:    version,
		},
	}
}

func (u *UnpackedDistroProvider) GetStartCommand(jvmMockEngine *JvmMockEngine, args []string) *exec.Cmd {
	if !u.Satisfied() {
		if err := u.Provide(engine.PullIfNotPresent); err != nil {
			logrus.Fatal(err)
		}
	}
	startScript := filepath.Join(u.distroPath, "bin", "imposter")
	command := exec.Command(startScript, args...)
	return command
}

func (u *UnpackedDistroProvider) Provide(policy engine.PullPolicy) error {
	envDistroDir := viper.GetString("jvm.distroDir")
	if envDistroDir != "" {
		fileInfo, err := os.Stat(envDistroDir)
		if err != nil {
			return fmt.Errorf("could not stat distribution directory: %v: %v", envDistroDir, err)
		} else if !fileInfo.IsDir() {
			return fmt.Errorf("distribution path is not a directory: %v", envDistroDir)
		}
		logrus.Debugf("using distribution at: %v", envDistroDir)
		u.distroPath = envDistroDir
		return nil
	}
	return fmt.Errorf("no distribution directory set")
}

func (u *UnpackedDistroProvider) Satisfied() bool {
	return u.distroPath != ""
}

func (u *UnpackedDistroProvider) GetEngineType() engine.EngineType {
	return u.EngineType
}
