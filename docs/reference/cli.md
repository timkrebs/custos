---
description: Every custos command, flag, and example.
icon: terminal
---

# CLI

## Synopsis

```
custos <command> [flags]
```

## Commands

| Command | Description |
|---|---|
| [`custos test`](#custos-test) | Evaluate a test spec against HCL policy files |
| [`custos version`](#custos-version) | Print version and build information |

## `custos test`

Evaluate a YAML test spec against one or more HCL policy files.

```
custos test -f <spec-file> [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `-f, --file` | *required* | Path to the test spec YAML file |
| `--format` | `terminal` | Output format: `terminal`, `junit`, or `json` |
| `--compact` | `false` | Single-line JSON output (`--format=json` only) |
| `--fail-on-warn` | `false` | Exit non-zero if any analyzer warnings are emitted |
| `-v, --verbose` | `false` | Print per-test evaluation trace with matched rule and contributions |

### Examples

Basic run:

```bash
custos test -f specs/payment-svc.spec.yaml
```

Verbose output with full evaluation trace:

```bash
custos test -f specs/payment-svc.spec.yaml -v
```

Fail CI on warnings, not just assertion failures:

```bash
custos test -f specs/payment-svc.spec.yaml --fail-on-warn
```

JUnit XML for CI reporters:

```bash
custos test -f specs/payment-svc.spec.yaml --format=junit > results.xml
```

JSON for programmatic consumption:

```bash
custos test -f specs/payment-svc.spec.yaml --format=json | jq '.summary'
```

Compact NDJSON for log ingestion:

```bash
custos test -f specs/payment-svc.spec.yaml --format=json --compact >> results.ndjson
```

### Exit behaviour

| Condition | Exit code |
|---|---|
| All tests pass, no warnings | `0` |
| All tests pass, warnings emitted, no `--fail-on-warn` | `0` |
| All tests pass, warnings emitted, `--fail-on-warn` set | `1` |
| One or more tests fail | `1` |
| Spec file not found or invalid | `1` |
| HCL policy file invalid | `1` |

See [Exit codes](exit-codes.md) for a detailed table.

## `custos version`

Print version, commit hash, and build metadata.

```
custos version [--json]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--json` | `false` | Emit version metadata as JSON |

### Examples

Human-readable output:

```
$ custos version
custos v0.1.0 (abc1234, clean, 2026-04-13T12:00:00Z)
  Go version:     go1.25
  Platform:       darwin/arm64
```

JSON output:

```json
{
  "version": "0.1.0",
  "commit": "abc1234",
  "dirty": false,
  "build_time": "2026-04-13T12:00:00Z",
  "go_version": "go1.25",
  "platform": "darwin/arm64"
}
```

## Environment variables

| Variable | Effect |
|---|---|
| `NO_COLOR` | Disables ANSI color codes in the terminal reporter |

## Planned commands

The following commands are on the [roadmap](../roadmap.md) and not yet available:

- **`custos validate`** — syntax-check a spec file without evaluating it
- **`custos init --from policy.hcl`** — generate a spec skeleton from an existing policy
- **`custos scan`** — standalone security scan (the analyzer already runs inside `test`)

## Global conventions

- All file paths are resolved relative to the current working directory unless they are absolute.
- Policy paths inside a spec file are resolved relative to the spec file's directory, not the working directory.
- Standard error carries diagnostics; standard output carries reporter output. Pipe safely:

```bash
custos test -f spec.yaml --format=json 2>errors.log | jq .
```
