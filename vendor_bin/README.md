# Vendored Binaries

This directory contains vendored binaries that are used by the project. These binaries are included to ensure consistent builds and to avoid dependency on external tools.

All binaries must have proper attestations before they can be added here. For example:

```bash
$ gh attestation verify --repo goreleaser/goreleaser ./vendor_bin/goreleaser_2.11.2_amd64.deb
Loaded digest sha256:a9ddd4791bc0f862a665dbd7a8f077cf2861fc6d41c153c47252243cea3c1d67 for file://vendor_bin/goreleaser_2.11.2_amd64.deb
Loaded 1 attestation from GitHub API

The following policy criteria will be enforced:
- Predicate type must match:................ https://slsa.dev/provenance/v1
- Source Repository Owner URI must match:... https://github.com/goreleaser
- Source Repository URI must match:......... https://github.com/goreleaser/goreleaser
- Subject Alternative Name must match regex: (?i)^https://github.com/goreleaser/goreleaser/
- OIDC Issuer must match:................... https://token.actions.githubusercontent.com

✓ Verification succeeded!

The following 1 attestation matched the policy criteria

- Attestation #1
  - Build repo:..... goreleaser/goreleaser
  - Build workflow:. .github/workflows/release.yml@refs/tags/v2.11.2
  - Signer repo:.... goreleaser/goreleaser
  - Signer workflow: .github/workflows/release.yml@refs/tags/v2.11.2
```
