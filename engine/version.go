package engine

import (
	"encoding/json"
	"fmt"
	"gatehill.io/imposter/prefs"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const latestReleaseApi = "https://api.github.com/repos/outofcoffee/imposter/releases/latest"
const checkThresholdSeconds = 86_400

func ResolveLatestToVersion(allowCached bool) (string, error) {
	logrus.Tracef("resolving latest version (cache allowed: %v)", allowCached)

	now := time.Now().Unix()
	var latest string

	if allowCached {
		latest = loadCached(now)
	}

	if latest == "" {
		lookup, err := lookupLatest(now, allowCached)
		if err != nil {
			return "", err
		}
		latest = lookup
	}

	logrus.Tracef("resolved latest version: %s", latest)
	return latest, nil
}

func loadCached(now int64) string {
	var latest string

	lastCheck, _ := prefs.ReadPropertyInt("last_version_check")
	if now-int64(lastCheck) < checkThresholdSeconds {
		latest, _ = prefs.ReadPropertyString("latest")
	}

	logrus.Tracef("latest version cached value: %s", latest)
	return latest
}

func lookupLatest(now int64, allowFallbackToCached bool) (string, error) {
	latest, err := fetchLatestFromApi()
	if err != nil {
		if !allowFallbackToCached {
			return "", fmt.Errorf("failed to fetch latest version from API: %s", err)
		}

		logrus.Warnf("failed to fetch latest version from API (%s) - checking cache", err)
		latest = loadCached(now)
		if latest == "" {
			return "", fmt.Errorf("failed to resolve latest version (%s) and no cached version found", err)
		} else {
			// don't persist the cached version back to the prefs store
			return latest, nil
		}
	}

	err = prefs.WriteProperty("latest", latest)
	if err != nil {
		logrus.Warnf("failed to record latest version: %s", err)
	}
	err = prefs.WriteProperty("last_version_check", now)
	if err != nil {
		logrus.Warnf("failed to record last version check time: %s", err)
	}
	return latest, nil
}

func fetchLatestFromApi() (string, error) {
	logrus.Tracef("fetching latest version from: %s", latestReleaseApi)
	resp, err := http.Get(latestReleaseApi)
	if err != nil {
		return "", fmt.Errorf("failed to determine latest version from %s: %s", latestReleaseApi, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("failed to determine latest version from %s - status code: %d", latestReleaseApi, resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to determine latest version from %s - cannot read response body: %s", latestReleaseApi, err)
	}
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", fmt.Errorf("failed to determine latest version from %s - cannot unmarshall response body: %s", latestReleaseApi, err)
	}
	tagName := data["tag_name"].(string)
	return strings.TrimPrefix(tagName, "v"), nil
}
