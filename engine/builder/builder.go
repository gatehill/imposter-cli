package builder

import (
	"gatehill.io/imposter/cliconfig"
	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/engine/docker"
	"gatehill.io/imposter/engine/jvm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func DetermineEngine(engineType string, configDir string, startOptions engine.StartOptions) engine.MockEngine {
	switch cliconfig.GetOrDefaultString(engineType, viper.GetString("engine")) {
	case "", "docker":
		return docker.BuildEngine(configDir, startOptions)
	case "jvm":
		return jvm.BuildEngine(configDir, startOptions)
	default:
		logrus.Fatalf("unsupported engine type: %v", engineType)
		return nil
	}
}
