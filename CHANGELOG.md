# Changelog

All notable changes to **custos** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Changes that are not yet released live under the `Unreleased` section and are
moved under a version header when a release is cut.

## [Unreleased]

### Added

- **JSON reporter** (`pkg/reporter/json.go`) — new output format for
  programmatic consumption. Emits a stable, versioned
  (`schema_version: "1.0"`) document containing the suite name,
  duration, summary counts, top-level warnings, and a `results` array
  with per-test `name`, `path`, `capabilities`, `expected`, `actual`,
  `pass`, `duration_seconds`, `explanation`, `matched_rule`, and a full
  `composed` provenance sub-object (`granted`, `denied`,
  `contributions`, `denied_by`). Array fields are always emitted as
  arrays (never null) so `jq` filters do not need null-checking, and
  field order is deterministic because the document is built from
  Go structs. Pretty-printed by default; `--compact` emits single-line
  output for line-oriented tools (`custos test -f spec.yaml
  --format=json --compact >> results.ndjson`).
- **`--compact` flag on `custos test`** — toggles compact single-line
  output when `--format=json` is set. Ignored as a no-op for other
  formats so users can add it preemptively without error.
- **JUnit XML reporter** (`pkg/reporter/junit.go`) — new output format for
  CI/CD systems. Emits a standard `<testsuites>` / `<testsuite>` /
  `<testcase>` / `<failure>` document with per-test timing (microsecond
  precision), ISO 8601 timestamps, and XML-escaped content. Failure
  elements carry a short dashboard message plus a chardata body with
  expected/got, path, capabilities, matched rule, explanation, and
  multi-policy contribution provenance.
- **`--format` flag on `custos test`** — selects the output format.
  Accepts `terminal` (default), `junit`, or `json`. Unknown values fail
  fast with an error listing supported options. JUnit output is written
  as a complete XML document to stdout so it can be redirected to a
  file (`custos test -f spec.yaml --format=junit > results.xml`) and
  consumed by dorny/test-reporter, Jenkins, GitLab, and GitHub Actions
  test reporters without further processing.
- **`reporter.Reporter` interface + `reporter.New` factory** — unified
  entry point for format selection; already hosts terminal, JUnit, and
  JSON reporters, and future formats (SARIF) can be added without
  touching the CLI.
- **Per-test timing** (`evaluator.TestResult.Duration`,
  `evaluator.SuiteResult.Duration`) — measured by `EvaluateSuite` via
  `time.Now()` around each `Evaluate` call. Consumed by JUnit; ignored
  by the terminal reporter.
- **Multi-policy composition engine** (`pkg/evaluator/composer.go`) —
  new `Compose(policies, requestPath) Composed` primitive implementing
  Vault's real composition semantics: per-policy longest-prefix match,
  union across policies, deny override. Fixes two latent correctness
  bugs in the pre-composer global-best selector and exposes full
  provenance (`Composed.GrantedBy`, `DeniedBy`, `Contributions`).
- **Multi-policy provenance rendering in the terminal reporter** —
  failing tests that involve two or more contributing policies now
  display a compact `contributions:` block listing each policy, its
  matched rule, and either the capabilities it granted or a DENIED
  marker. Verbose mode (`-v`) renders the same block on passing tests.
- **Strict YAML decoding for test specs** — `KnownFields(true)` on the
  spec loader rejects unknown top-level fields and typos (e.g.
  `capabilties:`) instead of silently ignoring them.
- **Rich HCL parser diagnostics** (`pkg/parser.ParsePolicyDiag`,
  `ParsePolicyFileDiag`) — new functions return `hcl.Diagnostics` plus
  the underlying `hclparse.Parser` so callers can render file:line:col
  source-annotated errors via `hcl.NewDiagnosticTextWriter`.
- **Typed capability vocabulary** (`pkg/vaultpolicy`) — new package
  owning the canonical Vault capability set (`Capabilities` map and
  `IsValidCapability` helper) so parser and spec validator share one
  source of truth.
- **Schema versioning for test specs** — optional top-level `version`
  field accepting `v1` or empty for back-compat. Unknown versions are
  rejected at load time.
- **Typed `Percentage` for `min_coverage`** — custom YAML unmarshaler
  accepts numeric (`80`, `80.5`) and string-with-percent (`"80%"`)
  forms. Invalid values fail at decode time, range `[0, 100]` is
  enforced in validation.
- **Scalar-form policy references** — `policies:` entries now accept
  both `- foo.hcl` and `- path: foo.hcl` via a custom YAML unmarshaler.
- **Error aggregation in spec validation** — collects every validation
  error via `errors.Join` so users see all problems in one run.
  Duplicate test-name detection, `AnalyzeCheck` field validation
  (`check`, `severity`, `min_coverage`) included.
- **Per-attribute source ranges on HCL parameter errors** — type
  mismatches in `allowed_parameters`, `denied_parameters`, and
  `required_parameters` now return `hcl.Diagnostics` with
  file:line:col instead of panicking on non-string elements.
- **Codecov integration** — coverage reports are uploaded from CI for
  every push and pull request.

### Changed

- **`reporter.Terminal.Report` now returns `error`** (always nil) to
  satisfy the new `Reporter` interface uniformly across formats.
  Existing callers that ignored the return value continue to compile;
  callers that check the return handle the JUnit encoding path too.
- **`evaluator.Evaluate` is now a thin adapter over `Compose`** — the
  per-policy composition primitive owns match selection, allowing
  `Result.Composed` to expose complete multi-policy provenance to
  downstream reporters. The pre-composer global-best selector has been
  removed.
- **`AnalyzeCheck.MinCoverage`** is now `*Percentage` (was `string`).
  The change is source-compatible with any existing spec files because
  both numeric and string-with-percent forms parse into the same type.

### Fixed

- **Cross-policy union bug** — when policy A granted `[read, create]`
  on `secret/*` and policy B granted `[read]` on `secret/foo`,
  requesting `create` on `secret/foo` previously denied because the
  old selector only kept the most specific global match. Per-policy
  composition now unions both policies' contributions correctly.
- **Cross-policy deny-override bug** — an explicit deny on a less
  specific rule in one policy was previously hidden when another
  policy had a more specific allow for the same path. The deny
  override now fires whenever any policy's per-policy winner carries
  the deny capability, matching Vault's runtime behavior.
- **HCL parameter decoder crashes** — `allowed_parameters = ["foo"]`
  and similar non-map values used to panic inside `cty.AsValueMap`.
  The decoder now returns a typed diagnostic with source range
  instead.
- **Silent attribute errors** — a malformed attribute in one path
  block previously caused the parser to drop subsequent attributes on
  that block. All diagnostics are now accumulated and returned.
- **Noisy library logging** — removed `log.Printf` calls from the HCL
  parser's remain-attribute path so the library no longer writes to
  the global logger.

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
