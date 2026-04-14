---
layout: default
title: CLI Reference
---

[Home](.) |
[Getting Started](getting-started) |
[**CLI Reference**](cli-reference) |
[Architecture](architecture) |
[Roadmap](roadmap) |
[Contributing](contributing)

# CLI Reference

Complete reference for all custos commands and flags.

---

## Usage

```
custos <command> [flags]
```

## Commands

### `custos test`

Run test assertions against Vault policies. This is the core command — it evaluates your test spec against one or more HCL policy files and reports pass/fail results.

```bash
custos test -f <spec-file> [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|:-----|:------|:-----|:--------|:------------|
| `--file` | `-f` | string | *(required)* | Path to test spec YAML file |
| `--format` | | string | `terminal` | Output format: `terminal` or `junit` |
| `--fail-on-warn` | | bool | `false` | Exit non-zero on security warnings |
| `--verbose` | `-v` | bool | `false` | Show detailed evaluation trace |

**Examples:**

```bash
# Basic offline testing
custos test -f payment-svc.spec.yaml

# Verbose output showing evaluation trace (including multi-policy provenance)
custos test -f payment-svc.spec.yaml -v

# Fail on warnings (useful in CI)
custos test -f payment-svc.spec.yaml --fail-on-warn

# JUnit XML output for CI reporters
custos test -f payment-svc.spec.yaml --format=junit > results.xml
```

#### Output formats

`custos test` supports two output formats, selected with `--format`:

**`terminal`** (default) — colored, human-readable output for interactive
shells and local development. Shows pass/fail markers, the failure detail
line with `expected: ... got: ... via policy ...`, and a compact
`contributions:` block for failures that involved multiple policies.
Respects `NO_COLOR`.

**`junit`** — JUnit XML intended for CI test reporters. The output is a
standard `<testsuites>` → `<testsuite>` → `<testcase>` tree with:

- Per-test timing in the `time` attribute (float seconds, microsecond precision).
- Suite-level timestamp in ISO 8601 UTC.
- `<failure>` elements on failing cases, with:
  - a short `message` attribute (`expected allow, got deny at path X`),
  - a `type="AssertionError"` attribute,
  - a longer chardata body containing expected/got, path, capabilities, the
    matched rule, explanation, and multi-policy contribution provenance when
    more than one policy contributed.
- XML-escaped special characters so test names and paths containing
  `<`, `>`, `&`, or quotes remain valid.

The document is written entirely to stdout with no mixed-in terminal
output, so it can be redirected directly to a file:

```bash
custos test -f spec.yaml --format=junit > results.xml
```

**GitHub Actions example** (using [`dorny/test-reporter`](https://github.com/dorny/test-reporter)):

```yaml
- name: Run custos policy tests
  run: custos test -f testdata/specs/payment-svc.spec.yaml --format=junit > custos-results.xml

- name: Upload JUnit results
  if: always()
  uses: dorny/test-reporter@v1
  with:
    name: custos policy tests
    path: custos-results.xml
    reporter: java-junit
```

The same file works with Jenkins' JUnit plugin and GitLab CI's
`artifacts:reports:junit` field without modification.

---

### `custos version`

Print version information.

```bash
custos version [--json]
```

**Flags:**

| Flag | Type | Default | Description |
|:-----|:-----|:--------|:------------|
| `--json` | bool | `false` | Output version as JSON |

**Examples:**

```bash
$ custos version
custos v0.1.0 (abc1234, clean, 2026-04-13T12:00:00Z)

$ custos version --json
{"version":"0.1.0","git_commit":"abc1234...","git_tree_state":"clean",...}
```

---

## Exit codes

| Code | Meaning |
|:-----|:--------|
| `0` | All tests passed, no errors |
| `1` | One or more tests failed, or security warnings with `--fail-on-warn` |

## Environment variables

| Variable | Description |
|:---------|:------------|
| `NO_COLOR` | Disable colored terminal output when set |

---

## Planned commands (v0.2.0+)

These commands are on the [Roadmap](roadmap) but not yet implemented:

| Command | Purpose |
|:--------|:--------|
| `custos scan` | Security scan policies for dangerous patterns |
| `custos init` | Generate a test spec skeleton from existing policies |
| `custos validate` | Syntax-check a test spec file |
