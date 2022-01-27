package cmd

import (
	"github.com/spf13/viper"
	"testing"
)

func Test_installPlugins(t *testing.T) {
	type args struct {
		argPlugins    []string
		configPlugins []string
		version       string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "install no plugins", args: args{argPlugins: nil, configPlugins: nil, version: "2.6.0"}},
		{name: "install plugins from args", args: args{argPlugins: []string{"store-redis"}, configPlugins: nil, version: "2.6.0"}},
		{name: "install plugins from config", args: args{argPlugins: nil, configPlugins: []string{"store-redis"}, version: "2.6.0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("plugins", tt.args.configPlugins)
			t.Cleanup(func() {
				viper.Set("plugins", nil)
			})
			installPlugins(tt.args.argPlugins, tt.args.version)
		})
	}
}
