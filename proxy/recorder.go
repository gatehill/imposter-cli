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

package proxy

import (
	"fmt"
	"gatehill.io/imposter/impostermodel"
	"gatehill.io/imposter/stringutil"
	"github.com/google/uuid"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
)

func StartRecorder(upstream string, dir string) chan HttpExchange {
	recordC := make(chan HttpExchange)
	go func() {
		for {
			if err := record(upstream, dir, <-recordC); err != nil {
				logger.Warn(err)
			}
		}
	}()
	return recordC
}

func record(upstream string, dir string, exchange HttpExchange) error {
	upstreamUrl, err := url.Parse(upstream)
	if err != nil {
		return fmt.Errorf("failed to parse upstream URL: %v", err)
	}
	upstreamHost := upstreamUrl.Host

	req := exchange.Req
	reqId := uuid.New().String()

	fileExt := getFileExtension(exchange.Headers)
	respFile := path.Join(dir, upstreamHost+"-"+reqId+"-response"+fileExt)
	err = os.WriteFile(respFile, *exchange.Body, 0644)
	if err != nil {
		return fmt.Errorf("failed to write response file %s for %s %v: %v", respFile, req.Method, req.URL, err)
	}
	logger.Debugf("wrote response file %s for %s %v [%d bytes]", respFile, req.Method, req.URL, len(*exchange.Body))

	config := generateConfig(exchange, req, respFile)
	configFile := path.Join(dir, upstreamHost+"-"+reqId+"-config.yaml")
	err = os.WriteFile(configFile, config, 0644)
	if err != nil {
		logger.Warnf("failed to write config file %s for %s %v: %v", configFile, req.Method, req.URL, err)
	}
	logger.Debugf("wrote config file %s for %s %v", configFile, req.Method, req.URL)

	return nil
}

func getFileExtension(respHeaders *http.Header) string {
	contentType := respHeaders.Get("Content-Type")
	extensions, err := mime.ExtensionsByType(contentType)
	if err != nil || extensions == nil {
		return ".txt"
	}
	return extensions[0]
}

func generateConfig(exchange HttpExchange, req *http.Request, respFile string) []byte {
	var resources []impostermodel.Resource
	headers := make(map[string]string)
	for headerName, headerValues := range *exchange.Headers {
		if !stringutil.Contains(skipProxyHeaders, headerName) && !stringutil.Contains(skipRecordHeaders, headerName) {
			if len(headerValues) > 0 {
				headers[headerName] = headerValues[0]
			}
		}
	}
	resource := impostermodel.Resource{
		Path:   req.URL.Path,
		Method: req.Method,
		Response: &impostermodel.ResponseConfig{
			StatusCode: exchange.StatusCode,
			StaticFile: path.Base(respFile),
			Headers:    &headers,
		},
	}
	resources = append(resources, resource)

	options := impostermodel.ConfigGenerationOptions{PluginName: "rest"}
	config := impostermodel.GenerateConfig(options, resources)
	return config
}
