package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"
)

func Test_getResponseFile(t *testing.T) {
	outputDir, err := os.MkdirTemp(os.TempDir(), "imposter-cli")
	if err != nil {
		panic(err)
	}
	rootUrl, _ := url.Parse("https://example.com")

	responseBody := []byte("test")
	bodyHash := "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"

	type args struct {
		upstreamHost string
		dir          string
		options      RecorderOptions
		exchange     HttpExchange
		fileHashes   *map[string]string
		prefix       string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "empty response body",
			args: args{
				upstreamHost: "example.com",
				dir:          outputDir,
				options:      RecorderOptions{FlatResponseFileStructure: false},
				exchange: HttpExchange{
					Request:      &http.Request{Method: "GET", URL: rootUrl},
					ResponseBody: &[]byte{},
				},
				fileHashes: buildMap(outputDir, []string{}),
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "existing response body",
			args: args{
				upstreamHost: "example.com",
				dir:          outputDir,
				options:      RecorderOptions{FlatResponseFileStructure: false},
				exchange: HttpExchange{
					Request:         &http.Request{Method: "GET", URL: rootUrl},
					ResponseBody:    &responseBody,
					ResponseHeaders: &http.Header{},
				},
				fileHashes: buildMap(outputDir, []string{bodyHash}),
			},
			want:    path.Join(outputDir, "existing-file-0.txt"),
			wantErr: false,
		},
		{
			name: "new response body",
			args: args{
				upstreamHost: "example.com",
				dir:          outputDir,
				options:      RecorderOptions{FlatResponseFileStructure: false},
				exchange: HttpExchange{
					Request:         &http.Request{Method: "GET", URL: rootUrl},
					ResponseBody:    &responseBody,
					ResponseHeaders: &http.Header{},
				},
				fileHashes: buildMap(outputDir, []string{}),
			},
			want:    path.Join(outputDir, "GET-index.txt"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getResponseFile(tt.args.upstreamHost, tt.args.dir, tt.args.options, tt.args.exchange, tt.args.fileHashes, tt.args.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("getResponseFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getResponseFile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func buildMap(dir string, hashes []string) *map[string]string {
	m := make(map[string]string)
	for i, hash := range hashes {
		filename := fmt.Sprintf("existing-file-%d.txt", i)
		m[hash] = path.Join(dir, filename)
	}
	return &m
}
