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
	"path/filepath"
	"strings"
)

type RecorderOptions struct {
	IgnoreDuplicateRequests   bool
	RecordOnlyResponseHeaders []string
	FlatResponseFileStructure bool
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

			var responseFileSuffix string
			requestHash := getRequestHash(exchange.Request)
			if stringutil.Contains(requestHashes, requestHash) {
				responseFileSuffix = "-" + uuid.New().String()
				if options.IgnoreDuplicateRequests {
					logger.Debugf("skipping recording of duplicate request %s %v", exchange.Request.Method, exchange.Request.URL)
					continue
				}
			} else {
				responseFileSuffix = ""
			}
			requestHashes = append(requestHashes, requestHash)

			resource, err := record(upstreamHost, dir, &responseHashes, responseFileSuffix, exchange, options)
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

func record(upstreamHost string, dir string, responseHashes *map[string]string, fileSuffix string, exchange HttpExchange, options RecorderOptions) (resource *impostermodel.Resource, err error) {
	respFile, err := getResponseFile(upstreamHost, dir, options, exchange, responseHashes, fileSuffix)
	if err != nil {
		return nil, err
	}
	r, err := buildResource(dir, options, exchange, respFile)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// getResponseFile checks the map for the hash of the response body to see if it has already been
// written. If not, a new file is written and its hash stored in the map.
func getResponseFile(
	upstreamHost string,
	dir string,
	options RecorderOptions,
	exchange HttpExchange,
	fileHashes *map[string]string,
	fileSuffix string,
) (string, error) {
	var respFile string

	req := exchange.Request
	respBody := *exchange.ResponseBody
	bodyHash := stringutil.Sha1hash(respBody)

	if existing := (*fileHashes)[bodyHash]; existing != "" {
		respFile = existing
		logger.Debugf("reusing identical response file %s for %s %v", respFile, req.Method, req.URL)
	} else {
		sanitisedPath := strings.TrimPrefix(req.URL.EscapedPath(), "/")
		fileExt := getFileExtension(exchange.ResponseHeaders)

		var parentDir, respFileName string
		if options.FlatResponseFileStructure {
			flatPath := strings.ReplaceAll(sanitisedPath, "/", "_")
			parentDir = dir
			respFileName = upstreamHost + "-" + req.Method + "-" + flatPath

		} else {
			respFullPath := path.Join(dir, sanitisedPath)
			respDir, err := ensureParentDirExists(respFullPath)
			if err != nil {
				return "", err
			}
			parentDir = respDir
			respFileName = req.Method + "-" + path.Base(respFullPath)
		}

		respFile = path.Join(parentDir, respFileName+fileSuffix+fileExt)
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

func ensureParentDirExists(respFullPath string) (string, error) {
	respDir := path.Dir(respFullPath)
	_, err := os.Stat(respDir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(respDir, 0700)
			if err != nil {
				return "", fmt.Errorf("failed to create response file dir: %s: %v", respDir, err)
			}
		} else {
			return "", fmt.Errorf("failed to stat response file dir: %s: %v", respDir, err)
		}
	}
	return respDir, nil
}

func buildResource(dir string, options RecorderOptions, exchange HttpExchange, respFile string) (impostermodel.Resource, error) {
	req := *exchange.Request
	relResponseFile, err := filepath.Rel(dir, respFile)
	if err != nil {
		return impostermodel.Resource{}, fmt.Errorf("failed to get relative path for response file: %s: %v", respFile, err)
	}
	resource := impostermodel.Resource{
		Path:   req.URL.Path,
		Method: req.Method,
		Response: &impostermodel.ResponseConfig{
			StatusCode: exchange.StatusCode,
			StaticFile: relResponseFile,
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
			shouldSkip := stringutil.Contains(skipProxyHeaders, headerName) || stringutil.Contains(skipRecordHeaders, headerName)
			if !shouldSkip &&
				(options.RecordOnlyResponseHeaders == nil) || stringutil.Contains(options.RecordOnlyResponseHeaders, headerName) {

				if len(headerValues) > 0 {
					headers[headerName] = headerValues[0]
				}
			}
		}
		resource.Response.Headers = &headers
	}
	return resource, nil
}

// getRequestHash generates a hash for a request based on the HTTP method and the URL. It does
// not take into consideration request headers.
func getRequestHash(req *http.Request) string {
	return stringutil.Sha1hashString(req.Method + req.URL.String())
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