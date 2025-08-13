# gh-pin 📌

[![test](https://github.com/grantbirki/gh-pin/actions/workflows/test.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/test.yml)
[![build](https://github.com/grantbirki/gh-pin/actions/workflows/build.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/build.yml)
[![lint](https://github.com/grantbirki/gh-pin/actions/workflows/lint.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/lint.yml)
[![acceptance](https://github.com/grantbirki/gh-pin/actions/workflows/acceptance.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/acceptance.yml)
[![golangci-lint](https://github.com/grantbirki/gh-pin/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/golangci-lint.yml)
[![release](https://github.com/grantbirki/gh-pin/actions/workflows/release.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/release.yml)
![slsa-level3](docs/assets/slsa-level3.svg)

Pin Docker container images and GitHub Actions to exact digests for better build reproducibility.

![gh-pin](docs/assets/gh-pin.png)

## About ⭐

This project is a [`gh cli`](https://github.com/cli/cli) extension that is used to pin Docker container images and GitHub Actions to exact digests. This is useful for ensuring that builds are reproducible and secure.

Container images referenced by mutable tags (like `latest` or `v1.0`) and GitHub Actions referenced by mutable tags (like `v4` or `main`) can change over time, leading to inconsistent builds and potential security vulnerabilities. When a tag is updated to point to a new version, your builds may suddenly start using different dependencies or even malicious content without your knowledge.

The `gh pin` tool solves this by automatically converting mutable references to immutable digest references. Instead of `ubuntu:latest`, your files will reference `ubuntu@sha256:abc123...`, and instead of `actions/checkout@v5`, your workflows will reference `actions/checkout@sha123abc`. This ensures that the exact same versions are used every time. This approach follows security best practices recommended by organizations like the [CNCF](https://www.cncf.io/online-programs/cloud-native-live-automate-pinning-github-actions-and-container-images-to-their-digests/) and [SLSA](https://slsa.dev/) for supply chain security.

All updated pins (Dependabot + Actions) will work out of the box with Dependabot!

> Moving towards immutable image references lives in the same ecosystem as [Hermetic Builds](https://software.birki.io/posts/hermetic-builds/) which is a topic I am passionate about and a key reason for building this CLI.

## Installation 💻

Install this gh cli extension by running the following command:

```bash
gh extension install grantbirki/gh-pin
```

### Upgrading 📦

You can upgrade this extension by running the following command:

```bash
gh ext upgrade pin
```

## Usage 🚀

### Basic Usage

Pin images in a specific Dockerfile:

```bash
gh pin Dockerfile
```

Pin images in a specific Docker compose file:

```bash
gh pin docker-compose.yml
```

Pin GitHub Actions in a workflow file:

```bash
gh pin .github/workflows/ci.yml
```

Pin all Docker images and GitHub Actions in the current directory and subdirectories:

```bash
gh pin .
```

> Note: The `gh pin` command works best when you run it from the root of your repository when using `gh pin .`

Pin all images and actions in a specific directory and its subdirectories:

```bash
gh pin /path/to/project
```

### Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `--algo` | Digest algorithm to check for (sha256, sha512, etc.) | `sha256` |
| `--dry-run` | Preview changes without writing files | `false` |
| `--expand-registry` | Expand short image names to fully qualified registry names | `false` |
| `--no-color` | Disable colored output | `false` |
| `--pervasive` | Scan all YAML files, not just docker-compose files | `false` |
| `--recursive` | Scan directories recursively | `true` |
| `--version` | Show version information | `false` |

### Examples

**Preview changes without modifying files:**

```bash
gh pin --dry-run
```

**Pin images and expand registry names:**

```bash
# Without --expand-registry (default):
# ubuntu:latest → ubuntu@sha256:abc123...

# With --expand-registry:
# ubuntu:latest → docker.io/library/ubuntu:latest@sha256:abc123...
gh pin --expand-registry
```

**Scan all YAML files (including Kubernetes manifests, CI configs, etc.):**

```bash
gh pin --pervasive
```

**Combine multiple options:**

```bash
gh pin --dry-run --pervasive --expand-registry /path/to/project
```

### Supported File Types

| File Type | Detection Pattern | Description |
|-----------|------------------|-------------|
| **Dockerfiles** | `Dockerfile*` | Any file starting with "Dockerfile" (ex: `Dockerfile`, `Dockerfile.test`, `Dockerfile.dev`, etc) |
| **Docker Compose** | `docker-compose.yml`, `docker-compose.yaml` | Standard Docker Compose files |
| **GitHub Actions** | `.github/workflows/*.yml`, `.github/workflows/*.yaml` | GitHub Actions workflow files |
| **Generic YAML** | `*.yml`, `*.yaml` | When using `--pervasive` flag |

### Force Mode

You can force the tool to only process specific file types:

```bash
# Only process Docker-related files
gh pin --mode=docker

# Only process GitHub Actions workflows
gh pin --mode=actions
```

### Output Examples

```bash
$ gh pin --dry-run
📌 [DOCKERFILE] ubuntu:latest → ubuntu@sha256:7c06e91f61fa88c08cc74f7e1b7c69ae24910d745357e0dfe1d2c0322aaf20f9
📌 [COMPOSE] nginx:alpine → nginx@sha256:2d194b9da5f3b2f19d8b03b48d36c3f8af53e24b96b8c48a82db8d7b6e6e4c6a
📌 [ACTIONS] actions/checkout@v5 → actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
```

### Exit Codes

- `0`: Success - all resources processed successfully
- `1`: Error - failed to process one or more resources

## How it Works 📚

The `gh-pin` CLI scans your project files and replaces mutable references with immutable digest references for better security and reproducibility.

### High-Level Process

1. **File Discovery**: Recursively scans directories to find supported files:
   - `Dockerfile*` (any file starting with "Dockerfile")
   - `docker-compose.yml/yaml` files
   - `.github/workflows/*.yml` GitHub Actions workflow files
   - Generic `.yml/.yaml` files (when using `--pervasive` flag)

2. **Reference Detection**: Parses files to identify mutable references:
   - Extracts `FROM` statements in Dockerfiles
   - Finds `image:` fields in Compose/YAML files
   - Extracts `uses:` statements in GitHub Actions workflows
   - Skips references that already have digest/SHA pinning

3. **Resolution**: For each unpinned reference, performs API queries:
   - **Container Images**: Makes HTTP `HEAD` requests to container registries (Docker Hub, GHCR, etc.)
   - **GitHub Actions**: Makes API requests to GitHub to resolve tags to commit SHAs
   - Retrieves digest/SHA that uniquely identifies the version

4. **File Updates**: Replaces mutable references with immutable digest references:
   - `nginx:latest` → `nginx@sha256:abc123...`
   - `ubuntu:20.04` → `ubuntu@sha256:def456...`
   - `actions/checkout@v5` → `actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8`
   - Preserves original formatting and indentation
   - Supports comment-based pinning (e.g., `# pin@v5`)

### Benefits

- **Reproducible Builds**: Same digest/SHA always references the exact same version
- **Security**: Prevents supply chain attacks from tag manipulation
- **Efficiency**: Uses HEAD requests to minimize network bandwidth
- **Compatibility**: Works with all OCI-compatible registries and GitHub Actions
- **Comment Support**: Supports `# pin@v5` style comments for explicit version control

## Prior Art, Inspiration, and Alternatives

- [mheap/pin-github-action](https://github.com/mheap/pin-github-action)
- Follow a guide like this from [Step Security](https://www.stepsecurity.io/blog/pinning-github-actions-for-enhanced-security-a-complete-guide) and manually update Actions pins then use dependabot

You can pull Docker digests manually by pulling down the entire image (can be slow) like this:

```bash
TAG="postgres:15"
docker pull "$TAG"
DIGEST=$(docker inspect --format='{{index .RepoDigests 0}}' "$TAG")
echo "$TAG -> $DIGEST"
```

You could also do something like this and manually edit your Docker / Docker-Compose files:

```bash
regctl image digest postgres:15
# outputs: sha256:9b2a...
```

## Verifying Release Binaries 🔏

This project uses [goreleaser](https://goreleaser.com/) to build binaries and [actions/attest-build-provenance](https://github.com/actions/attest-build-provenance) to publish the provenance of the release.

You can verify the release binaries by following these steps:

1. Download a release from the [releases page](https://github.com/grantbirki/gh-pin/releases).
2. Verify it `gh attestation verify --owner github ~/Downloads/darwin-arm64` (an example for darwin-arm64).

---

Run `gh pin --help` for more information and full command/options usage.
