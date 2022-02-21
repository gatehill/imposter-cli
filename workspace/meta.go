package workspace

import (
	"encoding/json"
	"fmt"
	"gatehill.io/imposter/library"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

type Manager interface {
	GetType() string
	GetUrl() string
	SetUrl(url string) error
	GetToken() (string, error)
	SetToken(token string) error
	Deploy() (*EndpointDetails, error)
}

type Workspace struct {
	Name        string `json:"name"`
	ManagerType string `json:"managerType"`
}

type Metadata struct {
	Workspaces []*Workspace `json:"workspaces"`
	Active     string       `json:"active"`
}

type EndpointDetails struct {
	BaseUrl   string
	SpecUrl   string
	StatusUrl string
}

func createOrLoadMetadata(dir string) (m *Metadata, err error) {
	metaFilePath, err := getMetaFilePath(dir)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(metaFilePath); err != nil {
		if os.IsNotExist(err) {
			logrus.Trace("creating empty workspace metadata")
			m = &Metadata{}
		} else {
			return nil, fmt.Errorf("failed to stat workspace file: %s: %s", metaFilePath, err)
		}
	} else {
		j, err := os.ReadFile(metaFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read workspace file: %s: %s", metaFilePath, err)
		}
		err = json.Unmarshal(j, &m)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshall workspace file: %s: %s", metaFilePath, err)
		}
	}
	return m, nil
}

func saveMetadata(dir string, m *Metadata) error {
	metaFilePath, err := getMetaFilePath(dir)
	if err != nil {
		return err
	}
	j, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshall metadata: %s", err)
	}
	err = os.WriteFile(metaFilePath, j, 0644)
	if err != nil {
		return fmt.Errorf("failed to save metadata to: %s: %s", metaFilePath, err)
	}
	return nil
}

func getMetaFilePath(dir string) (string, error) {
	metaDir, err := EnsureMetadataDir(dir)
	if err != nil {
		return "", err
	}
	metaFilePath := filepath.Join(metaDir, "workspaces.json")
	return metaFilePath, nil
}

func EnsureMetadataDir(dir string) (string, error) {
	metaDir := filepath.Join(dir, ".imposter_workspace")
	if err := library.EnsureDir(metaDir); err != nil {
		return "", fmt.Errorf("failed to ensure workspace metadata directory exists: %s: %s", metaDir, err)
	}
	return metaDir, nil
}
