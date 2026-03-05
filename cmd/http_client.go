package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type SarifUploader struct {
	Client          HTTPClient
	Url             string
	FS              afero.Fs
	Timeout         time.Duration
	APIKey          string
	GitHubRepoOwner string
	GitHubRepoName  string
	Revision        string
}

func NewSarifUploader(url, apiKey, repoOwner, repoName, revision string, fs afero.Fs) *SarifUploader {
	return &SarifUploader{
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
		Url:             url,
		FS:              fs,
		Timeout:         15 * time.Second,
		APIKey:          apiKey,
		GitHubRepoOwner: repoOwner,
		GitHubRepoName:  repoName,
		Revision:        revision,
	}
}

func (u *SarifUploader) Upload(ctx context.Context, filePath string) error {
	f, err := u.FS.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open SARIF file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Warn().Err(cerr).Msgf("warning: failed to close file %s", filePath)
		}
	}()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("github_repo_owner", u.GitHubRepoOwner); err != nil {
		return fmt.Errorf("failed to write github_repo_owner field: %w", err)
	}
	if err := writer.WriteField("github_repo_name", u.GitHubRepoName); err != nil {
		return fmt.Errorf("failed to write github_repo_name field: %w", err)
	}
	if err := writer.WriteField("revision", u.Revision); err != nil {
		return fmt.Errorf("failed to write revision field: %w", err)
	}

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return fmt.Errorf("failed to write file to form: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize multipart form: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.Url, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Access-Token", u.APIKey)

	resp, err := u.Client.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Warn().Err(cerr).Msg("warning: failed to close response body")
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("upload failed: status=%d body=%q", resp.StatusCode, string(body))
	}

	return nil
}
