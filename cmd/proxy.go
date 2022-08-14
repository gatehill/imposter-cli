/*
Copyright © 2022 Pete Cornish <outofcoffee@gmail.com>

Licensed under the Apache License, Proxy 2.0 (the "License");
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
	"fmt"
	"gatehill.io/imposter/proxy"
	"github.com/spf13/cobra"
	"net/http"
	"os"
)

var proxyFlags = struct {
	port      int
	outputDir string
}{}

// proxyCmd represents the up command
var proxyCmd = &cobra.Command{
	Use:   "proxy [URL]",
	Short: "Proxy an endpoint and record HTTP exchanges",
	Long:  `Proxies an endpoint and records HTTP exchanges to file, in Imposter format.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		upstream := args[0]
		var outputDir string
		if proxyFlags.outputDir != "" {
			outputDir = proxyFlags.outputDir
		} else {
			workingDir, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			outputDir = workingDir
		}
		proxyUpstream(upstream, proxyFlags.port, outputDir)
	},
}

func init() {
	proxyCmd.Flags().IntVarP(&proxyFlags.port, "port", "p", 8080, "Port on which to listen")
	proxyCmd.Flags().StringVarP(&proxyFlags.outputDir, "output-dir", "o", "", "Directory in which HTTP exchanges are recorded (default: current working directory)")
	rootCmd.AddCommand(proxyCmd)
}

func proxyUpstream(upstream string, port int, dir string) {
	logger.Infof("starting proxy for upstream %s on port %v", upstream, port)
	recorderC := proxy.StartRecorder(upstream, dir)

	http.HandleFunc("/system/status", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprintf(writer, "ok\n")
	})
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		proxy.Handle(upstream, writer, request, func(statusCode int, respBody *[]byte, respHeaders *http.Header) {
			recorderC <- proxy.HttpExchange{
				Req:        request,
				StatusCode: statusCode,
				Body:       respBody,
				Headers:    respHeaders,
			}
		})
	})

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		panic(err)
	}
}
