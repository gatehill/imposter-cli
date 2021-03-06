package plugin

import (
	"fmt"
	"gatehill.io/imposter/config"
	"gatehill.io/imposter/library"
	"gatehill.io/imposter/stringutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

const pluginBaseDir = ".imposter/plugins/"

func EnsurePlugins(plugins []string, version string, saveDefault bool) (int, error) {
	logrus.Tracef("ensuring %d plugins: %v", len(plugins), plugins)
	if len(plugins) == 0 {
		return 0, nil
	}
	for _, plugin := range plugins {
		err := EnsurePlugin(plugin, version)
		if err != nil {
			return 0, fmt.Errorf("error ensuring plugin %s: %s", plugin, err)
		}
		logrus.Debugf("plugin %s version %s is installed", plugin, version)
	}
	if saveDefault {
		err := addDefaultPlugins(plugins)
		if err != nil {
			logrus.Warnf("error setting plugins as default: %s", err)
		}
	}
	return len(plugins), nil
}

func EnsureDefaultPlugins(version string) (int, error) {
	var plugins []string

	configPlugins, err := ListDefaultPlugins()
	if err != nil {
		return 0, err
	}

	for _, configPlugin := range configPlugins {
		// work-around for https://github.com/spf13/viper/issues/380
		if strings.Contains(configPlugin, ",") {
			for _, p := range strings.Split(configPlugin, ",") {
				plugins = append(plugins, p)
			}
		} else {
			plugins = append(plugins, configPlugin)
		}
	}

	logrus.Tracef("found %d default plugin(s): %v", len(plugins), plugins)
	return EnsurePlugins(plugins, version, false)
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
		logrus.Tracef("plugin %s version %s already exists at: %s", pluginName, version, pluginFilePath)
		return nil
	}
	logrus.Debugf("plugin %s version %s is not installed", pluginName, version)
	err = downloadPlugin(pluginName, version)
	if err != nil {
		return err
	}
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
	logrus.Infof("downloaded plugin %s version %s", pluginName, version)
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

// addDefaultPlugins adds the provided plugins to the list of default
// plugins, if they are not already present, and writes the
// configuration file.
func addDefaultPlugins(plugins []string) error {
	existing, err := ListDefaultPlugins()
	if err != nil {
		return fmt.Errorf("failed to lead default plugins: %s", err)
	}
	combined := stringutil.CombineUnique(existing, plugins)
	if len(existing) == len(combined) {
		// none added
		return nil
	}
	return writeDefaultPlugins(combined)
}

func ListDefaultPlugins() ([]string, error) {
	v, err := parseConfigFile()
	if err != nil {
		return []string{}, err
	} else {
		return v.GetStringSlice("default.plugins"), nil
	}
}

func writeDefaultPlugins(plugins []string) error {
	v, err := parseConfigFile()
	if err != nil {
		return err
	}
	v.Set("default.plugins", plugins)

	configDir, err := config.GetConfigDir()
	if err != nil {
		return err
	}
	configFilePath := filepath.Join(configDir, config.ConfigFileName+".yaml")
	err = v.WriteConfigAs(configFilePath)
	if err != nil {
		return fmt.Errorf("error writing default plugin configuration to: %s: %s", configFilePath, err)
	}

	logrus.Tracef("wrote default plugin configuration to: %s", configFilePath)
	return nil
}

func parseConfigFile() (*viper.Viper, error) {
	v := viper.New()
	configDir, err := config.GetConfigDir()
	if err != nil {
		return nil, err
	}
	v.AddConfigPath(configDir)
	v.SetConfigName(config.ConfigFileName)

	// sink if does not exist
	_ = v.ReadInConfig()
	return v, nil
}
