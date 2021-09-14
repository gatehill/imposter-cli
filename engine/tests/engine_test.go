/*
Copyright © 2021 Pete Cornish <outofcoffee@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"fmt"
	"gatehill.io/imposter/debounce"
	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/engine/builder"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}

func TestEngine_StartStop(t *testing.T) {
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
		engine string
		fields fields
	}{
		{
			name:   "start docker engine",
			engine: "docker",
			fields: fields{
				configDir: testConfigPath,
				options: engine.StartOptions{
					Port:       8080,
					Version:    "1.21.0",
					PullPolicy: engine.PullIfNotPresent,
					LogLevel:   "DEBUG",
				},
			},
		},
		{
			name:   "start jvm engine",
			engine: "jvm",
			fields: fields{
				configDir: testConfigPath,
				options: engine.StartOptions{
					Port:       8080,
					Version:    "1.21.0",
					PullPolicy: engine.PullIfNotPresent,
					LogLevel:   "DEBUG",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := builder.DetermineEngine(tt.engine, tt.fields.configDir, tt.fields.options)
			mockEngine.Start()
			defer mockEngine.Stop()

			baseUrl := fmt.Sprintf("http://localhost:%d", tt.fields.options.Port)

			waitUntilUp(t, baseUrl)

			resp, err := http.Get(baseUrl + "/example")
			if err != nil {
				t.Fatalf("failed to invoke mock endpoint: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			expected := "Hello world"
			actual := string(body)
			if actual != expected {
				t.Fatalf("expected body to be '%v' but was '%v'", expected, actual)
			}
		})
	}
}

func TestEngine_Restart(t *testing.T) {
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
		engine string
		fields fields
	}{
		{
			name:   "restart docker engine",
			engine: "docker",
			fields: fields{
				configDir: testConfigPath,
				options: engine.StartOptions{
					Port:       8080,
					Version:    "1.21.0",
					PullPolicy: engine.PullIfNotPresent,
					LogLevel:   "DEBUG",
				},
			},
		},
		{
			name:   "restart jvm engine",
			engine: "jvm",
			fields: fields{
				configDir: testConfigPath,
				options: engine.StartOptions{
					Port:       8080,
					Version:    "1.21.0",
					PullPolicy: engine.PullIfNotPresent,
					LogLevel:   "DEBUG",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := builder.DetermineEngine(tt.engine, tt.fields.configDir, tt.fields.options)
			mockEngine.Start()
			defer mockEngine.Stop()

			baseUrl := fmt.Sprintf("http://localhost:%d", tt.fields.options.Port)

			waitUntilUp(t, baseUrl)

			stopCh := make(chan debounce.AtMostOnceEvent)
			mockEngine.Restart(stopCh)

			waitUntilUp(t, baseUrl)
		})
	}
}

func waitUntilUp(t *testing.T, baseUrl string) {
	url := baseUrl + "/system/status"
	t.Logf("waiting for mock engine to come up at %v", url)
	for {
		time.Sleep(100 * time.Millisecond)
		resp, err := http.Get(url)
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
}
