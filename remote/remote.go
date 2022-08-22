package remote

import (
	"fmt"
	"gatehill.io/imposter/logging"
	"gatehill.io/imposter/workspace"
	"net/url"
	"os"
	"path/filepath"
)

type Remote interface {
	GetType() string
	GetConfig() (*map[string]string, error)
	SetConfigValue(key string, value string) error
	Deploy() (*EndpointDetails, error)
	GetStatus() (*Status, error)
	GetConfigKeys() []string
}

type EndpointDetails struct {
	BaseUrl   string
	SpecUrl   string
	StatusUrl string
}

type Status struct {
	Status       string
	LastModified int64
}

var logger = logging.GetLogger()

var providers = make(map[string]func(dir string, workspace *workspace.Workspace) (Remote, error))

func Register(remoteType string, fn func(dir string, workspace *workspace.Workspace) (Remote, error)) {
	providers[remoteType] = fn
}

func SaveActiveRemoteType(dir string, remoteType string) (*workspace.Workspace, error) {
	f := providers[remoteType]
	if f == nil {
		return nil, fmt.Errorf("unsupported remote type: %s", remoteType)
	}

	active, m, err := workspace.GetActiveWithMetadata(dir)
	if err != nil {
		return nil, err
	}
	active.RemoteType = remoteType
	err = workspace.SaveMetadata(dir, m)
	if err != nil {
		return nil, err
	}
	logger.Tracef("set remote type: %s for active workspace: %s", remoteType, active.Name)
	return active, nil
}

func Load(dir string, workspace *workspace.Workspace) (*Remote, error) {
	provider := providers[workspace.RemoteType]
	if provider == nil {
		return nil, fmt.Errorf("unsupported remote type: %s", workspace.RemoteType)
	}
	remote, err := provider(dir, workspace)
	logger.Tracef("loaded remote [%s] for workspace: %s", remote.GetType(), workspace.Name)
	return &remote, err
}

func LoadActive(dir string) (*workspace.Workspace, *Remote, error) {
	active, err := workspace.GetActive(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load workspace: %s", err)
	} else if active == nil {
		return nil, nil, fmt.Errorf("no active workspace")
	}

	r, err := Load(dir, active)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load remote: %s", err)
	}
	return active, r, err
}

func GetConfigPath(dir string, w *workspace.Workspace) (exists bool, remoteFilePath string, err error) {
	metadataDir, err := workspace.EnsureMetadataDir(dir)
	if err != nil {
		return false, "", err
	}
	remoteFileName := fmt.Sprintf("%s_%s.json", w.RemoteType, w.Name)
	remoteFilePath = filepath.Join(metadataDir, remoteFileName)
	if _, err = os.Stat(remoteFilePath); err != nil {
		if os.IsNotExist(err) {
			logger.Tracef("no remote config file for workspace: %s", w.Name)
			return false, remoteFilePath, nil
		} else {
			return false, "", fmt.Errorf("failed to stat remote config file: %s: %s", remoteFilePath, err)
		}
	}
	logger.Tracef("found remote config file for workspace: %s: %s", w.Name, remoteFilePath)
	return true, remoteFilePath, nil
}

func MustJoinPath(base string, elem string) string {
	result, err := url.JoinPath(base, elem)
	if err != nil {
		panic(fmt.Errorf("failed to join base URL %s to %s", base, elem))
	}
	return result
}
