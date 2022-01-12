package engine

import (
	"github.com/spf13/viper"
	"testing"
)

func TestGetConfiguredVersion(t *testing.T) {
	type args struct {
		override    string
		allowCached bool
	}
	tests := []struct {
		name             string
		args             args
		configureVersion string
		want             string
	}{
		{name: "return overridden version", args: args{override: "2.0.0", allowCached: false}, want: "2.0.0"},
		{name: "return configured version", args: args{override: "", allowCached: false}, configureVersion: "1.2.3", want: "1.2.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.configureVersion != "" {
				viper.Set("version", tt.configureVersion)
			}
			t.Cleanup(func() {
				viper.Set("version", nil)
			})
			if got := GetConfiguredVersion(tt.args.override, tt.args.allowCached); got != tt.want {
				t.Errorf("GetConfiguredVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetConfiguredType(t *testing.T) {
	type args struct {
		override string
	}
	tests := []struct {
		name          string
		args          args
		configureType string
		want          EngineType
	}{
		{name: "return overridden engine type", args: args{override: "docker"}, want: "docker"},
		{name: "return configured engine type", args: args{override: ""}, configureType: "jvm", want: "jvm"},
		{name: "return default engine type", args: args{override: ""}, want: defaultEngineType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.configureType != "" {
				viper.Set("engine", tt.configureType)
			}
			t.Cleanup(func() {
				viper.Set("engine", nil)
			})
			if got := GetConfiguredType(tt.args.override); got != tt.want {
				t.Errorf("GetConfiguredType() = %v, want %v", got, tt.want)
			}
		})
	}
}
