package golang

import (
	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/library"
)

const binCacheDir = "bin/golang"

// Library implements the engine.EngineLibrary interface for the golang engine
type Library struct {
	binCache string
}

// NewLibrary creates a new instance of the golang engine library
func NewLibrary() *Library {
	return &Library{}
}

func (l *Library) CheckPrereqs() (bool, []string) {
	return true, nil
}

func (l *Library) List() ([]engine.EngineMetadata, error) {
	return []engine.EngineMetadata{
		{
			EngineType: engine.EngineTypeGolang,
			Version:    "latest",
		},
	}, nil
}

func (l *Library) GetProvider(version string) engine.Provider {
	binCachePath, err := l.ensureBinCache()
	if err != nil {
		providerLogger.Fatal(err)
	}
	return NewProvider(version, binCachePath)
}

func (l *Library) IsSealedDistro() bool {
	return false
}

func (l *Library) ShouldEnsurePlugins() bool {
	return false
}

func (l *Library) ensureBinCache() (string, error) {
	return library.EnsureDirUsingConfig("golang.binCache", binCacheDir)
}
