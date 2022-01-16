package plugin

import (
	"fmt"
	"gatehill.io/imposter/library"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"path/filepath"
)

const pluginBaseDir = ".imposter/plugins/"

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

func DownloadPlugin(pluginName string, engineVersion string) error {
	pluginDir, err := EnsurePluginDir(engineVersion)
	if err != nil {
		return err
	}
	fullPluginFileName := fmt.Sprintf("imposter-plugin-%s.jar", pluginName)
	pluginFilePath := filepath.Join(pluginDir, fullPluginFileName)

	err = library.DownloadBinary(pluginFilePath, fullPluginFileName, engineVersion)
	if err != nil {
		return err
	}
	return nil
}
