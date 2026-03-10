package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingClient struct {
	err error
}

func (c *failingClient) Do(_ *http.Request) (*http.Response, error) {
	return nil, c.err
}

func TestNewSarifUploader(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	uploader := NewSarifUploader("https://api.example.com/v1/sarif/github", "token", "owner", "repo", "sha1", fs)

	assert.Equal(t, "https://api.example.com/v1/sarif/github", uploader.Url)
	assert.Equal(t, "token", uploader.APIKey)
	assert.Equal(t, "owner", uploader.GitHubRepoOwner)
	assert.Equal(t, "repo", uploader.GitHubRepoName)
	assert.Equal(t, "sha1", uploader.Revision)
	assert.Equal(t, 15*time.Second, uploader.Timeout)
	assert.NotNil(t, uploader.Client)
	assert.Equal(t, fs, uploader.FS)
}

func TestSarifUploader_Upload(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "report.sarif", []byte(`{"runs": []}`), 0o644))

	t.Run("successful upload with multipart form", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/", r.URL.Path)
			assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"))
			assert.Equal(t, "test-secret-token", r.Header.Get("X-Access-Token"))

			require.NoError(t, r.ParseMultipartForm(10<<20))

			assert.Equal(t, "myorg", r.FormValue("github_repo_owner"))
			assert.Equal(t, "myrepo", r.FormValue("github_repo_name"))
			assert.Equal(t, "abc123", r.FormValue("revision"))

			file, header, err := r.FormFile("file")
			require.NoError(t, err)
			defer func() { _ = file.Close() }()

			assert.Equal(t, "report.sarif", header.Filename)

			data, err := io.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, `{"runs": []}`, string(data))

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ticket_ids":["uuid-1"]}`))
		}))
		defer server.Close()

		uploader := NewSarifUploader(server.URL, "test-secret-token", "myorg", "myrepo", "abc123", fs)
		err := uploader.Upload(context.Background(), "report.sarif")
		require.NoError(t, err)
	})

	t.Run("successful upload returns no error for 201 status", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		uploader := NewSarifUploader(server.URL, "token", "owner", "repo", "rev", fs)
		err := uploader.Upload(context.Background(), "report.sarif")
		require.NoError(t, err)
	})

	t.Run("upload without API key is rejected", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Access-Token") == "" {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		uploader := NewSarifUploader(server.URL, "", "myorg", "myrepo", "abc123", fs)
		err := uploader.Upload(context.Background(), "report.sarif")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upload failed: status=401")
	})

	t.Run("server returns 500 internal server error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))
		defer server.Close()

		uploader := NewSarifUploader(server.URL, "token", "owner", "repo", "rev", fs)
		err := uploader.Upload(context.Background(), "report.sarif")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upload failed: status=500")
	})

	t.Run("server returns 400 bad request", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "bad request", http.StatusBadRequest)
		}))
		defer server.Close()

		uploader := NewSarifUploader(server.URL, "token", "owner", "repo", "rev", fs)
		err := uploader.Upload(context.Background(), "report.sarif")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upload failed: status=400")
	})

	t.Run("error body is included in error message", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"invalid sarif format"}`))
		}))
		defer server.Close()

		uploader := NewSarifUploader(server.URL, "token", "owner", "repo", "rev", fs)
		err := uploader.Upload(context.Background(), "report.sarif")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upload failed: status=422")
		assert.Contains(t, err.Error(), "invalid sarif format")
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()

		uploader := NewSarifUploader("http://example.com", "token", "owner", "repo", "rev", fs)
		err := uploader.Upload(context.Background(), "missing.sarif")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open SARIF file")
	})

	t.Run("upload request creation fails due to invalid URL", func(t *testing.T) {
		t.Parallel()

		uploader := NewSarifUploader("http://[::1]:namedport", "token", "owner", "repo", "rev", fs)
		err := uploader.Upload(context.Background(), "report.sarif")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create upload request")
	})

	t.Run("http client Do returns error", func(t *testing.T) {
		t.Parallel()

		uploader := &SarifUploader{
			Client:          &failingClient{err: errors.New("connection refused")},
			Url:             "http://localhost:9999",
			FS:              fs,
			APIKey:          "token",
			GitHubRepoOwner: "owner",
			GitHubRepoName:  "repo",
			Revision:        "rev",
		}

		err := uploader.Upload(context.Background(), "report.sarif")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upload request failed")
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("cancelled context returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		uploader := NewSarifUploader(server.URL, "token", "owner", "repo", "rev", fs)
		err := uploader.Upload(ctx, "report.sarif")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upload request failed")
	})

	t.Run("file in subdirectory uses basename for filename", func(t *testing.T) {
		t.Parallel()

		subFs := afero.NewMemMapFs()
		require.NoError(t, subFs.MkdirAll("path/to", 0o755))
		require.NoError(t, afero.WriteFile(subFs, "path/to/scan.sarif", []byte(`{"runs":[]}`), 0o644))

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, r.ParseMultipartForm(10<<20))

			file, header, err := r.FormFile("file")
			require.NoError(t, err)
			defer func() { _ = file.Close() }()

			assert.Equal(t, "scan.sarif", header.Filename)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		uploader := NewSarifUploader(server.URL, "token", "owner", "repo", "rev", subFs)
		err := uploader.Upload(context.Background(), "path/to/scan.sarif")
		require.NoError(t, err)
	})

	t.Run("large file content is sent correctly", func(t *testing.T) {
		t.Parallel()

		largeFs := afero.NewMemMapFs()
		largeContent := `{"version":"2.1.0","$schema":"https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json","runs":[{"tool":{"driver":{"name":"TestTool","version":"1.0.0","rules":[{"id":"RULE001","name":"test-rule","shortDescription":{"text":"A test rule"}}]}},"results":[{"ruleId":"RULE001","level":"error","message":{"text":"Something went wrong"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/main.go"},"region":{"startLine":42,"startColumn":10}}}]}]}]}`
		require.NoError(t, afero.WriteFile(largeFs, "detailed.sarif", []byte(largeContent), 0o644))

		var receivedContent string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, r.ParseMultipartForm(10<<20))

			file, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer func() { _ = file.Close() }()

			data, err := io.ReadAll(file)
			require.NoError(t, err)
			receivedContent = string(data)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		uploader := NewSarifUploader(server.URL, "token", "owner", "repo", "rev", largeFs)
		err := uploader.Upload(context.Background(), "detailed.sarif")
		require.NoError(t, err)
		assert.Equal(t, largeContent, receivedContent)
	})

	t.Run("all form fields are sent correctly with special characters", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, r.ParseMultipartForm(10<<20))

			assert.Equal(t, "my-org-name", r.FormValue("github_repo_owner"))
			assert.Equal(t, "my-repo.name", r.FormValue("github_repo_name"))
			assert.Equal(t, "abcdef1234567890abcdef1234567890abcdef12", r.FormValue("revision"))

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		uploader := NewSarifUploader(
			server.URL, "token",
			"my-org-name", "my-repo.name", "abcdef1234567890abcdef1234567890abcdef12",
			fs,
		)
		err := uploader.Upload(context.Background(), "report.sarif")
		require.NoError(t, err)
	})
}
