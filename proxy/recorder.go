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

type RecorderOptions struct {
	IgnoreDuplicateRequests bool
}

func StartRecorder(upstream string, dir string, options RecorderOptions) (chan HttpExchange, error) {
	upstreamHost, err := formatUpstreamHostPort(upstream)
	if err != nil {
		return nil, err
	}
	configFile := path.Join(dir, upstreamHost+"-config.yaml")
	if _, err := os.Stat(configFile); err == nil {
		return nil, fmt.Errorf("config file %s already exists", configFile)
	}

	var resources []impostermodel.Resource
	genOptions := impostermodel.ConfigGenerationOptions{PluginName: "rest"}

	var requestHashes []string
	responseHashes := make(map[string]string)

	recordC := make(chan HttpExchange)
	go func() {
		for {
			exchange := <-recordC

			if options.IgnoreDuplicateRequests {
				requestHash := getRequestHash(exchange)
				if stringutil.Contains(requestHashes, requestHash) {
					logger.Debugf("skipping recording of duplicate of request %s %v", exchange.Request.Method, exchange.Request.URL)
					continue
				}
				requestHashes = append(requestHashes, requestHash)
			}

			reqId := uuid.New().String()
			resource, err := record(upstreamHost, dir, &responseHashes, reqId, exchange)
			if err != nil {
				logger.Warn(err)
				continue
			}
			resources = append(resources, *resource)

			if err := updateConfigFile(exchange, genOptions, resources, configFile); err != nil {
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

func record(upstreamHost string, dir string, responseHashes *map[string]string, reqId string, exchange HttpExchange) (resource *impostermodel.Resource, err error) {
	respFile, err := getResponseFile(upstreamHost, dir, responseHashes, reqId, exchange)
	if err != nil {
		return nil, err
	}
	r := buildResource(exchange, respFile)
	return &r, nil
}

// getResponseFile checks the map for the hash of the response body to see if it has already been
// written. If not, a new file is written and its hash stored in the map.
func getResponseFile(upstreamHost string, dir string, fileHashes *map[string]string, reqId string, exchange HttpExchange) (string, error) {
	var respFile string

	req := exchange.Request
	respBody := *exchange.ResponseBody
	bodyHash := stringutil.Sha1hash(respBody)

	if existing := (*fileHashes)[bodyHash]; existing != "" {
		respFile = existing
		logger.Debugf("reusing existing response file %s for %s %v [%d bytes]", respFile, req.Method, req.URL, len(respBody))
	} else {
		sanitisedPath := strings.ReplaceAll(req.URL.EscapedPath(), "/", "_")

		fileExt := getFileExtension(exchange.ResponseHeaders)
		respFile = path.Join(dir, upstreamHost+"-"+req.Method+"-"+sanitisedPath+"-"+reqId+"-response"+fileExt)
		err := os.WriteFile(respFile, respBody, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write response file %s for %s %v: %v", respFile, req.Method, req.URL, err)
		}
		logger.Debugf("wrote response file %s for %s %v [%d bytes]", respFile, req.Method, req.URL, len(respBody))
		(*fileHashes)[bodyHash] = respFile
	}
	return respFile, nil
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

func getRequestHash(exchange HttpExchange) string {
	return stringutil.Sha1hashString(exchange.Request.Method + exchange.Request.URL.String())
}

func updateConfigFile(exchange HttpExchange, options impostermodel.ConfigGenerationOptions, resources []impostermodel.Resource, configFile string) error {
	req := exchange.Request
	config := impostermodel.GenerateConfig(options, resources)
	err := os.WriteFile(configFile, config, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file %s for %s %v: %v", configFile, req.Method, req.URL, err)
	}
	logger.Debugf("wrote config file %s for %s %v", configFile, req.Method, req.URL)
	return nil
}
