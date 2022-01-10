package library

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

func EnsureCache(settingsKey string, homeSubDirPath string) (string, error) {
	cachePath, err := GetCachePath(settingsKey, homeSubDirPath)
	if err != nil {
		return "", err
	}
	err = EnsurePath(cachePath)
	if err != nil {
		return "", err
	}
	logrus.Tracef("ensured cache directory: %v", cachePath)
	return cachePath, nil
}

func EnsurePath(cachePath string) error {
	if _, err := os.Stat(cachePath); err != nil {
		if os.IsNotExist(err) {
			logrus.Tracef("creating cache directory: %v", cachePath)
			err := os.MkdirAll(cachePath, 0700)
			if err != nil {
				return fmt.Errorf("failed to create cache directory: %v: %v", cachePath, err)
			}
		} else {
			return fmt.Errorf("failed to stat: %v: %v", cachePath, err)
		}
	}
	return nil
}

func GetCachePath(settingsKey string, homeSubDirPath string) (string, error) {
	if envCachePath := viper.GetString(settingsKey); envCachePath != "" {
		return envCachePath, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}
	return filepath.Join(homeDir, homeSubDirPath), nil
}
