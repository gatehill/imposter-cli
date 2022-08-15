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
	"bytes"
	"fmt"
	"mime"
	"net/http"
	"regexp"
)

var rewriteMediaTypes = []string{
	"text/.+",
	"application/javascript",
	"application/json",
	"application/xml",
}

func Rewrite(respHeaders *http.Header, respBody *[]byte, upstream string, port int) *[]byte {
	contentType := (*respHeaders).Get("Content-Type")
	if contentType == "" {
		logger.Warnf("no content type - skipping rewrite")
		return respBody
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		logger.Warnf("failed to parse content type - skipping rewrite: %v", err)
		return respBody
	}
	rewrite := false
	for _, rewriteMediaType := range rewriteMediaTypes {
		if matched, _ := regexp.MatchString(rewriteMediaType, mediaType); matched {
			rewrite = true
			break
		}
	}
	if !rewrite {
		logger.Debugf("unsupported content type %s for rewrite - skipping rewrite: %v", mediaType, err)
		return respBody
	}
	rewritten := bytes.ReplaceAll(*respBody, []byte(upstream), []byte(fmt.Sprintf("http://localhost:%d", port)))
	respBody = &rewritten
	return respBody
}
