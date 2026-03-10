# upload-sarif

A CLI tool and GitHub Action to upload SARIF (Static Analysis Results Interchange Format) files to the Neuralsec API for security analysis and tracking.

For complete documentation, visit [docs.neuralsec.io](https://docs.neuralsec.io/).

## Installation

### GitHub Action (Recommended)

The easiest way to use upload-sarif in CI/CD is via the GitHub Action:

```yaml
- name: Upload SARIF to Neuralsec
  uses: neuralsec-io/upload-sarif@v1
  with:
    api-key: ${{ secrets.NEURALSEC_API_KEY }}
    path: results.sarif
```

### Pre-built Binaries

Download pre-built binaries from the [Releases](https://github.com/neuralsec-io/upload-sarif/releases) page.

Available for:
- Linux (x86_64, arm64, i386)
- macOS (x86_64, arm64)
- Windows (x86_64, arm64, i386)

### Docker

```bash
docker pull ghcr.io/neuralsec-io/upload-sarif:latest
```

### Build from Source

```bash
go install github.com/neuralsec-io/upload-sarif/cmd/upload-sarif@latest
```

## Usage

### GitHub Action

```yaml
name: Security Scan

on: [push, pull_request]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run security scanner
        run: |
          # Example: run your SAST tool that generates SARIF
          semgrep --sarif -o results.sarif .

      - name: Upload SARIF to Neuralsec
        uses: neuralsec-io/upload-sarif@v1
        with:
          api-key: ${{ secrets.NEURALSEC_API_KEY }}
          path: results.sarif
```

#### Action Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `api-key` | Yes | - | API key for authentication with Neuralsec API |
| `path` | Yes | - | Path to SARIF file or directory containing SARIF files |
| `version` | No | `latest` | Version of the CLI to use (e.g., `v1.0.0`) |

The action automatically detects repository context from GitHub:
- Repository owner and name
- Commit SHA (revision)

### CLI

```bash
upload-sarif \
  --github-repo-owner <owner> \
  --github-repo-name <repo> \
  --revision <commit-sha> \
  --path <sarif-file-or-directory> \
  --api-key <your-api-key>
```

#### CLI Flags

| Flag | Environment Variable | Required | Description |
|------|---------------------|----------|-------------|
| `--github-repo-owner` | `GITHUB_REPO_OWNER` | Yes | Repository owner name |
| `--github-repo-name` | `GITHUB_REPO_NAME` | Yes | Repository name |
| `--revision` | `REVISION` | Yes | Git commit SHA |
| `--path` | - | Yes | Path to SARIF file or directory |
| `--api-key` | `API_KEY` | Yes | API key for authentication |
| `--log-level` | `LOG_LEVEL` | No | Log level (debug, info, warn, error) |
| `--version` | - | No | Display version and exit |

#### Examples

Upload a single SARIF file:

```bash
upload-sarif \
  --github-repo-owner myorg \
  --github-repo-name myrepo \
  --revision abc123def456 \
  --path ./results.sarif \
  --api-key $NEURALSEC_API_KEY
```

Upload all SARIF files in a directory:

```bash
upload-sarif \
  --github-repo-owner myorg \
  --github-repo-name myrepo \
  --revision abc123def456 \
  --path ./sarif-reports/ \
  --api-key $NEURALSEC_API_KEY
```

Using environment variables:

```bash
export GITHUB_REPO_OWNER=myorg
export GITHUB_REPO_NAME=myrepo
export REVISION=abc123def456
export API_KEY=your-api-key

upload-sarif --path ./results.sarif
```

### Docker

```bash
docker run --rm \
  -v $(pwd):/workspace \
  -e GITHUB_REPO_OWNER=myorg \
  -e GITHUB_REPO_NAME=myrepo \
  -e REVISION=abc123def456 \
  -e API_KEY=your-api-key \
  ghcr.io/neuralsec-io/upload-sarif:latest \
  --path /workspace/results.sarif
```

## Verifying Releases

All releases are signed using [Sigstore cosign](https://github.com/sigstore/cosign) for supply chain security.

### Verify Checksums Signature

Download the checksums file and its sigstore bundle:

```bash
VERSION=v1.0.0
curl -sLO "https://github.com/neuralsec-io/upload-sarif/releases/download/${VERSION}/checksums.txt"
curl -sLO "https://github.com/neuralsec-io/upload-sarif/releases/download/${VERSION}/checksums.txt.sigstore.json"
```

Verify the signature:

```bash
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  checksums.txt
```

### Verify Binary Checksum

After verifying the signature, verify the binary checksum:

```bash
# Download the binary (example for Linux x86_64)
curl -sLO "https://github.com/neuralsec-io/upload-sarif/releases/download/${VERSION}/upload-sarif_Linux_x86_64.tar.gz"

# Verify checksum
sha256sum -c checksums.txt --ignore-missing
```

### Verify Docker Image

Docker images are also signed with cosign:

```bash
cosign verify ghcr.io/neuralsec-io/upload-sarif:v1.0.0
```

## Security

- All release artifacts are signed using keyless signing via Sigstore
- Checksums are provided for all binaries
- Docker images use a minimal distroless base image
- SARIF files are uploaded via multipart form to the Neuralsec API

## License

See [LICENSE](LICENSE) for details.
