package engine

import (
	"gatehill.io/imposter/cliconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	providers = make(map[string]func(version string) Provider)
	engines   = make(map[string]func(configDir string, startOptions StartOptions) MockEngine)
)

func RegisterProvider(engineType string, b func(version string) Provider) {
	providers[engineType] = b
}

func RegisterEngine(engineType string, b func(configDir string, startOptions StartOptions) MockEngine) {
	engines[engineType] = b
}

func GetProvider(engineType string, version string) Provider {
	et := getConfiguredEngineType(engineType)
	provider := providers[et]
	if provider == nil {
		logrus.Fatalf("unsupported engine type: %v", et)
	}
	return provider(version)
}

func BuildEngine(engineType string, configDir string, startOptions StartOptions) MockEngine {
	et := getConfiguredEngineType(engineType)
	eng := engines[et]
	if eng == nil {
		logrus.Fatalf("unsupported engine type: %v", et)
	}
	return eng(configDir, startOptions)
}

func getConfiguredEngineType(engineType string) string {
	return cliconfig.GetOrDefaultString(engineType, viper.GetString("engine"))
}
