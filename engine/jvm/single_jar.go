package jvm

import (
	"fmt"
	"gatehill.io/imposter/engine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type SingleJarProvider struct {
	JvmProviderOptions
	javaCmd string
	jarPath string
}

const binCacheDir = ".imposter/cache/"
const latestUrl = "https://github.com/outofcoffee/imposter/releases/latest/download/imposter.jar"
const versionedBaseUrlTemplate = "https://github.com/outofcoffee/imposter/releases/download/v%v/"

func init() {
	engine.RegisterProvider(engine.EngineTypeJvmSingleJar, func(version string) engine.Provider {
		return newSingleJarProvider(version)
	})
	engine.RegisterEngine(engine.EngineTypeJvmSingleJar, func(configDir string, startOptions engine.StartOptions) engine.MockEngine {
		provider := newSingleJarProvider(startOptions.Version)
		return buildEngine(configDir, &provider, startOptions)
	})
}

func newSingleJarProvider(version string) JvmProvider {
	return &SingleJarProvider{
		JvmProviderOptions: JvmProviderOptions{
			ProviderOptions: engine.ProviderOptions{
				EngineType: engine.EngineTypeJvmSingleJar,
				Version:    version,
			},
		},
	}
}

func (p *SingleJarProvider) GetStartCommand(args []string, env []string) *exec.Cmd {
	if p.javaCmd == "" {
		javaCmd, err := GetJavaCmdPath()
		if err != nil {
			logrus.Fatal(err)
		}
		p.javaCmd = javaCmd
	}
	if !p.Satisfied() {
		if err := p.Provide(engine.PullIfNotPresent); err != nil {
			logrus.Fatal(err)
		}
	}
	allArgs := append(
		[]string{"-jar", p.jarPath},
		args...,
	)
	command := exec.Command(p.javaCmd, allArgs...)
	command.Env = env
	return command
}

func (p *SingleJarProvider) Provide(policy engine.PullPolicy) error {
	jarPath, err := ensureBinary(p.Version, policy)
	if err != nil {
		return err
	}
	p.jarPath = jarPath
	return nil
}

func (p *SingleJarProvider) Satisfied() bool {
	return p.jarPath != ""
}

func ensureBinary(version string, policy engine.PullPolicy) (string, error) {
	if envJarFile := viper.GetString("jvm.jarFile"); envJarFile != "" {
		if _, err := os.Stat(envJarFile); err != nil {
			return "", fmt.Errorf("could not stat JAR file: %v: %v", envJarFile, err)
		}
		logrus.Debugf("using JAR file: %v", envJarFile)
		return envJarFile, nil
	}
	return checkOrDownloadBinary(version, policy)
}

func checkOrDownloadBinary(version string, policy engine.PullPolicy) (string, error) {
	binCachePath, err := ensureBinCache()
	if err != nil {
		logrus.Fatal(err)
	}

	binFilePath := filepath.Join(binCachePath, fmt.Sprintf("imposter-%v.jar", version))
	if policy == engine.PullSkip {
		return binFilePath, nil
	}

	if policy == engine.PullIfNotPresent {
		if _, err = os.Stat(binFilePath); err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("failed to stat: %v: %v", binFilePath, err)
			}
		} else {
			logrus.Debugf("engine binary '%v' already present", version)
			logrus.Tracef("binary for version %v found at: %v", version, binFilePath)
			return binFilePath, nil
		}
	}

	if err := downloadBinary(binFilePath, version); err != nil {
		return "", fmt.Errorf("failed to fetch binary: %v", err)
	}
	logrus.Tracef("using imposter at: %v", binFilePath)
	return binFilePath, nil
}

func ensureBinCache() (string, error) {
	binCachePath, err := getBinCachePath()
	if _, err = os.Stat(binCachePath); err != nil {
		if os.IsNotExist(err) {
			logrus.Tracef("creating cache directory: %v", binCachePath)
			err := os.MkdirAll(binCachePath, 0700)
			if err != nil {
				return "", fmt.Errorf("failed to create cache directory: %v: %v", binCachePath, err)
			}
		} else {
			return "", fmt.Errorf("failed to stat: %v: %v", binCachePath, err)
		}
	}

	logrus.Tracef("ensured binary cache directory: %v", binCachePath)
	return binCachePath, nil
}

func getBinCachePath() (string, error) {
	if envBinCache := viper.GetString("jvm.binCache"); envBinCache != "" {
		return envBinCache, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}
	return filepath.Join(homeDir, binCacheDir), nil
}

func downloadBinary(localPath string, version string) error {
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("error creating file: %v: %v", localPath, err)
	}
	defer file.Close()

	var url string
	var resp *http.Response
	if version == "latest" {
		url = latestUrl
		resp, err = makeHttpRequest(url, err)
		if err != nil {
			return err
		}

	} else {
		versionedBaseUrl := fmt.Sprintf(versionedBaseUrlTemplate, version)

		url := versionedBaseUrl + "imposter.jar"
		resp, err = makeHttpRequest(url, err)
		if err != nil {
			return err
		}

		// fallback to versioned binary filename
		if resp.StatusCode == 404 {
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
