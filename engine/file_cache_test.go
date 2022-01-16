package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureFileCacheDir(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{name: "ensure file cache dir", want: filepath.Join(homeDir, cacheBaseDir), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EnsureFileCacheDir()
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureFileCacheDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EnsureFileCacheDir() got = %v, want %v", got, tt.want)
			}
		})
	}
}
