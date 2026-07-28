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
		if paths[i], err = ResolvePath(directory, path, checkExists); err != nil {
			return err
		}
	}
	return nil
}

func ResolvePath(directory string, path string, checkExists bool) (string, error) {
	// An empty path means "unset" (an absent optional cover/stylesheet, etc.).
	// Resolving it would yield the directory itself (filepath.Join(dir, "") ==
	// dir), a bogus present path that callers' `if path != ""` guards then
	// treat as real — Typst tries to load a directory as an image, and EPUB
	// adds a directory as the cover. Keep unset unset.
	if path == "" {
		return "", nil
	}

	p, err := filepath.Abs(filepath.Join(directory, path))
	if err != nil {
		return path, err
	}

	if checkExists {
		if _, err = os.Stat(p); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				err = fmt.Errorf("path '%s' does not exist", p)
			}
			return p, err
		}
	}

	return p, nil
}
