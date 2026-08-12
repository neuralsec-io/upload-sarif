package main

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// envKeys is every variable parseFlags reads. Each case clears all of them
// before setting the ones it needs, so a variable left over from the ambient
// environment cannot satisfy a flag the case expects to be missing.
var envKeys = []string{
	"GITHUB_REPO_OWNER",
	"GITHUB_REPO_NAME",
	"REVISION",
	"API_KEY",
	"LOG_LEVEL",
}

// Not parallel: subtests mutate the process environment via t.Setenv, which
// other tests in this package run concurrently with.
func TestParseFlags(t *testing.T) {
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
			// Deliberately not parallel. These cases set process-wide
			// environment variables, so running them concurrently meant one
			// subtest's os.Clearenv() could wipe variables another had just
			// set — the env-var case then failed with "missing required
			// flags" depending on scheduling. t.Setenv restores the previous
			// value automatically and panics if the test is parallel, so it
			// also stops this regressing.
			for _, k := range envKeys {
				t.Setenv(k, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
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
