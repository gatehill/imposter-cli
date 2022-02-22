package cloudmocks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gatehill.io/imposter/remote"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type createMockResponse struct {
	MockId string `json:"mockId"`
}

type getStatusResponse struct {
	Status       string `json:"status"`
	LastModified int    `json:"lastModified"`
}

type getEndpointResponse struct {
	BaseUrl string `json:"baseUrl"`
	SpecUrl string `json:"specUrl"`
}

func (m Remote) Deploy() (*remote.EndpointDetails, error) {
	if m.config.Url == "" {
		return nil, fmt.Errorf("URL cannot be null")
	} else if token, _ := m.GetObfuscatedToken(); token == "" {
		return nil, fmt.Errorf("auth token cannot be null")
	}

	err := m.ensureMockExists()
	if err != nil {
		return nil, err
	}

	err = m.setMockState("DRAFT")
	if err != nil {
		return nil, err
	}

	err = m.syncFiles(m.dir)
	if err != nil {
		return nil, err
	}

	err = m.setMockState("LIVE")
	if err != nil {
		return nil, err
	}
	if success := m.waitForStatus("ACTIVE", make(chan bool)); !success {
		return nil, fmt.Errorf("timed out waiting for mock to reach active status")
	}
	endpoint, err := m.getEndpoint()
	if err != nil {
		return nil, err
	}

	details := &remote.EndpointDetails{
		BaseUrl:   endpoint.BaseUrl,
		SpecUrl:   endpoint.SpecUrl,
		StatusUrl: endpoint.BaseUrl + "/system/status",
	}
	return details, nil
}

func (m Remote) ensureMockExists() error {
	if m.config.MockId == "" {
		logrus.Debugf("creating new mock")

		var resp createMockResponse
		err := m.request("POST", "/api/mocks", &resp)
		if err != nil {
			return fmt.Errorf("failed to create mock: %s", err)
		}

		logrus.Debugf("created mock with ID: %s", resp.MockId)
		m.config.MockId = resp.MockId
		err = m.saveConfig()
		if err != nil {
			return fmt.Errorf("failed to save mock ID: %s", err)
		}

	} else {
		logrus.Debugf("using existing mock with ID: %s", m.config.MockId)
	}
	return nil
}

func (m Remote) setMockState(state string) error {
	logrus.Debugf("setting mock state to: %s", state)
	var resp interface{}
	err := m.request("PATCH", fmt.Sprintf("/api/mocks/%s/state/%s", m.config.MockId, state), &resp)
	if err != nil {
		return fmt.Errorf("failed to set mock state to: %s: %s", state, err)
	}
	logrus.Infof("set mock state to: %s", state)
	return nil
}

func (m Remote) getStatus() (*getStatusResponse, error) {
	var resp getStatusResponse
	err := m.request("GET", fmt.Sprintf("/api/mocks/%s/status", m.config.MockId), &resp)
	if err != nil {
		return nil, fmt.Errorf("error getting status: %s", err)
	}
	return &resp, nil
}

func (m Remote) getEndpoint() (*getEndpointResponse, error) {
	var resp getEndpointResponse
	err := m.request("GET", fmt.Sprintf("/api/mocks/%s/endpoint", m.config.MockId), &resp)
	if err != nil {
		return nil, fmt.Errorf("error getting endpoint: %s", err)
	}
	return &resp, nil
}

func (m Remote) waitForStatus(s string, shutDownC chan bool) bool {
	logrus.Infof("waiting for mock status to be: %s...", s)

	finishedC := make(chan bool)
	max := time.NewTimer(120 * time.Second)
	defer max.Stop()

	go func() {
		for {
			time.Sleep(1000 * time.Millisecond)
			status, err := m.getStatus()
			if err != nil {
				continue
			}
			logrus.Tracef("mock status: %v", status.Status)
			if status.Status == s {
				finishedC <- true
				break
			} else if status.Status == "FAILED" {
				finishedC <- false
			}
		}
	}()

	finished := false
	select {
	case <-max.C:
		finished = true
		logrus.Fatalf("timed out waiting for mock status to be: %s", s)
		return false
	case success := <-finishedC:
		finished = success
		logrus.Tracef("finished probe with desired mock status: %v", success)
		return success
	case <-shutDownC:
		if !finished {
			logrus.Debugf("aborted status probe")
		}
		return false
	}
}

func (m Remote) request(method string, path string, response interface{}) error {
	url := m.config.Url + path
	client := http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}

	token, _ := m.getCleartextToken()
	req.Header = http.Header{
		"Authorization": []string{"Bearer " + token},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed to %s: %s", url, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("error requesting %s - HTTP status: %d", url, resp.StatusCode)
	}

	if resp.ContentLength > 0 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response from %s: %s", url, err)
		}
		_ = resp.Body.Close()
		err = json.Unmarshal(body, response)
		if err != nil {
			return fmt.Errorf("failed to unmarshall response from: %s: %s", url, err)
		}
	}
	return nil
}

func (m Remote) upload(method string, path string, src string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fi.Name())
	if err != nil {
		return err
	}
	part.Write(fileContents)

	err = writer.Close()
	if err != nil {
		return err
	}

	url := m.config.Url + path
	client := http.Client{}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	token, _ := m.getCleartextToken()
	req.Header = http.Header{
		"Authorization": []string{"Bearer " + token},
		"Content-Type":  []string{"multipart/form-data; boundary=" + writer.Boundary()},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed to %s: %s", url, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("error requesting %s - HTTP status: %d", url, resp.StatusCode)
	}
	return nil
}
