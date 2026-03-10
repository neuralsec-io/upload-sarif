package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

func detectSarifFiles(fs afero.Fs, path string) ([]string, error) {
	info, err := fs.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", path)
		}
		return nil, fmt.Errorf("failed to access path %s: %w", path, err)
	}

	var sarifFiles []string

	if info.IsDir() {
		files, err := afero.ReadDir(fs, path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".sarif" {
				sarifFiles = append(sarifFiles, filepath.Join(path, f.Name()))
			}
		}
	} else {
		if filepath.Ext(path) != ".sarif" {
			return nil, fmt.Errorf("path must be a .sarif file or directory: %s", path)
		}
		sarifFiles = append(sarifFiles, path)
	}

	return sarifFiles, nil
}
