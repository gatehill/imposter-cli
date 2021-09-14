package cliconfig

import (
	"os"
	"strings"
)

type CliConfig struct {
	Version  string
	LogLevel string
}

var Config CliConfig

func init() {
	Config = CliConfig{
		Version:  "dev",
		LogLevel: "DEBUG",
	}
}

func GetOrDefaultString(s string, fallbacks ...string) string {
	if s != "" {
		return s
	}
	for _, fallback := range fallbacks {
		if fallback != "" {
			return fallback
		}
	}
	return ""
}

func MatchesConfigFileFmt(file os.DirEntry) bool {
	for _, configFileSuffix := range getConfigFileSuffixes() {
		if strings.HasSuffix(file.Name(), configFileSuffix) {
			return true
		}
	}
	return false
}

func getConfigFileSuffixes() []string {
	return []string{
		"-config.yaml",
		"-config.yml",
		"-config.json",
	}
}
