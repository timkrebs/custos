---
description: Stable-schema structured output for jq, dashboards, and drift detection.
icon: brackets-curly
---

# JSON

The JSON reporter emits a structured, versioned document describing the full suite run. It is the right choice when you want to pipe custos output into `jq`, build dashboards, or run drift detection against previous results.

## Enable it

```bash
# Pretty-printed (default)
custos test -f spec.yaml --format=json

# Single-line (for NDJSON, log ingestion)
custos test -f spec.yaml --format=json --compact
```

## Schema

```json
{
  "schema_version": "1.0",
  "suite": "payment-service-policies",
  "duration_seconds": 0.000039,
  "summary": {
    "total": 9,
    "passed": 8,
    "failed": 1,
    "warnings": 0
  },
  "warnings": [],
  "results": [
    {
      "name": "can read its own secrets",
      "path": "secret/data/payment-svc/db-creds",
      "capabilities": ["read"],
      "expected": "allow",
      "actual": "allow",
      "pass": true,
      "duration_seconds": 0.000021,
      "explanation": "allowed by rule \"secret/data/payment-svc/*\" in policies/payment-svc.hcl",
      "matched_rule": {
        "policy_file": "policies/payment-svc.hcl",
        "rule_path": "secret/data/payment-svc/*",
        "capabilities": ["read", "list"]
      },
      "composed": {
        "denied": false,
        "granted": ["list", "read"],
        "contributions": [
          {
            "policy_file": "policies/payment-svc.hcl",
            "rule_path": "secret/data/payment-svc/*",
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

## Schema stability

`schema_version` follows semantic versioning:

- **Major bump** — backwards-incompatible change (field removed, field type changed).
- **Minor bump** — new field added, existing fields unchanged.
- **Patch bump** — cosmetic change, no semantic difference.

Pin on the major version in any code that parses custos output:

```bash
custos test -f spec.yaml --format=json \
  | jq -e '.schema_version | startswith("1.")'
```

## Top-level fields

| Field | Type | Description |
|---|---|---|
| `schema_version` | string | Schema version this output conforms to. |
| `suite` | string | Value of the spec's `suite:` field. |
| `duration_seconds` | number | Total wall-clock duration. |
| `summary` | object | Aggregate counts. |
| `warnings` | array | Analyzer findings from the current run. Always an array, never null. |
| `results` | array | One entry per test assertion. Always an array, never null. |

## Result object

| Field | Type | Description |
|---|---|---|
| `name` | string | Test name from the spec. |
| `path` | string | Vault path under test. |
| `capabilities` | array[string] | Capabilities checked. |
| `expected` | string | `allow` or `deny`. |
| `actual` | string | What the engine decided. |
| `pass` | bool | Whether `expected == actual`. |
| `duration_seconds` | number | Time spent evaluating this test. |
| `explanation` | string | Human-readable sentence describing the decision. |
| `matched_rule` | object or null | The decisive rule. `null` for implicit deny. |
| `composed` | object or null | Composition trace. `null` when only a single policy contributes and no deny is involved. |

## Composed object

| Field | Type | Description |
|---|---|---|
| `denied` | bool | True if any contributing rule carried deny. |
| `granted` | array[string] | Union of granted capabilities, alphabetized. |
| `contributions` | array | One per contributing policy (matched or not). |
| `denied_by` | array | Policy files whose rule carried deny. |

## `jq` recipes

**Summary only**

```bash
custos test -f spec.yaml --format=json | jq '.summary'
```

**Names of failing tests**

```bash
custos test -f spec.yaml --format=json \
  | jq -r '.results[] | select(.pass == false) | .name'
```

**Paths that were denied**

```bash
custos test -f spec.yaml --format=json \
  | jq -r '.results[] | select(.actual == "deny") | .path' \
  | sort -u
```

**Policies that granted the `read` capability on any path**

```bash
custos test -f spec.yaml --format=json \
  | jq -r '.results[].composed.contributions[]
           | select(.capabilities | index("read"))
           | .policy_file' \
  | sort -u
```

**Fail a shell script if any test failed**

```bash
failed=$(custos test -f spec.yaml --format=json | jq '.summary.failed')
[ "$failed" -eq 0 ]
```

## Compact mode

`--compact` emits the entire document on a single line, suitable for NDJSON-style logs and line-oriented tools:

```bash
for spec in specs/*.spec.yaml; do
  custos test -f "$spec" --format=json --compact
done >> results.ndjson
```

`jq` handles the NDJSON naturally:

```bash
jq -s '[.[] | .summary.failed] | add' results.ndjson
```

## Guarantees

- **Arrays are always arrays.** `warnings`, `results`, `capabilities`, `contributions`, `denied_by` are always emitted as arrays, never `null`. Safe to index without null checks.
- **Field order is deterministic.** Running the same spec twice produces byte-identical JSON (modulo timing fields).
- **Timings are floats.** Scientific notation is used for very small values to avoid rounding.
- **Strings are UTF-8.** No locale-dependent encoding.
