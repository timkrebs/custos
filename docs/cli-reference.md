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
| `--format` | | string | `terminal` | Output format: `terminal`, `junit`, or `json` |
| `--compact` | | bool | `false` | Emit compact single-line output (JSON only) |
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

# Structured JSON output for programmatic consumption
custos test -f payment-svc.spec.yaml --format=json > results.json

# Compact JSON piped to jq for ad-hoc filtering
custos test -f payment-svc.spec.yaml --format=json --compact | jq '.summary'
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

**`json`** — structured JSON for programmatic consumption. Intended for
`jq` filters, custom dashboards, policy drift detectors, and any tool
that wants to post-process custos results without re-parsing terminal
output. The schema is stable within a major version (see
`schema_version`) so downstream consumers can version-lock safely.

Top-level shape:

```json
{
  "schema_version": "1.0",
  "suite": "payment-service-policies",
  "duration_seconds": 0.000039,
  "summary": {
    "total": 13,
    "passed": 12,
    "failed": 1,
    "warnings": 0
  },
  "warnings": [],
  "results": [
    {
      "name": "can read secrets",
      "path": "secret/data/app/db",
      "capabilities": ["read"],
      "expected": "allow",
      "actual": "allow",
      "pass": true,
      "duration_seconds": 0.000021,
      "explanation": "allowed by rule \"secret/data/app/*\" in policies/app.hcl",
      "matched_rule": {
        "policy_file": "policies/app.hcl",
        "rule_path": "secret/data/app/*",
        "capabilities": ["read", "list"]
      },
      "composed": {
        "denied": false,
        "granted": ["list", "read"],
        "contributions": [
          {
            "policy_file": "policies/app.hcl",
            "rule_path": "secret/data/app/*",
            "capabilities": ["read", "list"],
            "is_deny": false
          }
        ],
        "denied_by": []
      }
    }
  ]
}
```

Schema guarantees:

- `schema_version` follows semver. Consumers should pin on the major
  number (`1`). Additive changes (new optional fields) bump the minor;
  breaking changes bump the major.
- Array-valued fields (`warnings`, `results`, `capabilities`,
  `contributions`, `denied_by`, `granted`) are always emitted as
  arrays, never as `null`. `jq` filters such as
  `.results[] | .capabilities[]` work without null-checking.
- `matched_rule` and `composed` are `null` when the result has no
  matching rule (implicit deny). Consumers should check for null
  before drilling into them.
- Field order within each object is deterministic across runs and Go
  versions because the document is built from Go structs.
- `duration_seconds` is a float (scientific notation allowed for very
  small values; `jq` handles this natively).

**`jq` examples:**

```bash
# Show just the summary counts
custos test -f spec.yaml --format=json | jq '.summary'

# List failing test names with their paths
custos test -f spec.yaml --format=json | jq '.results[] | select(.pass == false) | {name, path}'

# Find every path where a deny overrode an allow from another policy
custos test -f spec.yaml --format=json | jq '.results[] | select(.composed.denied == true) | .path'

# Which policies contributed the "read" capability on any test?
custos test -f spec.yaml --format=json | jq -r '.results[].composed.contributions[] | select(.capabilities | index("read")) | .policy_file' | sort -u

# Exit non-zero if any failed (for ad-hoc pipelines)
test "$(custos test -f spec.yaml --format=json | jq '.summary.failed')" -eq 0
```

Use `--compact` to emit single-line JSON for line-oriented tools and
log ingestion:

```bash
custos test -f spec.yaml --format=json --compact >> custos-results.ndjson
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
