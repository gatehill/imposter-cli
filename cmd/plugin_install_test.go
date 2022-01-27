package cmd

import "testing"

func Test_installPlugins(t *testing.T) {
	type args struct {
		plugins []string
		version string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "install all configured plugins", args: args{plugins: nil, version: "2.6.0"}},
		{name: "install plugins as args", args: args{plugins: []string{"store-redis"}, version: "2.6.0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installPlugins(tt.args.plugins, tt.args.version)
		})
	}
}
