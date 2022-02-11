package cmd

import (
	"gatehill.io/imposter/config"
	"gatehill.io/imposter/plugin"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func init() {
	configDir, err := os.MkdirTemp(os.TempDir(), "imposter-cli")
	if err != nil {
		panic(err)
	}
	config.DirPath = configDir
}

func Test_installPlugins(t *testing.T) {
	type args struct {
		argPlugins    []string
		configPlugins []string
		version       string
		saveDefault   bool
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "install no plugins", args: args{argPlugins: nil, configPlugins: nil, version: "2.6.0"}},
		{name: "install plugins from args", args: args{argPlugins: []string{"store-redis"}, configPlugins: nil, version: "2.6.0"}},
		{name: "install plugins from config", args: args{argPlugins: nil, configPlugins: []string{"store-redis"}, version: "2.6.0"}},
		{name: "install and save plugins as default", args: args{argPlugins: []string{"store-redis"}, configPlugins: nil, version: "2.6.0", saveDefault: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("plugins", tt.args.configPlugins)
			viper.Set("default.plugins", []string{})
			t.Cleanup(func() {
				viper.Set("plugins", nil)
			})
			installPlugins(tt.args.argPlugins, tt.args.version, tt.args.saveDefault)

			if tt.args.saveDefault {
				defaultPlugins, err := plugin.ListDefaultPlugins()
				if err != nil {
					t.Fatal(err)
				}
				require.ElementsMatch(t, tt.args.argPlugins, defaultPlugins, "default plugins should be set")
			}
		})
	}
}
