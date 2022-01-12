package meta

import (
	"encoding/json"
	"fmt"
	"gatehill.io/imposter/library"
	"os"
	"path/filepath"
)

func loadState() (map[string]interface{}, error) {
	metaFile, err := ensureMetaFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata file: %s", err)
	}
	file, err := os.ReadFile(metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata file: %s: %s", metaFile, err)
	}
	var data map[string]interface{}
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall metadata file: %s: %s", metaFile, err)
	}
	return data, nil
}

func ensureMetaFile() (string, error) {
	metaDir, err := ensureMetaDir()
	if err != nil {
		return "", fmt.Errorf("failed to ensure meta dir: %s", err)
	}
	return filepath.Join(metaDir, "meta.json"), nil
}

func ensureMetaDir() (string, error) {
	return library.EnsureDirUsingConfig("meta.dir", ".imposter")
}

// ReadMetaPropertyString attempts to read the property with the given key
// from the meta store and then attempts to type assert any non-nil value
// as a string. If the store does not exist, or if the store cannot be parsed,
// or the property does not exist, the empty string is returned.
func ReadMetaPropertyString(key string) (string, error) {
	value, err := readMetaProperty(key)
	if err != nil || value == nil {
		return "", err
	}
	return value.(string), nil
}

// ReadMetaPropertyInt attempts to read the property with the given key
// from the meta store and then attempts to type assert any non-nil value
// as an int. If the store does not exist, or if the store cannot be parsed,
// or the property does not exist, 0 is returned.
func ReadMetaPropertyInt(key string) (int, error) {
	value, err := readMetaProperty(key)
	if err != nil || value == nil {
		return 0, err
	}
	return int(value.(float64)), nil
}

func readMetaProperty(key string) (interface{}, error) {
	state, err := loadState()
	if err != nil {
		return "", err
	}
	return state[key], nil
}

// WriteMetaProperty persists a key-value pair to the meta store.
// If the store cannot be loaded, the write will fail and an error
// will be logged.
func WriteMetaProperty(key string, value interface{}) error {
	state, err := loadState()
	if err != nil {
		return fmt.Errorf("failed to read existing metadata: %s", err)
	}
	state[key] = value
	j, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshall metadata: %s", err)
	}
	metaFile, err := ensureMetaFile()
	if err != nil {
		return fmt.Errorf("failed to get metadata file: %s", err)
	}
	err = os.WriteFile(metaFile, j, 0644)
	if err != nil {
		return fmt.Errorf("failed to write metadata file: %s", err)
	}
	return nil
}
