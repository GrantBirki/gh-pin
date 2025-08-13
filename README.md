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

Pin images in a specific Dockerfile using an exact platform:

```bash
gh pin --platform=linux/amd64 Dockerfile
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
| `--platform` | Target specific platform architecture (e.g., `linux/amd64`, `linux/arm64`) | (uses index digest) |
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

**Pin to specific platform architecture:**

```bash
# Pin for linux/amd64 architecture specifically
gh pin --platform=linux/amd64 docker-compose.yml

# Pin for ARM64 (Apple Silicon, AWS Graviton instances)
gh pin --platform=linux/arm64 .
```

### Supported File Types

| File Type | Detection Pattern | Description |
|-----------|------------------|-------------|
| **Dockerfiles** | `Dockerfile*` | Any file starting with "Dockerfile" (ex: `Dockerfile`, `Dockerfile.test`, `Dockerfile.dev`, etc) |
| **Docker Compose** | `docker-compose.yml`, `docker-compose.yaml` | Standard Docker Compose files |
| **GitHub Actions** | `.github/workflows/*.yml`, `.github/workflows/*.yaml` | GitHub Actions workflow files |
| **Generic YAML** | `*.yml`, `*.yaml` | When using `--pervasive` flag |

### Platform-Specific Pinning

The `--platform` flag allows you to pin images to platform-specific manifest digests instead of multi-platform index digests.

**Index Digests (Default Behavior):**

```bash
# Pins to the multi-platform index digest
gh pin Dockerfile
# Result: FROM nginx@sha256:abc123... # pin@nginx:latest
```

**Platform-Specific Manifest Digests:**

Increased determinism by pinning to a specific platform's manifest digest:

```bash
# Pins to the linux/amd64 specific manifest digest
gh pin --platform=linux/amd64 Dockerfile
# Result: FROM nginx:latest@sha256:def456...
```

#### When to Use Index vs Platform-Specific Digests

**Use Index Digests (Default) when:**

- You want maximum compatibility across different architectures
- Your build system automatically selects the correct platform
- You're building multi-platform images that should work everywhere
- You want human-readable comments showing the original tag

**Use Platform-Specific Digests (`--platform`) when:**

- You need deterministic builds for a specific architecture
- You're building for embedded systems or specific hardware
- You want to ensure the exact same binary artifacts every time
- You're troubleshooting platform-specific issues
- Your deployment targets only run on specific architectures

#### Supported Platforms

Common platform specifications include:

- `linux/amd64` - 64-bit x86 Linux (most common)
- `linux/arm64` - 64-bit ARM Linux (Apple Silicon, AWS Graviton, etc.)
- `linux/arm/v7` - 32-bit ARM Linux
- `linux/arm64/v8` - 64-bit ARM Linux (Raspberry Pi, etc.)
- `windows/amd64` - 64-bit x86 Windows
- `darwin/amd64` - 64-bit x86 macOS
- `darwin/arm64` - 64-bit ARM macOS (Apple Silicon)

#### Platform Examples

**Target specific architecture:**

```bash
gh pin --platform=linux/amd64 docker-compose.yml

gh pin --platform=linux/arm64 Dockerfile

gh pin --platform=linux/arm/v7 docker-compose.yml
```

**Error handling:**

```bash
# If the provided platform doesn't exist it will gracefully falls back to index digest
gh pin --platform=invalid/platform Dockerfile
# Warning: Could not find manifest for platform invalid/platform. Falling back to index digest.
```

#### Index vs Platform Digest Comparison

| Aspect | Index Digest (Default) | Platform-Specific Digest |
|--------|------------------------|--------------------------|
| **Compatibility** | Works across all supported platforms | Works only on specified platform |
| **Build Reproducibility** | Good (platform selected at runtime) | Excellent (exact binary artifacts) |
| **Output Format** | `image@sha256:hash # pin@tag` | `image:tag@sha256:hash` |
| **Comments** | Shows human-readable original tag | Clean format, no comments |
| **Use Case** | General development, CI/CD | Specific deployments, debugging, hardened CI/CD environment |
| **Fallback** | N/A | Falls back to index if platform unavailable |

### Force Mode

You can force the tool to only process specific file types:

```bash
# Only process Docker-related files
gh pin --mode=docker

# Only process GitHub Actions workflows
gh pin --mode=actions
```

### Output Examples

**Default behavior (index digests with human-readable comments):**

```bash
$ gh pin --dry-run
📌 [DOCKERFILE] ubuntu:latest → ubuntu@sha256:7c06e91f61fa88c08cc74f7e1b7c69ae24910d745357e0dfe1d2c0322aaf20f9 # pin@ubuntu:latest
📌 [COMPOSE] nginx:alpine → nginx@sha256:2d194b9da5f3b2f19d8b03b48d36c3f8af53e24b96b8c48a82db8d7b6e6e4c6a # pin@nginx:alpine
📌 [ACTIONS] actions/checkout@v5 → actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
```

**Platform-specific behavior (clean format, no comments):**

```bash
$ gh pin --platform=linux/amd64 --dry-run
📌 [DOCKERFILE] ubuntu:latest → ubuntu:latest@sha256:1b8d8ff4777f36f19bfe73ee4df61e3a0b789caeff29caa019539ec7c9a57f95
📌 [COMPOSE] nginx:alpine → nginx:alpine@sha256:a97eb9ecc708c8aa715ddc4b375e7c130bd32e0bce17c74b4f8c3a90e8338e14
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

### Understanding Index vs Manifest Digests

When pinning container images, `gh-pin` can target two different types of digests:

**Index Digests (Default):**

- Point to a **manifest list/index** that contains multiple platform-specific manifests
- Allow Docker/container runtimes to automatically select the correct platform at pull time
- Provide maximum compatibility across different architectures
- Example: `nginx@sha256:abc123...` works on AMD64, ARM64, etc.

**Platform-Specific Manifest Digests (`--platform` flag):**

- Point directly to a **single platform's manifest**
- Ensure you get exactly the same binary artifacts every time
- Provide deterministic builds for specific architectures
- Example: `nginx:latest@sha256:def456...` only works on the specified platform

**Visual Example:**

```text
Container Registry:
├── nginx:latest (index/manifest list)
│   ├── linux/amd64 → manifest digest: sha256:aaa111...
│   ├── linux/arm64 → manifest digest: sha256:bbb222...
│   └── linux/arm/v7 → manifest digest: sha256:ccc333...
└── Index digest: sha256:index123...

Default: nginx@sha256:index123... (points to manifest list)
--platform=linux/amd64: nginx:latest@sha256:aaa111... (points to specific manifest)
```

1. **File Updates**: Replaces mutable references with immutable digest references:
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
