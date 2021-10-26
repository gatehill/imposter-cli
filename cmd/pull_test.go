package cmd

import (
	"gatehill.io/imposter/engine"
	"testing"
)

func Test_pull(t *testing.T) {
	type args struct {
		version    string
		pullPolicy engine.PullPolicy
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "pull latest always", args: args{
			version:    "latest",
			pullPolicy: engine.PullAlways,
		}},
		{name: "pull specific version always", args: args{
			version:    "1.22.0",
			pullPolicy: engine.PullAlways,
		}},
		{name: "pull specific version if not present", args: args{
			version:    "1.22.0",
			pullPolicy: engine.PullIfNotPresent,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pull(tt.args.version, tt.args.pullPolicy)
		})
	}
}
