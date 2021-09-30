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

package cmd

import (
	"gatehill.io/imposter/fileutil"
	"gatehill.io/imposter/impostermodel"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"testing"
)

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}

func Test_createMockConfig(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	testConfigPath := filepath.Join(workingDir, "/testdata")

	type args struct {
		generateResources bool
		forceOverwrite    bool
		scriptEngine      impostermodel.ScriptEngine
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "generate no resources no script",
			args: args{
				generateResources: false,
				forceOverwrite:    true,
				scriptEngine:      impostermodel.ScriptEngineNone,
			},
		},
		{
			name: "generate no resources with script",
			args: args{
				generateResources: false,
				forceOverwrite:    true,
				scriptEngine:      impostermodel.ScriptEngineJavaScript,
			},
		},
		{
			name: "generate with resources no script",
			args: args{
				generateResources: true,
				forceOverwrite:    true,
				scriptEngine:      impostermodel.ScriptEngineNone,
			},
		},
		{
			name: "generate with resources with script",
			args: args{
				generateResources: true,
				forceOverwrite:    true,
				scriptEngine:      impostermodel.ScriptEngineJavaScript,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir := prepTestData(t, testConfigPath)
			impostermodel.CreateFromSpecs(configDir, tt.args.generateResources, tt.args.forceOverwrite, tt.args.scriptEngine)

			if !doesFileExist(filepath.Join(configDir, "order_service-config.yaml")) {
				t.Fatalf("imposter config file should exist")
			}

			scriptPath := filepath.Join(configDir, "order_service.js")
			if impostermodel.IsScriptEngineEnabled(tt.args.scriptEngine) {
				if !doesFileExist(scriptPath) {
					t.Fatalf("script file should exist")
				}
			} else {
				if doesFileExist(scriptPath) {
					t.Fatalf("script file should not exist")
				}
			}
		})
	}
}

func doesFileExist(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func prepTestData(t *testing.T, src string) string {
	tempDir, err := os.MkdirTemp(os.TempDir(), "specs")
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.CopyDirShallow(src, tempDir)
	if err != nil {
		t.Fatal(err)
	}
	return tempDir
}
