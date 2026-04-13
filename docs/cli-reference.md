---
title: CLI Reference
layout: default
nav_order: 3
---

# CLI Reference
{: .no_toc }

Complete reference for all custos commands and flags.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

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

#### Flags

| Flag | Short | Type | Default | Description |
|:-----|:------|:-----|:--------|:------------|
| `--file` | `-f` | string | *(required)* | Path to test spec YAML file |
| `--vault-addr` | | string | | Vault server address (enables online mode) |
| `--vault-token` | | string | | Vault authentication token |
| `--vault-namespace` | | string | | Vault namespace (Enterprise only) |
| `--format` | | string | `terminal` | Output format: `terminal`, `junit`, `json` |
| `--fail-on-warn` | | bool | `false` | Exit non-zero on security warnings |
| `--timeout` | | duration | `10s` | Timeout for online Vault requests |
| `--verbose` | `-v` | bool | `false` | Show detailed evaluation trace |

#### Examples

```bash
# Offline testing (no Vault required)
custos test -f payment-svc.spec.yaml

# Online verification against live Vault
custos test -f payment-svc.spec.yaml \
  --vault-addr=https://vault.example.com \
  --vault-token=$VAULT_TOKEN

# CI-friendly JUnit output
custos test -f payment-svc.spec.yaml --format=junit > results.xml

# JSON output for programmatic consumption
custos test -f payment-svc.spec.yaml --format=json | jq '.failures'

# Verbose trace for debugging
custos test -f payment-svc.spec.yaml -v
```

---

### `custos scan`

Security scan policies for dangerous patterns without requiring a test spec.

```bash
custos scan <policy-files...> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|:-----|:-----|:--------|:------------|
| `--vault-addr` | string | | Scan live Vault policies instead of files |
| `--vault-token` | string | | Vault authentication token |
| `--severity` | string | `warning` | Minimum severity: `info`, `warning`, `error` |
| `--format` | string | `terminal` | Output format: `terminal`, `json` |

#### Examples

```bash
# Scan local policy files
custos scan policies/*.hcl

# Scan with all severity levels
custos scan policies/*.hcl --severity=info
```

---

### `custos init`

Generate a test spec skeleton from existing policy files.

```bash
custos init --from <policy-file> [flags]
```

#### Flags

| Flag | Type | Default | Description |
|:-----|:-----|:--------|:------------|
| `--from` | string | *(required)* | Path to HCL policy file(s) |
| `--all-paths` | bool | `false` | Generate assertions for every path in the policy |

#### Examples

```bash
# Generate a skeleton spec
custos init --from policies/payment-svc.hcl > payment-svc.spec.yaml

# Generate assertions for all paths
custos init --from policies/payment-svc.hcl --all-paths
```

---

### `custos validate`

Syntax-check a test spec file without running it.

```bash
custos validate -f <spec-file>
```

#### Examples

```bash
custos validate -f payment-svc.spec.yaml
```

---

### `custos version`

Print version information.

```bash
custos version [--json]
```

#### Flags

| Flag | Type | Default | Description |
|:-----|:-----|:--------|:------------|
| `--json` | bool | `false` | Output version as JSON |

#### Examples

```bash
$ custos version
custos v0.2.0 (abc1234, clean, 2026-04-13T12:00:00Z)

$ custos version --json
{"version":"0.2.0","git_commit":"abc1234...","git_tree_state":"clean","build_date":"2026-04-13T12:00:00Z","go_version":"go1.24.4","platform":"darwin/arm64"}
```

---

## Exit codes

| Code | Meaning |
|:-----|:--------|
| `0` | All tests passed, no errors |
| `1` | One or more tests failed |
| `2` | Invalid input (bad spec, missing file, etc.) |
| `3` | Security warnings detected (only with `--fail-on-warn`) |

## Environment variables

| Variable | Description |
|:---------|:------------|
| `VAULT_ADDR` | Vault server address (alternative to `--vault-addr`) |
| `VAULT_TOKEN` | Vault authentication token (alternative to `--vault-token`) |
| `VAULT_NAMESPACE` | Vault namespace (alternative to `--vault-namespace`) |
| `NO_COLOR` | Disable colored terminal output when set |
