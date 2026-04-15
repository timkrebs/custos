---
description: Release history and notable changes.
icon: clock-rotate-left
---

# Changelog

The authoritative changelog lives at [`CHANGELOG.md`](https://github.com/timkrebs/custos/blob/main/CHANGELOG.md) in the repository and follows the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format. This page summarizes the most relevant entries for documentation readers.

## Unreleased

Work in the current development line. All items below are available on `main` and will ship with the next tagged release.

### Added

- **Overprivilege analyzer.** Five static security checks (`wildcard_paths`, `sudo_capability`, `root_token_create`, `policy_escalation`, `secret_destroy`) that run against parsed policies and emit structured findings. See [Security analyzers](analyzers/README.md).
- **Per-check configuration.** New `analyze:` section in spec YAML. Each check supports `disabled`, `allow_paths`, and `severity` overrides so teams can keep broad defaults while whitelisting legitimate cases.
- **Source line numbers on parsed path rules.** Findings now point at the exact offending rule in the HCL file.
- **JSON reporter** (`--format=json`). Stable, versioned (`schema_version: "1.0"`) structured output for `jq` pipelines and programmatic consumption. See [JSON](output/json.md).
- **JUnit XML reporter** (`--format=junit`). Standard Ant JUnit output consumed natively by GitHub Actions, GitLab, Jenkins, and Buildkite. See [JUnit XML](output/junit.md).
- **Multi-policy composition engine.** Faithful implementation of Vault's semantics: per-policy most-specific match, union of granted capabilities, hard deny override. See [Policy composition](guides/policy-composition.md).
- **Multi-policy provenance in failures.** Failure messages now include per-policy contributions so the cause of a regression is immediately obvious.
- **Strict YAML decoding.** Unknown fields in spec files are rejected rather than silently ignored. Catches typos immediately.
- **Rich HCL parser diagnostics.** Source-annotated errors for invalid policies.
- **`--compact` flag.** Single-line JSON output for NDJSON log ingestion and line-oriented tools.

### Changed

- **Exit codes on `--fail-on-warn`.** Analyzer warnings now influence the exit code when the flag is set.

## v0.1.0 — 2025

The credibility release. First stable line.

### Added

- `custos test` — offline Vault policy testing via YAML spec files.
- HCL parser for Vault ACL policies (`path` blocks, capabilities, parameter whitelists, wrapping TTLs).
- Offline evaluation engine with Vault-accurate precedence, deny semantics, and glob pattern matching.
- YAML test spec parser.
- Terminal reporter with colored pass/fail output.
- `custos version` command with `--json` flag.
- GoReleaser-based release pipeline with cross-compiled binaries for Linux, macOS, and Windows.
- Homebrew tap and Docker image.

See the full [changelog on GitHub](https://github.com/timkrebs/custos/blob/main/CHANGELOG.md) for detailed notes on every change.
