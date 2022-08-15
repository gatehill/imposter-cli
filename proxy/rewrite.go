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
