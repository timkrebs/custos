# Changelog

All notable changes to **custos** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Changes that are not yet released live under the `Unreleased` section and are
moved under a version header when a release is cut.

## [Unreleased]

- n/a

## [0.1.0] - 2026-04-13

First working release — **"It works offline."**

custos can now load a YAML test spec, parse referenced HCL policy files,
evaluate each test case through the offline policy engine, and report
colored pass/fail results in the terminal.

### Added
- **HCL policy parser** (`pkg/parser`) — parses Vault ACL policy files with
  full field support: `capabilities`, `allowed_parameters`, `denied_parameters`,
  `required_parameters`, `min_wrapping_ttl`, `max_wrapping_ttl`, and glob
  patterns (`*`, `+`).
- **YAML test spec loader** (`pkg/spec`) — parses and validates test
  specification files with suite name, policy references, test cases
  (path, capabilities, expected result), and an optional `analyze` section.
- **Offline evaluation engine** (`pkg/evaluator`) — determines whether a
  path + capabilities combination is allowed or denied by a set of parsed
  policies. Replicates Vault's ACL evaluation logic:
  - Exact path matching takes precedence over glob matching.
  - Longest-prefix-match: more specific rules win.
  - Deny capability overrides allow from any policy.
  - Multi-policy composition: capabilities are unioned across policies.
  - Implicit deny: no matching rule means deny.
  - Support for `*` (prefix glob) and `+` (single-segment wildcard) patterns.
  - Returns explanation metadata (matched policy, rule path, reason).
- **Terminal reporter** (`pkg/reporter`) — colored pass/fail output using
  `fatih/color`. Respects `NO_COLOR` environment variable. Verbose mode
  (`-v`) shows per-test evaluation trace.
- **`custos test` command** — end-to-end pipeline: loads spec, parses
  policies, runs evaluator, reports results. Exit code 0 on all pass,
  1 on any failure. Supports `--fail-on-warn` and `-v` / `--verbose` flags.
- **`custos version` command** — prints version, git commit, tree state,
  build date, Go version, and platform. Supports `--json` flag.
- **Test fixtures and examples** — four HCL policy files and three YAML
  test specs in `testdata/` that serve as both test fixtures and user-facing
  examples:
  - `policies/payment-svc.hcl` — service-specific policy (README example)
  - `policies/admin.hcl` — broad admin policy with sys/ access
  - `policies/readonly.hcl` — read-only policy for monitoring
  - `policies/overprivileged.hcl` — intentionally dangerous policy for
    future scanner testing
  - `specs/payment-svc.spec.yaml` — 10 tests covering service boundaries
  - `specs/admin.spec.yaml` — 14 tests covering admin access and boundaries
  - `specs/composed.spec.yaml` — 13 tests demonstrating multi-policy
    composition with deny-override semantics
- **Build and release infrastructure:**
  - `.build/build.sh` — cross-compile binaries for all 6 platforms locally.
  - `.build/install.sh` — one-line installer that downloads, verifies
    checksums, and installs the binary (curl-pipe-bash pattern).
  - `.release/docker/Dockerfile` — multi-stage Docker image based on Alpine
    with non-root user.
  - `.release/security-scan.sh` — pre-release security scan script
    (govulncheck, staticcheck, go vet).
  - `.release/release-metadata.hcl` — release configuration metadata.
  - GoReleaser enhanced with Homebrew tap automation and Docker image
    publishing to `ghcr.io/timkrebs/custos`.
  - Release workflow updated with Docker Buildx and GHCR login.
- **Installation methods:** `go install`, Homebrew (`brew install
  timkrebs/tap/custos`), release binaries, Docker
  (`docker run ghcr.io/timkrebs/custos`), and curl installer script.
- **CI/CD** — GitHub Actions workflows for testing, auditing, and releasing.
- **Project scaffolding** — `CODE_OF_CONDUCT.md`, `SECURITY.md`,
  `MAINTAINERS.md`, `CONTRIBUTING.md`, GitHub issue and PR templates,
  `CODEOWNERS`, Dependabot configuration.

[Unreleased]: https://github.com/timkrebs/custos/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/timkrebs/custos/releases/tag/v0.1.0
