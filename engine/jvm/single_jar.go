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
	"runtime"
	"strings"
)

type SingleJarProvider struct {
	engine.ProviderOptions
	distroPath string
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
		ProviderOptions: engine.ProviderOptions{
			EngineType: engine.EngineTypeJvmSingleJar,
			Version:    version,
		},
	}
}

func (d *SingleJarProvider) GetStartCommand(jvmMockEngine *JvmMockEngine, args []string) *exec.Cmd {
	if jvmMockEngine.javaCmd == "" {
		javaCmd, err := GetJavaCmdPath()
		if err != nil {
			logrus.Fatal(err)
		}
		jvmMockEngine.javaCmd = javaCmd
	}
	if !d.Satisfied() {
		if err := d.Provide(engine.PullIfNotPresent); err != nil {
			logrus.Fatal(err)
		}
	}
	allArgs := append(
		[]string{"-jar", d.distroPath},
		args...,
	)
	command := exec.Command(jvmMockEngine.javaCmd, allArgs...)
	return command
}

func (d *SingleJarProvider) Provide(policy engine.PullPolicy) error {
	jarPath, err := ensureBinary(d.Version, policy)
	if err != nil {
		return err
	}
	d.distroPath = jarPath
	return nil
}

func (d *SingleJarProvider) Satisfied() bool {
	return d.distroPath != ""
}

func (d *SingleJarProvider) GetEngineType() engine.EngineType {
	return d.EngineType
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

func GetJavaCmdPath() (string, error) {
	var binaryPathSuffix string
	if runtime.GOOS == "Windows" {
		binaryPathSuffix = ".exe"
	} else {
		binaryPathSuffix = ""
	}

	// prefer JAVA_HOME environment variable
	if javaHomeEnv, found := os.LookupEnv("JAVA_HOME"); found {
		return filepath.Join(javaHomeEnv, "/bin/java"+binaryPathSuffix), nil
	}

	if runtime.GOOS == "darwin" {
		command, stdout := exec.Command("/usr/libexec/java_home"), new(strings.Builder)
		command.Stdout = stdout
		err := command.Run()
		if err != nil {
			return "", fmt.Errorf("error determining JAVA_HOME: %v", err)
		}
		if command.ProcessState.Success() {
			return filepath.Join(strings.TrimSpace(stdout.String()), "/bin/java"+binaryPathSuffix), nil
		} else {
			return "", fmt.Errorf("failed to determine JAVA_HOME using libexec")
		}
	}

	// search for 'java' in the PATH
	javaPath, err := exec.LookPath("java")
	if err != nil {
		return "", fmt.Errorf("could not find 'java' in PATH: %v", err)
	}
	logrus.Tracef("using java: %v", javaPath)
	return javaPath, nil
}
