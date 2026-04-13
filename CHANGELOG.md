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
- **Test fixtures** — example policy (`testdata/policies/payment-svc.hcl`)
  and spec (`testdata/specs/payment-svc.spec.yaml`) with 10 test cases.
- **CI/CD** — GitHub Actions workflows for testing, auditing, and releasing.
  GoReleaser configuration for cross-platform builds.
- **Project scaffolding** — `CODE_OF_CONDUCT.md`, `SECURITY.md`,
  `MAINTAINERS.md`, `CONTRIBUTING.md`, GitHub issue and PR templates,
  `CODEOWNERS`, Dependabot configuration.

[Unreleased]: https://github.com/timkrebs/custos/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/timkrebs/custos/releases/tag/v0.1.0
