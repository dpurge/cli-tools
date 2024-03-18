package tool

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func FileExists(name string) bool {
	info, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func DirectoryExists(name string) bool {
	info, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func ResolvePaths(directory string, paths []string, checkExists bool) error {
	var err error
	for i, path := range paths {
		if paths[i], err = filepath.Abs(filepath.Join(directory, path)); err != nil {
			return err
		}
		if checkExists {
			if _, err = os.Stat(paths[i]); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					err = fmt.Errorf("path '%s' does not exist", paths[i])
				}
				return err
			}
		}
	}
	return nil
}
