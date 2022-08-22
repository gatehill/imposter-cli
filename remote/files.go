package remote

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ListLocal(dir string) ([]string, error) {
	logger.Tracef("listing files in: %s", dir)

	var files []string
	filenames, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory contents: %s: %s", dir, err)
	}
	for _, filename := range filenames {
		if filename.IsDir() || strings.HasPrefix(filename.Name(), ".") {
			// TODO optionally recurse
			continue
		}
		f := filepath.Join(dir, filename.Name())
		files = append(files, f)
	}
	return files, nil
}
