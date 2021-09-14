package jvm

import (
	"fmt"
	"gatehill.io/imposter/engine"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
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
			mockEngine := BuildEngine(tt.fields.configDir, tt.fields.options)
			mockEngine.Start()

			baseUrl := fmt.Sprintf("http://localhost:%d", tt.fields.options.Port)
			t.Logf("waiting for mock engine to come up at %v", baseUrl)

			for {
				time.Sleep(100 * time.Millisecond)
				resp, err := http.Get(baseUrl + "/system/status")
				if err != nil {
					continue
				}
				if _, err := io.ReadAll(resp.Body); err != nil {
					continue
				}
				resp.Body.Close()
				if resp.StatusCode == 200 {
					break
				}
			}

			resp, err := http.Get(baseUrl + "/example")
			if err != nil {
				t.Fatalf("failed to invoke mock endpoint: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			wants := "Hello world"
			actual := string(body)
			if actual != wants {
				t.Fatalf("expected body to be '%v' but was '%v'", wants, actual)
			}
			mockEngine.Stop()
		})
	}
}
