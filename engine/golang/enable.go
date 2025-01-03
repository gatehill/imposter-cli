package golang

import (
	"gatehill.io/imposter/engine"
)

var golangInitialised = false

// EnableEngine registers the Golang engine implementation
func EnableEngine() {
	if !golangInitialised {
		golangInitialised = true

		engine.RegisterLibrary(engine.EngineTypeGolang, func() engine.EngineLibrary {
			return NewLibrary()
		})
		engine.RegisterEngine(engine.EngineTypeGolang, func(configDir string, startOptions engine.StartOptions) engine.MockEngine {
			provider := NewProvider(startOptions.Version, binCacheDir)
			return NewGolangMockEngine(configDir, startOptions, provider)
		})
	}
}
