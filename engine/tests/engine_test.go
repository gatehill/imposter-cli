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
	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/engine/builder"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
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
					Version:    "1.22.0",
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
					Port:       8081,
					Version:    "1.22.0",
					PullPolicy: engine.PullIfNotPresent,
					LogLevel:   "DEBUG",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			mockEngine := builder.BuildEngine(tt.engine, tt.fields.configDir, tt.fields.options)
			mockEngine.Start(wg)

			defer func() {
				mockEngine.Stop(wg)
				wg.Wait()
			}()

			checkUp(t, tt.fields.options.Port)

			url := fmt.Sprintf("http://localhost:%d/example", tt.fields.options.Port)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("failed to invoke mock endpoint: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			want := "Hello world"
			got := string(body)
			require.Equal(t, want, got, "expected body to match")
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
					Port:       8082,
					Version:    "1.22.0",
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
					Port:       8083,
					Version:    "1.22.0",
					PullPolicy: engine.PullIfNotPresent,
					LogLevel:   "DEBUG",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			mockEngine := builder.BuildEngine(tt.engine, tt.fields.configDir, tt.fields.options)
			mockEngine.Start(wg)

			defer func() {
				mockEngine.Stop(wg)
				wg.Wait()
			}()

			checkUp(t, tt.fields.options.Port)

			mockEngine.Restart(wg)
			checkUp(t, tt.fields.options.Port)
		})
	}
}

func checkUp(t *testing.T, port int) {
	url := fmt.Sprintf("http://localhost:%d/system/status", port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("error checking status endpoint: %v", err)
	}
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("error checking status endpoint: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == 200 {
		t.Logf("mock engine up at: %v", url)
	} else {
		t.Fatalf("unexpected response status code: %d", resp.StatusCode)
	}
}
