package cloudmocks

import (
	"encoding/json"
	"fmt"
	"gatehill.io/imposter/logging"
	"gatehill.io/imposter/prefs"
	"gatehill.io/imposter/remote"
	"gatehill.io/imposter/workspace"
	"net/url"
	"os"
	"strings"
)

const remoteType = "cloudmocks"
const defaultUrl = "https://api.mocks.cloud"

type Remote struct {
	workspace *workspace.Workspace
	dir       string
	config    config
}

type config struct {
	MockId string `json:"mockId"`
	Url    string `json:"url"`
}

var logger = logging.GetLogger()

func Register() {
	remote.Register(remoteType, func(dir string, workspace *workspace.Workspace) (remote.Remote, error) {
		return Load(dir, workspace)
	})
}

func Load(dir string, w *workspace.Workspace) (Remote, error) {
	c, err := loadConfig(dir, w)
	if err != nil {
		return Remote{}, err
	}

	r := Remote{
		workspace: w,
		dir:       dir,
		config:    c,
	}
	return r, nil
}

func (Remote) GetType() string {
	return remoteType
}

func (m Remote) GetUrl() string {
	return m.config.Url
}

func (m Remote) SetUrl(u string) error {
	u = strings.TrimSuffix(u, "/")
	if _, err := url.Parse(u); err != nil {
		return fmt.Errorf("failed to parse URL: %s: %s", u, err)
	}
	m.config.Url = u
	return m.saveConfig()
}

func (m Remote) getCleartextToken() (string, error) {
	cleartext, err := getCredsPrefs().ReadPropertyString(m.config.Url)
	if err != nil {
		return "", err
	}
	return cleartext, nil
}

func (m Remote) GetObfuscatedToken() (string, error) {
	cleartext, err := m.getCleartextToken()
	if err != nil {
		return "", err
	}
	obfuscated := strings.Repeat("*", 8) + cleartext[len(cleartext)-4:]
	return obfuscated, nil
}

func (m Remote) SetToken(token string) error {
	return getCredsPrefs().WriteProperty(m.config.Url, token)
}

func (m Remote) GetStatus() (*remote.Status, error) {
	s, err := m.getStatus()
	if err != nil {
		return nil, err
	}
	status := remote.Status{
		Status:       s.Status,
		LastModified: s.LastModified,
	}
	return &status, nil
}

func getCredsPrefs() prefs.Prefs {
	return prefs.Load("credentials.json")
}

func loadConfig(dir string, w *workspace.Workspace) (c config, err error) {
	exists, remoteFilePath, err := remote.GetConfigPath(dir, w)
	if err != nil {
		return config{}, err
	} else if !exists {
		c := config{
			Url: defaultUrl,
		}
		return c, nil
	}

	j, err := os.ReadFile(remoteFilePath)
	if err != nil {
		return config{}, fmt.Errorf("failed to load remote config file: %s: %s", remoteFilePath, err)
	}
	err = json.Unmarshal(j, &c)
	if err != nil {
		return config{}, fmt.Errorf("failed to unmarshall remote config file: %s: %s", remoteFilePath, err)
	}
	return c, nil
}

func (m Remote) saveConfig() error {
	_, remoteFilePath, err := remote.GetConfigPath(m.dir, m.workspace)
	if err != nil {
		return err
	}
	j, err := json.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshall remote config for workspace: %s: %s", m.workspace.Name, err)
	}
	err = os.WriteFile(remoteFilePath, j, 0644)
	if err != nil {
		return fmt.Errorf("failed to write remote config file: %s: %s", remoteFilePath, err)
	}
	return nil
}
