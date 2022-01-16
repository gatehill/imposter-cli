package engine

import (
	"gatehill.io/imposter/library"
	"github.com/sirupsen/logrus"
)

const cacheBaseDir = ".imposter/filecache/"

func EnsureFileCacheDir() (string, error) {
	// use IMPOSTER_CACHE_DIR directly, if set
	fileCacheDir, err := library.EnsureDirUsingConfig("cache.dir", cacheBaseDir)
	if err != nil {
		return "", err
	}
	err = library.EnsureDir(fileCacheDir)
	if err != nil {
		return "", err
	}
	logrus.Tracef("ensured file cache directory: %v", fileCacheDir)
	return fileCacheDir, nil
}
