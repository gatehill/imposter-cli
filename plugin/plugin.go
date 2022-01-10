package plugin

import (
	"fmt"
	"gatehill.io/imposter/library"
	"github.com/sirupsen/logrus"
	"path/filepath"
)

const pluginCacheDir = ".imposter/plugins/"

func EnsurePluginCache(version string) (string, error) {
	cachePath, err := library.EnsureCache("plugin.binCache", pluginCacheDir)
	if err != nil {
		return "", err
	}
	versionedPluginDir := filepath.Join(cachePath, version)
	err = library.EnsurePath(versionedPluginDir)
	if err != nil {
		return "", err
	}
	logrus.Tracef("ensured plugin directory: %v", versionedPluginDir)
	return versionedPluginDir, nil
}

func DownloadPlugin(pluginName string, engineVersion string) error {
	pluginDir, err := EnsurePluginCache(engineVersion)
	if err != nil {
		return err
	}
	fullPluginFileName := fmt.Sprintf("imposter-plugin-%s.jar", pluginName)
	pluginFilePath := filepath.Join(pluginDir, fullPluginFileName)

	err = library.DownloadBinary(pluginFilePath, fullPluginFileName, engineVersion, false)
	if err != nil {
		return err
	}
	return nil
}