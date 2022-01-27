package plugin

import (
	"fmt"
	"gatehill.io/imposter/library"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

const pluginBaseDir = ".imposter/plugins/"

func EnsurePlugins(version string) (int, error) {
	plugins := viper.GetStringSlice("plugins")
	logrus.Tracef("ensuring %d plugins: %v", len(plugins), plugins)
	if len(plugins) == 0 {
		return 0, nil
	}
	for _, plugin := range plugins {
		err := EnsurePlugin(plugin, version)
		if err != nil {
			return 0, fmt.Errorf("error ensuring plugin %s: %s", plugin, err)
		}
	}
	return len(plugins), nil
}

func EnsurePlugin(pluginName string, version string) error {
	_, pluginFilePath, err := getPluginFilePath(pluginName, version)
	if err != nil {
		return err
	}
	if _, err := os.Stat(pluginFilePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("unable to stat plugin file: %s: %s", pluginFilePath, err)
		}
	} else {
		logrus.Tracef("plugin %s already exists at: %s", pluginName, pluginFilePath)
		return nil
	}
	err = downloadPlugin(pluginName, version)
	if err != nil {
		return err
	}
	logrus.Debugf("downloaded plugin %s", pluginName)
	return nil
}

func EnsurePluginDir(version string) (string, error) {
	fullPluginDir, err := getPluginDir(version)
	if err != nil {
		return "", err
	}
	err = library.EnsureDir(fullPluginDir)
	if err != nil {
		return "", err
	}
	logrus.Tracef("ensured plugin directory: %v", fullPluginDir)
	return fullPluginDir, nil
}

func getPluginDir(version string) (dir string, err error) {
	// use IMPOSTER_PLUGIN_DIR directly, if set
	fullPluginDir := viper.GetString("plugin.dir")
	if fullPluginDir == "" {
		basePluginDir, err := library.EnsureDirUsingConfig("plugin.baseDir", pluginBaseDir)
		if err != nil {
			return "", err
		}
		fullPluginDir = filepath.Join(basePluginDir, version)
	}
	return fullPluginDir, nil
}

func downloadPlugin(pluginName string, version string) error {
	fullPluginFileName, pluginFilePath, err := getPluginFilePath(pluginName, version)
	if err != nil {
		return err
	}
	err = library.DownloadBinary(pluginFilePath, fullPluginFileName, version)
	if err != nil {
		return err
	}
	logrus.Infof("downloaded plugin '%s' version %s", pluginName, version)
	return nil
}

func getPluginFilePath(pluginName string, version string) (string, string, error) {
	pluginDir, err := EnsurePluginDir(version)
	if err != nil {
		return "", "", err
	}
	fullPluginFileName := fmt.Sprintf("imposter-plugin-%s.jar", pluginName)
	pluginFilePath := filepath.Join(pluginDir, fullPluginFileName)
	return fullPluginFileName, pluginFilePath, err
}
