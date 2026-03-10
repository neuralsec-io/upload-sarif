package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

type Flags struct {
	LogLevel        zerolog.Level
	Version         bool
	GitHubRepoOwner string
	GitHubRepoName  string
	Revision        string
	Path            string
	APIKey          string
}

func parseFlags(args []string) (*Flags, error) {
	fset := flag.NewFlagSet("upload-sarif", flag.ContinueOnError)

	logLevelStr := fset.String(
		"log-level",
		getEnvOrDefault("LOG_LEVEL", "info"),
		"Sets the log verbosity level. Options: debug, info, warn, error, fatal.",
	)
	version := fset.Bool("version", false, "Displays the application version and exits.")

	githubRepoOwner := fset.String(
		"github-repo-owner",
		getEnvOrDefault("GITHUB_REPO_OWNER", ""),
		"The name of the GitHub repository owner.",
	)
	githubRepoName := fset.String(
		"github-repo-name",
		getEnvOrDefault("GITHUB_REPO_NAME", ""),
		"The name of the GitHub repository.",
	)
	revision := fset.String("revision", getEnvOrDefault("REVISION", ""), "The git revision (commit SHA) to upload.")
	path := fset.String("path", "", "Path to a SARIF file or directory containing SARIF files.")
	apiKey := fset.String(
		"api-key",
		getEnvOrDefault("API_KEY", ""),
		"API key used for authentication (or set API_KEY environment variable).",
	)

	fset.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage of %s:
  %[1]s [flags]

Flags:
`, args[0])
		fset.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Environment variables:
  LOG_LEVEL            Sets default log level if not provided via flag
  GITHUB_REPO_OWNER    GitHub repository owner
  GITHUB_REPO_NAME     GitHub repository name
  REVISION             Git commit SHA
  API_KEY              API key used for authentication

Examples:
  # Show version
  %[1]s --version

  # Upload a specific SARIF file
  %[1]s --github-repo-owner=myorg --github-repo-name=myrepo --revision=abc123 --path=results.sarif --api-key=mytoken

  # Upload all SARIF files in a directory
  %[1]s --github-repo-owner=myorg --github-repo-name=myrepo --revision=abc123 --path=./reports/ --api-key=mytoken

  # Or use environment variables
  export GITHUB_REPO_OWNER=myorg
  export GITHUB_REPO_NAME=myrepo
  export REVISION=abc123
  export API_KEY=mytoken
  %[1]s
`, args[0])
	}

	if err := fset.Parse(args[1:]); err != nil {
		return nil, err
	}

	if *version {
		return &Flags{Version: true}, nil
	}

	logLevel, err := zerolog.ParseLevel(*logLevelStr)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %s", *logLevelStr)
	}

	flags := &Flags{
		LogLevel:        logLevel,
		Version:         *version,
		GitHubRepoOwner: *githubRepoOwner,
		GitHubRepoName:  *githubRepoName,
		Revision:        *revision,
		Path:            *path,
		APIKey:          *apiKey,
	}

	var validationErrors []string
	if flags.GitHubRepoOwner == "" {
		validationErrors = append(
			validationErrors,
			"--github-repo-owner is required (or set GITHUB_REPO_OWNER environment variable)",
		)
	}
	if flags.GitHubRepoName == "" {
		validationErrors = append(
			validationErrors,
			"--github-repo-name is required (or set GITHUB_REPO_NAME environment variable)",
		)
	}
	if flags.Revision == "" {
		validationErrors = append(validationErrors, "--revision is required (or set REVISION environment variable)")
	}
	if flags.Path == "" {
		validationErrors = append(validationErrors, "--path is required")
	}
	if flags.APIKey == "" {
		validationErrors = append(validationErrors, "--api-key is required (or set API_KEY environment variable)")
	}

	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("missing required flags:\n  %s", strings.Join(validationErrors, "\n  "))
	}

	return flags, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
