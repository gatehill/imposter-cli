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
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

func StartRecorder(upstream string, dir string) (chan HttpExchange, error) {
	upstreamHost, err := formatUpstreamHostPort(upstream)
	if err != nil {
		return nil, err
	}

	var resources []impostermodel.Resource
	options := impostermodel.ConfigGenerationOptions{PluginName: "rest"}

	recordC := make(chan HttpExchange)
	go func() {
		for {
			exchange := <-recordC
			reqId := uuid.New().String()

			resource, err := record(upstreamHost, dir, reqId, exchange)
			if err != nil {
				logger.Warn(err)
				continue
			}
			resources = append(resources, *resource)

			if err := updateConfigFile(exchange, options, resources, dir, upstreamHost); err != nil {
				logger.Warn(err)
			}
		}
	}()

	return recordC, nil
}

func formatUpstreamHostPort(upstream string) (string, error) {
	upstreamUrl, err := url.Parse(upstream)
	if err != nil {
		return "", fmt.Errorf("failed to parse upstream URL: %v", err)
	}
	host := upstreamUrl.Host
	if !strings.Contains(host, ":") {
		return host, nil
	} else {
		hostOnly, port, err := net.SplitHostPort(host)
		if err != nil {
			return "", fmt.Errorf("failed to parse split upstream host/port: %v", err)
		}
		if port != "" {
			hostOnly += "-" + port
		}
		return hostOnly, nil
	}
}

func record(upstreamHost string, dir string, reqId string, exchange HttpExchange) (resource *impostermodel.Resource, err error) {
	req := exchange.Request

	fileExt := getFileExtension(exchange.ResponseHeaders)
	respFile := path.Join(dir, upstreamHost+"-"+reqId+"-response"+fileExt)
	err = os.WriteFile(respFile, *exchange.ResponseBody, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write response file %s for %s %v: %v", respFile, req.Method, req.URL, err)
	}
	logger.Debugf("wrote response file %s for %s %v [%d bytes]", respFile, req.Method, req.URL, len(*exchange.ResponseBody))

	r := buildResource(exchange, respFile)
	return &r, nil
}

func getFileExtension(respHeaders *http.Header) string {
	contentType := respHeaders.Get("Content-Type")
	if contentType != "" {
		if extensions, err := mime.ExtensionsByType(contentType); err == nil && len(extensions) > 0 {
			return extensions[0]
		}
	}
	return ".txt"
}

func buildResource(exchange HttpExchange, respFile string) impostermodel.Resource {
	req := *exchange.Request
	resource := impostermodel.Resource{
		Path:   req.URL.Path,
		Method: req.Method,
		Response: &impostermodel.ResponseConfig{
			StatusCode: exchange.StatusCode,
			StaticFile: path.Base(respFile),
		},
	}
	if len(req.URL.Query()) > 0 {
		queryParams := make(map[string]string)
		for qk, qvs := range req.URL.Query() {
			if len(qvs) > 0 {
				queryParams[qk] = qvs[0]
			}
		}
		resource.QueryParams = &queryParams
	}
	if len(*exchange.ResponseHeaders) > 0 {
		headers := make(map[string]string)
		for headerName, headerValues := range *exchange.ResponseHeaders {
			if !stringutil.Contains(skipProxyHeaders, headerName) && !stringutil.Contains(skipRecordHeaders, headerName) {
				if len(headerValues) > 0 {
					headers[headerName] = headerValues[0]
				}
			}
		}
		resource.Response.Headers = &headers
	}
	return resource
}

func updateConfigFile(exchange HttpExchange, options impostermodel.ConfigGenerationOptions, resources []impostermodel.Resource, dir string, upstreamHost string) error {
	req := exchange.Request
	config := impostermodel.GenerateConfig(options, resources)
	configFile := path.Join(dir, upstreamHost+"-config.yaml")
	err := os.WriteFile(configFile, config, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file %s for %s %v: %v", configFile, req.Method, req.URL, err)
	}
	logger.Debugf("wrote config file %s for %s %v", configFile, req.Method, req.URL)
	return nil
}
