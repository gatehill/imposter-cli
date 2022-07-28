package cmd

import (
	"gatehill.io/imposter/engine"
	"testing"
)

func Test_pull(t *testing.T) {
	type args struct {
		version    string
		engineType engine.EngineType
		pullPolicy engine.PullPolicy
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "pull latest docker always", args: args{
			version:    "latest",
			engineType: engine.EngineTypeDocker,
			pullPolicy: engine.PullAlways,
		}},
		{name: "pull specific docker version always", args: args{
			version:    "3.0.2",
			engineType: engine.EngineTypeDocker,
			pullPolicy: engine.PullAlways,
		}},
		{name: "pull specific docker version if not present", args: args{
			version:    "3.0.2",
			engineType: engine.EngineTypeDocker,
			pullPolicy: engine.PullIfNotPresent,
		}},
		{name: "pull latest jvm always", args: args{
			version:    "latest",
			engineType: engine.EngineTypeJvmSingleJar,
			pullPolicy: engine.PullAlways,
		}},
		{name: "pull specific jvm version always", args: args{
			version:    "3.0.2",
			engineType: engine.EngineTypeJvmSingleJar,
			pullPolicy: engine.PullAlways,
		}},
		{name: "pull specific jvm version if not present", args: args{
			version:    "3.0.2",
			engineType: engine.EngineTypeJvmSingleJar,
			pullPolicy: engine.PullIfNotPresent,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pull(tt.args.version, tt.args.engineType, tt.args.pullPolicy)
		})
	}
}
