# Agents

This repository is `gh-pin`, a Go-based GitHub CLI extension for pinning mutable supply-chain references to immutable identifiers. It pins Docker image references in Dockerfiles and Docker Compose/YAML files to OCI digests, and pins GitHub Actions `uses:` references to exact commit SHAs. The goal is reproducibility, stability, and supply-chain hardening while keeping files readable and compatible with Dependabot.

## Project Shape

- `cmd/gh-pin/main.go`: CLI entry point, flag parsing, `processor.ProcessorConfig` construction, and routing each requested file or directory into the scanner.
- `internal/scanner`: file discovery and type detection. It decides whether a path is a Dockerfile, Docker Compose file, GitHub Actions workflow, or generic YAML file when `--pervasive` is enabled.
- `internal/processor`: the core rewriting logic for Dockerfiles, Compose/YAML, GitHub Actions, image digest resolution, output formatting, shared file-processing helpers, and runtime config.
- `internal/version`: build metadata and version string handling. GoReleaser injects tag, commit, and build time via ldflags; local builds can fall back to Go VCS build info.
- `script/`: scripts-to-rule-them-all entrypoints for bootstrap, test, lint, build, update, acceptance, and release.
- `.github/workflows`: CI, acceptance, release, and provenance workflows.
- `vendor/`: committed Go module dependencies. The normal development/test/build flow must work from vendored code.
- `vendor_bin/`: attested vendored tool packages. Currently this is for GoReleaser only.
- `test/fixtures`: sample Dockerfile, Compose, and workflow inputs for acceptance-style testing.

## CLI Behavior

Supported targets:

- Dockerfiles: `FROM` image references in `Dockerfile`, `Dockerfile.*`, and `*.dockerfile` during directory scans. Explicit single-file processing is intentionally more permissive for Dockerfile-like names.
- Docker Compose: `image:` references in `docker-compose.yml` and `docker-compose.yaml`.
- GitHub Actions: `uses:` references in `.github/workflows/*.yml` and `.github/workflows/*.yaml`.
- Generic YAML: only with `--pervasive`, and only when structure detection identifies Compose or Actions content.

Important flags:

- `--dry-run`: report changes without writing files.
- `--recursive`: scan directories recursively; defaults to true.
- `--pervasive`: inspect all YAML files for Compose/Actions-shaped content.
- `--expand-registry`: expand short Docker image names to fully qualified registry names.
- `--mode=docker|actions`: force processing to only Docker/Compose or Actions files.
- `--quiet`: suppress informational "already pinned" messages.
- `--platform=linux/amd64`: resolve platform-specific manifest digests when available.
- `--algo=sha256`: digest algorithm used to decide whether an image is already pinned.
- `--no-color`: disable colored terminal output.

## Implementation Notes

- Docker image resolution uses `github.com/regclient/regclient`. Default behavior pins to the image/index digest. When `--platform` is set, the processor tries to select that platform's manifest digest and falls back to the index digest with a warning if it cannot.
- GitHub Action resolution uses `github.com/cli/go-gh/v2/pkg/api` through `DefaultGitHubResolver`. Keep the `GitHubResolver` interface: tests depend on dependency injection instead of network calls.
- File rewrites are intentionally line-oriented. Preserve indentation, comments, ordering, and file mode; do not reserialize whole Docker/YAML files unless there is a strong reason.
- Compose processing validates YAML with `goccy/go-yaml` but rewrites `image:` lines directly to avoid formatting churn.
- GitHub Actions pin comments use `# pin@<ref>`. If an action is rewritten from a tag to a SHA, preserve or add the original tag as a pin comment so Dependabot can keep updates understandable.
- Docker and Compose output should use Docker-compatible `tag@sha256:<digest>` style references.
- Already-pinned references should be skipped and reported unless `--quiet` is set.
- Errors resolving a single image/action should warn and leave that line unchanged rather than corrupting the file.

## Development Flow

Use the repo scripts, not ad hoc commands, unless you are doing a narrow investigation.

- Bootstrap: `script/bootstrap`
- Test: `script/test`
- Lint/format: `script/lint`
- Build: `script/build`
- Fast local build: `script/build --single-target`
- Acceptance: `script/acceptance`
- Dependency update: `script/update` or `script/update --all`
- Release tag helper: `script/release`

`script/env` centralizes the hermetic Go environment. It sets `GOPROXY=off` and `GOSUMDB=off` for normal script execution, creates `vendor/` if needed, and derives `PROJECT_MODULE_PATH` from `go list -m`.

`script/bootstrap` must verify vendored dependency resolution with `go list -mod=vendor -deps ./...`. In CI it installs GoReleaser from `vendor_bin/` if needed.

`script/test` runs:

```bash
go test -mod=vendor -race -count 10 -v -cover -coverprofile=coverage.out ./...
```

`script/lint` is intentionally simple and mirrors the local Go template style: it runs `go fmt -mod=vendor ./...`, skips `go mod tidy` while `GOPROXY=off`, and in CI fails if formatting leaves a dirty diff.

Do not add a separate Go lint toolchain back to this repository.

## Hermetic/Vendored Standards

- Keep Go dependencies committed under `vendor/`.
- Normal bootstrap, test, lint, and build commands should not require module network access.
- `script/update` is the sanctioned path for dependency refreshes. It temporarily enables the Go proxy, updates modules, runs `go mod tidy`, vendors dependencies, verifies modules, and restores the hermetic environment on exit.
- GoReleaser builds use `-mod=vendor`, `GOPROXY=off`, `GOSUMDB=off`, `CGO_ENABLED=0`, `-trimpath`, and reproducibility metadata.
- Do not add new runtime dependencies casually. Prefer standard-library code and existing vendored packages where practical.
- If adding a vendored binary under `vendor_bin/`, include a clear attestation verification path in `vendor_bin/README.md`.

## Testing Expectations

- Add or update unit tests for behavior changes. Prefer table-driven tests using the standard `testing` package.
- Avoid networked unit tests. Mock GitHub resolution through `GitHubResolver`; keep real registry/API behavior in acceptance tests or carefully isolated integration checks.
- Cover edge cases around comments, indentation, malformed input, already-pinned refs, dry-run behavior, quiet mode, forced mode, platform fallback, and non-standard file names.
- For scanner changes, test both directory scanning and explicit single-file processing; their filename rules intentionally differ.
- For rewrite changes, assert that dry-run leaves files unchanged and non-dry-run preserves file permissions.
- For behavior that touches Docker registry resolution, run `script/acceptance` when feasible because unit coverage intentionally avoids most network calls.

## CI and Release

- Regular CI jobs are split by concern: `test`, `build`, `lint`, and `acceptance`.
- CI should checkout with `persist-credentials: false`, set up Go from `go.mod`, run `script/bootstrap`, then run the matching script.
- The release workflow is tag-driven and gated to `grantbirki/gh-pin`.
- Releases build with `script/build --release`, upload `dist/`, generate build provenance with `actions/attest-build-provenance`, and verify artifacts with `gh attestation verify`.
- Keep workflow permissions minimal and scoped to the job that needs them.

## Code Standards

- Follow idiomatic Go and the existing package boundaries.
- Prefer small, direct functions over unnecessary abstractions, but keep dependency injection where it prevents network or filesystem coupling in tests.
- Preserve the current script-first, vendored, hermetic project style inherited from the Go template repo.
- Keep user-facing CLI output concise, stable, and useful for automation logs.
- Avoid broad rewrites or formatting churn in user files; this tool is meant to make minimal supply-chain pinning edits.
- Keep docs and README examples aligned with actual flags, scripts, workflows, and output format.
