package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
			Timeout: 30 * time.Second,
		},
		Url:             url,
		FS:              fs,
		Timeout:         30 * time.Second,
		APIKey:          apiKey,
		GitHubRepoOwner: repoOwner,
		GitHubRepoName:  repoName,
		Revision:        revision,
	}
}

func (u *SarifUploader) Upload(ctx context.Context, filePath string) error {
	raw, err := afero.ReadFile(u.FS, filePath)
	if err != nil {
		return fmt.Errorf("failed to open SARIF file: %w", err)
	}

	payload, err := minifySARIF(raw)
	if err != nil {
		return fmt.Errorf("failed to prepare SARIF file %s: %w", filePath, err)
	}

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
	if _, err := part.Write(payload); err != nil {
		return fmt.Errorf("failed to write file to form: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize multipart form: %w", err)
	}

	// Gzip-compress the full multipart body. The server runs Echo's
	// middleware.Decompress() which transparently inflates the request
	// body before the SARIF handler reads it. Compressing the whole
	// request body (not just the file part) keeps the server side a
	// one-liner and lets us stay well under the 6 MiB Lambda Function
	// URL request payload limit — SARIF JSON typically compresses ~10x.
	gzBody, err := gzipBytes(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to gzip request body: %w", err)
	}

	log.Debug().
		Int("raw_bytes", len(raw)).
		Int("minified_bytes", len(payload)).
		Int("multipart_bytes", buf.Len()).
		Int("gzip_bytes", len(gzBody)).
		Msg("prepared SARIF upload payload")

	// SHA256 hash of the body as sent on the wire — required by CloudFront
	// OAC for POST requests to Lambda Function URLs with IAM auth. Must be
	// computed on the gzipped bytes, since that is what CloudFront forwards.
	hash := sha256.Sum256(gzBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.Url, bytes.NewReader(gzBody))
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("X-Access-Token", u.APIKey)
	req.Header.Set("x-amz-content-sha256", hex.EncodeToString(hash[:]))

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
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("upload failed: status=%d body=%q", resp.StatusCode, string(respBody))
	}

	return nil
}

// minifySARIF returns the SARIF document with insignificant whitespace
// removed. Scanner output is usually pretty-printed; minifying typically
// shaves 20–40% before gzip even kicks in. If the input is not valid
// JSON, the original bytes are returned so the server can surface a
// clear parse error instead of our client silently rewriting payloads.
func minifySARIF(src []byte) ([]byte, error) {
	var doc any
	if err := json.Unmarshal(src, &doc); err != nil {
		return src, nil //nolint:nilerr // fall through to server-side validation
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal minified SARIF: %w", err)
	}
	return out, nil
}

func gzipBytes(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(src); err != nil {
		_ = gw.Close()
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
