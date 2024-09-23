package util

import (
	"os"
	"path/filepath"
)

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

func GetRootPath() (string, error) {
	rootDirMarker := "/migrations"
	if cwd, err := os.Getwd(); err == nil {
		for {
			if info, errDir := os.Stat(cwd + rootDirMarker); errDir == nil && info.IsDir() {
				return cwd, nil
			}
			parent := filepath.Dir(cwd)
			if parent == cwd {
				break
			}
			cwd = parent
		}
	}
	return "", os.ErrNotExist
}
