# gh-pin 📌

[![test](https://github.com/grantbirki/gh-pin/actions/workflows/test.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/test.yml)
[![build](https://github.com/grantbirki/gh-pin/actions/workflows/build.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/build.yml)
[![lint](https://github.com/grantbirki/gh-pin/actions/workflows/lint.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/lint.yml)
[![golangci-lint](https://github.com/grantbirki/gh-pin/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/golangci-lint.yml)
[![release](https://github.com/grantbirki/gh-pin/actions/workflows/release.yml/badge.svg)](https://github.com/grantbirki/gh-pin/actions/workflows/release.yml)
![slsa-level3](docs/assets/slsa-level3.svg)

Pin Docker container images to an exact index digest for better build reproducibility.

## About ⭐

This project is a gh cli extension that is used to pin Docker container images to an exact index digest. This is useful for ensuring that builds are reproducible.

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

TODO

## How it Works 📚

The `gh-pin` CLI scans your project files and replaces mutable image tags with immutable digest references for better security and reproducibility.

### High-Level Process

1. **File Discovery**: Recursively scans directories to find Docker-related files:
   - `Dockerfile*` (any file starting with "Dockerfile")
   - `docker-compose.yml/yaml` files
   - Generic `.yml/.yaml` files containing Docker images (when using `--pervasive` flag)

2. **Image Detection**: Parses files to identify container image references:
   - Extracts `FROM` statements in Dockerfiles
   - Finds `image:` fields in Compose/YAML files
   - Skips images that already have digest pinning

3. **Registry Lookup**: For each unpinned image, performs efficient registry queries:
   - Makes HTTP `HEAD` requests to container registries (Docker Hub, GHCR, etc.)
   - Retrieves manifest metadata without downloading full manifest content
   - Extracts the SHA256 digest that uniquely identifies the image

4. **File Updates**: Replaces mutable tags with immutable digest references:
   - `nginx:latest` → `nginx@sha256:abc123...`
   - `ubuntu:20.04` → `ubuntu@sha256:def456...`
   - Preserves original formatting and indentation

### Benefits

- **Reproducible Builds**: Same digest always references the exact same image
- **Security**: Prevents supply chain attacks from tag manipulation
- **Efficiency**: Uses HEAD requests to minimize network bandwidth
- **Compatibility**: Works with all OCI-compatible registries

## Verifying Release Binaries 🔏

This project uses [goreleaser](https://goreleaser.com/) to build binaries and [actions/attest-build-provenance](https://github.com/actions/attest-build-provenance) to publish the provenance of the release.

You can verify the release binaries by following these steps:

1. Download a release from the [releases page](https://github.com/grantbirki/gh-pin/releases).
2. Verify it `gh attestation verify --owner github ~/Downloads/darwin-arm64` (an example for darwin-arm64).

---

Run `gh combine --help` for more information and full command/options usage.
