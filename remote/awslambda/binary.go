package awslambda

import (
	"fmt"
	"gatehill.io/imposter/library"
	"os"
	"path/filepath"
)

func checkOrDownloadBinary(version string) (string, error) {
	binCachePath, err := ensureBinCache()
	if err != nil {
		logger.Fatal(err)
	}

	binFilePath := filepath.Join(binCachePath, fmt.Sprintf("imposter-awslambda-%v.zip", version))

	if _, err = os.Stat(binFilePath); err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat: %v: %v", binFilePath, err)
		}
	} else {
		logger.Debugf("lambda binary '%v' already present", version)
		logger.Tracef("lambda binary for version %v found at: %v", version, binFilePath)
		return binFilePath, nil
	}

	if err := library.DownloadBinary(binFilePath, "imposter-awslambda.zip", version); err != nil {
		return "", fmt.Errorf("failed to fetch lambda binary: %v", err)
	}
	logger.Tracef("using lambda binary at: %v", binFilePath)
	return binFilePath, nil
}

func ensureBinCache() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}
	dirPath := filepath.Join(homeDir, ".imposter/awslambda")
	if err = library.EnsureDir(dirPath); err != nil {
		return "", err
	}
	logger.Tracef("ensured directory: %v", dirPath)
	return dirPath, nil
}
