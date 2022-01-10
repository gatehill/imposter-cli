package library

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
)

const latestBaseUrl = "https://github.com/outofcoffee/imposter/releases/latest/download/"
const versionedBaseUrlTemplate = "https://github.com/outofcoffee/imposter/releases/download/v%v/"

func DownloadBinary(localPath string, remoteFileName string, version string, fallbackToVersionedFilename bool) error {
	logrus.Tracef("attempting to download %s version %s to %s", remoteFileName, version, localPath)
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("error creating file: %v: %v", localPath, err)
	}
	defer file.Close()

	var url string
	var resp *http.Response
	if version == "latest" {
		url = latestBaseUrl + remoteFileName
		resp, err = makeHttpRequest(url, err)
		if err != nil {
			return err
		}

	} else {
		versionedBaseUrl := fmt.Sprintf(versionedBaseUrlTemplate, version)

		url := versionedBaseUrl + remoteFileName
		resp, err = makeHttpRequest(url, err)
		if err != nil {
			return err
		}

		// fallback to versioned binary filename
		if resp.StatusCode == 404 && fallbackToVersionedFilename {
			logrus.Tracef("binary not found at: %v - retrying with versioned filename", url)
			url = versionedBaseUrl + fmt.Sprintf("imposter-%v.jar", version)
			resp, err = makeHttpRequest(url, err)
			if err != nil {
				return err
			}
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("error downloading from: %v: status code: %d", url, resp.StatusCode)
	}
	defer resp.Body.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}

func makeHttpRequest(url string, err error) (*http.Response, error) {
	logrus.Debugf("downloading %v", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error downloading from: %v: %v", url, err)
	}
	return resp, nil
}
