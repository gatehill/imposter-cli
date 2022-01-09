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
	distroDir string
}

const mainClass = "io.gatehill.imposter.cmd.ImposterLauncher"

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
			EngineMetadata: engine.EngineMetadata{
				EngineType: engine.EngineTypeJvmUnpacked,
				Version:    version,
			},
		},
	}
}

func (p *UnpackedDistroProvider) GetStartCommand(args []string, env []string) *exec.Cmd {
	if p.javaCmd == "" {
		javaCmd, err := GetJavaCmdPath()
		if err != nil {
			logrus.Fatal(err)
		}
		p.javaCmd = javaCmd
	}
	if !p.Satisfied() {
		if err := p.Provide(engine.PullIfNotPresent); err != nil {
			logrus.Fatal(err)
		}
	}
	allArgs := append(
		[]string{"-classpath", filepath.Join(p.distroDir, "lib") + "/*", mainClass},
		args...,
	)
	command := exec.Command(p.javaCmd, allArgs...)
	command.Env = env
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
