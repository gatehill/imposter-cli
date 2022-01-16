package engine

import (
	"gatehill.io/imposter/library"
	"github.com/sirupsen/logrus"
)

const fileCacheDir = ".imposter/filecache/"

func EnsureFileCacheDir() (string, error) {
	// use IMPOSTER_CACHE_DIR directly, if set
	fileCacheDir, err := library.EnsureDirUsingConfig("cache.dir", fileCacheDir)
	if err != nil {
		return "", err
	}
	logrus.Tracef("ensured file cache directory: %v", fileCacheDir)
	return fileCacheDir, nil
}
