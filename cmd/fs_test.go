package main

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectSarifFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(fs afero.Fs)
		path     string
		expErr   string
		expFiles []string
	}{
		{
			name:   "Path does not exist",
			path:   "missing.sarif",
			expErr: "path does not exist: missing.sarif",
		},
		{
			name: "Path is not .sarif file",
			setup: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, "file.txt", []byte("dummy"), 0o644))
			},
			path:   "file.txt",
			expErr: "path must be a .sarif file or directory: file.txt",
		},
		{
			name: "Valid single .sarif file",
			setup: func(fs afero.Fs) {
				require.NoError(t, afero.WriteFile(fs, "report.sarif", []byte("{}"), 0o644))
			},
			path:     "report.sarif",
			expFiles: []string{"report.sarif"},
		},
		{
			name: "Directory with multiple .sarif files",
			setup: func(fs afero.Fs) {
				require.NoError(t, fs.MkdirAll("reports", 0o755))
				require.NoError(t, afero.WriteFile(fs, filepath.Join("reports", "one.sarif"), []byte("{}"), 0o644))
				require.NoError(t, afero.WriteFile(fs, filepath.Join("reports", "two.sarif"), []byte("{}"), 0o644))
				require.NoError(t, afero.WriteFile(fs, filepath.Join("reports", "ignore.txt"), []byte("nope"), 0o644))
			},
			path: "reports",
			expFiles: []string{
				filepath.Join("reports", "one.sarif"),
				filepath.Join("reports", "two.sarif"),
			},
		},
		{
			name: "Directory without .sarif files",
			setup: func(fs afero.Fs) {
				require.NoError(t, fs.MkdirAll("emptydir", 0o755))
				require.NoError(t, afero.WriteFile(fs, filepath.Join("emptydir", "file.txt"), []byte("nope"), 0o644))
			},
			path:     "emptydir",
			expFiles: []string{}, // no files, should just return empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setup != nil {
				tt.setup(fs)
			}

			files, err := detectSarifFiles(fs, tt.path)

			if tt.expErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.expErr)
				return
			}

			require.NoError(t, err)
			assert.ElementsMatch(t, tt.expFiles, files)
		})
	}
}
