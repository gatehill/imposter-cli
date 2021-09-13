package jvm

import (
	"gatehill.io/imposter/engine"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"testing"
)

func TestJvmMockEngine_Start(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	testConfigPath := filepath.Join(workingDir, "/testdata")

	type fields struct {
		configDir string
		options   engine.StartOptions
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{name: "start engine", fields: fields{
			configDir: testConfigPath,
			options: engine.StartOptions{
				Port:       8080,
				Version:    "1.21.0",
				PullPolicy: engine.PullIfNotPresent,
				LogLevel:   "DEBUG",
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := BuildEngine(tt.fields.configDir, tt.fields.options)
			engine.Start()
		})
	}
}
