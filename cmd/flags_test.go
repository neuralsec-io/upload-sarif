package main

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		env      map[string]string
		expErr   string
		expFlags *Flags
	}{
		{
			name:   "Version only",
			args:   []string{"upload-sarif", "--version"},
			expErr: "",
			expFlags: &Flags{
				Version: true,
			},
		},
		{
			name:   "Missing all required flags",
			args:   []string{"upload-sarif"},
			expErr: "missing required flags:\n  --github-repo-owner is required (or set GITHUB_REPO_OWNER environment variable)\n  --github-repo-name is required (or set GITHUB_REPO_NAME environment variable)\n  --revision is required (or set REVISION environment variable)\n  --path is required\n  --api-key is required (or set API_KEY environment variable)",
		},
		{
			name: "Using environment variables for repo owner, name, revision and API key",
			args: []string{"upload-sarif", "--path", "report.sarif"},
			env: map[string]string{
				"GITHUB_REPO_OWNER": "neuralsec-io",
				"GITHUB_REPO_NAME":  "upload-sarif",
				"REVISION":          "abc123",
				"API_KEY":           "test-api-key",
			},
			expErr: "",
			expFlags: &Flags{
				LogLevel:        zerolog.InfoLevel,
				GitHubRepoOwner: "neuralsec-io",
				GitHubRepoName:  "upload-sarif",
				Revision:        "abc123",
				Path:            "report.sarif",
				APIKey:          "test-api-key",
			},
		},
		{
			name:   "Missing repo name, revision, path and API key",
			args:   []string{"upload-sarif", "--github-repo-owner", "neuralsec-io"},
			expErr: "missing required flags:\n  --github-repo-name is required (or set GITHUB_REPO_NAME environment variable)\n  --revision is required (or set REVISION environment variable)\n  --path is required\n  --api-key is required (or set API_KEY environment variable)",
		},
		{
			name: "Missing revision, path and API key",
			args: []string{
				"upload-sarif",
				"--github-repo-owner", "neuralsec-io",
				"--github-repo-name", "upload-sarif",
			},
			expErr: "missing required flags:\n  --revision is required (or set REVISION environment variable)\n  --path is required\n  --api-key is required (or set API_KEY environment variable)",
		},
		{
			name: "Missing path and API key only",
			args: []string{
				"upload-sarif",
				"--github-repo-owner", "neuralsec-io",
				"--github-repo-name", "upload-sarif",
				"--revision", "abc123",
			},
			expErr: "missing required flags:\n  --path is required\n  --api-key is required (or set API_KEY environment variable)",
		},
		{
			name: "All required flags provided (default log level)",
			args: []string{
				"upload-sarif",
				"--github-repo-owner", "neuralsec-io",
				"--github-repo-name", "upload-sarif",
				"--revision", "abc123",
				"--path", "results.sarif",
				"--api-key", "super-secret",
			},
			expErr: "",
			expFlags: &Flags{
				LogLevel:        zerolog.InfoLevel,
				GitHubRepoOwner: "neuralsec-io",
				GitHubRepoName:  "upload-sarif",
				Revision:        "abc123",
				Path:            "results.sarif",
				APIKey:          "super-secret",
			},
		},
		{
			name: "All required flags provided with debug log level",
			args: []string{
				"upload-sarif",
				"--github-repo-owner", "neuralsec-io",
				"--github-repo-name", "upload-sarif",
				"--revision", "abc123",
				"--path", "results.sarif",
				"--log-level", "debug",
				"--api-key", "super-secret",
			},
			expErr: "",
			expFlags: &Flags{
				LogLevel:        zerolog.DebugLevel,
				GitHubRepoOwner: "neuralsec-io",
				GitHubRepoName:  "upload-sarif",
				Revision:        "abc123",
				Path:            "results.sarif",
				APIKey:          "super-secret",
			},
		},
		{
			name: "Invalid log level",
			args: []string{
				"upload-sarif",
				"--github-repo-owner", "neuralsec-io",
				"--github-repo-name", "upload-sarif",
				"--revision", "abc123",
				"--path", "results.sarif",
				"--log-level", "verbose",
				"--api-key", "super-secret",
			},
			expErr: "invalid log level: verbose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			os.Clearenv()
			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
			}

			opts, err := parseFlags(tt.args)

			if tt.expErr == "" {
				require.NoError(t, err, "Expected no error")
				assert.Equal(t, tt.expFlags, opts, "Flags mismatch")
			} else {
				require.Error(t, err, "Expected error")
				assert.Equal(t, tt.expErr, err.Error())
			}
		})
	}
}
