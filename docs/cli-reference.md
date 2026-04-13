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
| `--fail-on-warn` | | bool | `false` | Exit non-zero on security warnings |
| `--verbose` | `-v` | bool | `false` | Show detailed evaluation trace |

**Examples:**

```bash
# Basic offline testing
custos test -f payment-svc.spec.yaml

# Verbose output showing evaluation trace
custos test -f payment-svc.spec.yaml -v

# Fail on warnings (useful in CI)
custos test -f payment-svc.spec.yaml --fail-on-warn
```

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
