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
	JvmProviderOptions
	javaHome  string
	distroDir string
}

var unpackedDistroInitialised = false

func EnableUnpackedDistroEngine() engine.EngineType {
	if !unpackedDistroInitialised {
		unpackedDistroInitialised = true

		engine.RegisterProvider(engine.EngineTypeJvmUnpacked, func(version string) engine.Provider {
			return newUnpackedDistroProvider(version)
		})
		engine.RegisterEngine(engine.EngineTypeJvmUnpacked, func(configDir string, startOptions engine.StartOptions) engine.MockEngine {
			provider := newUnpackedDistroProvider(startOptions.Version)
			return buildEngine(configDir, &provider, startOptions)
		})
	}
	return engine.EngineTypeJvmUnpacked
}

func newUnpackedDistroProvider(version string) JvmProvider {
	return &UnpackedDistroProvider{
		JvmProviderOptions: JvmProviderOptions{
			ProviderOptions: engine.ProviderOptions{
				EngineType: engine.EngineTypeJvmUnpacked,
				Version:    version,
			},
		},
	}
}

func (p *UnpackedDistroProvider) GetStartCommand(args []string, env []string) *exec.Cmd {
	if p.javaHome == "" {
		javaHome, err := getJavaHome()
		if err != nil {
			logrus.Fatal(err)
		}
		p.javaHome = javaHome
	}
	if !p.Satisfied() {
		if err := p.Provide(engine.PullIfNotPresent); err != nil {
			logrus.Fatal(err)
		}
	}
	startScript := filepath.Join(p.distroDir, "bin", "imposter")
	command := exec.Command(startScript, args...)
	command.Env = append(env, "JAVA_HOME="+p.javaHome)
	return command
}

func (p *UnpackedDistroProvider) Provide(engine.PullPolicy) error {
	envDistroDir := viper.GetString("jvm.distroDir")
	if envDistroDir != "" {
		fileInfo, err := os.Stat(envDistroDir)
		if err != nil {
			return fmt.Errorf("could not stat distribution directory: %v: %v", envDistroDir, err)
		} else if !fileInfo.IsDir() {
			return fmt.Errorf("distribution path is not a directory: %v", envDistroDir)
		}
		logrus.Debugf("using distribution at: %v", envDistroDir)
		p.distroDir = envDistroDir
		return nil
	}
	return fmt.Errorf("no distribution directory set")
}

func (p *UnpackedDistroProvider) Satisfied() bool {
	return p.distroDir != ""
}
