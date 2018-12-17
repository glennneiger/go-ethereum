package main

import (
	"os"
	"path"
	"path/filepath"
)

func ResolvePath() error {
	if path.IsAbs(filename) {
		if _, err := os.Stat(filename); err == nil {
			// path exists, we will override the file
			return nil
		}
	}

	d, f := path.Split(filename)
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}

	_, err = os.Stat(path.Join(dir, filename))
	if err == nil {
		// path exists, we will override
		return nil
	}

	dirPath := path.Join(dir, d)
	filePath := path.Join(dirPath, f)
	if d != "" {
		err = os.MkdirAll(dirPath, os.ModeDir)
		if err != nil {
			return err
		}
	}

	filename = filePath
	return nil
}
