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
	"gatehill.io/imposter/engine"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_describeVersions(t *testing.T) {
	// set an explicit version
	viper.Set("version", "3.0.2")

	type args struct {
		engineType engine.EngineType
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "print version with docker engine",
			args: args{
				engineType: engine.EngineTypeDocker,
			},
		},
		{
			name: "print version with jvm engine",
			args: args{
				engineType: engine.EngineTypeJvmSingleJar,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := `imposter-cli dev
imposter-engine 3.0.2
engine-output 3.0.2
`
			got := describeVersions(tt.args.engineType, outputFormatPlain, false)
			require.Equal(t, want, got, "version should match")
		})
	}
}
