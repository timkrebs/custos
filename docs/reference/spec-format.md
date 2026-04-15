---
description: Complete YAML schema for custos test spec files.
icon: file-code
---

# Spec file format

A custos test spec is a YAML file that declares:

1. Which HCL policy files to load
2. A list of test assertions describing expected allow/deny results
3. Optional static analyzer configuration

## Minimal example

```yaml
suite: "payment-service-policies"

policies:
  - path: ../policies/payment-svc.hcl

tests:
  - name: "can read its own secrets"
    path: "secret/data/payment-svc/db-creds"
    capabilities: [read]
    expect: allow
```

## Top-level schema

| Field | Type | Required | Description |
|---|---|---|---|
| `version` | string | no | Spec format version. If omitted, treated as v1. |
| `suite` | string | yes | Human-readable suite name. Appears in reporter output. |
| `policies` | list | yes | Policy files to load. See [`policies`](#policies). |
| `tests` | list | yes | Test assertions. See [`tests`](#tests). |
| `analyze` | list | no | Per-check analyzer configuration. See [`analyze`](#analyze). |

Unknown top-level fields are rejected. This catches typos in keys rather than silently ignoring them.

## `policies`

A list of HCL files whose rules compose into the entity being tested. Paths are resolved relative to the spec file's directory.

Two forms are accepted:

```yaml
# Scalar form
policies:
  - ../policies/payment-svc.hcl
  - ../policies/readonly.hcl

# Mapping form (preferred, leaves room for future fields)
policies:
  - path: ../policies/payment-svc.hcl
  - path: ../policies/readonly.hcl
```

Mixed forms in the same list are allowed.

Multiple policies compose using Vault's own rules: per-policy most-specific match, union of granted capabilities, deny as a hard override. See [Policy composition](../guides/policy-composition.md).

## `tests`

Each test is an assertion that a given path and capability set should result in `allow` or `deny`.

```yaml
tests:
  - name: "can read its own secrets"
    path: "secret/data/payment-svc/db-creds"
    capabilities: [read]
    expect: allow
```

### Test fields

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Human-readable test name. Appears in reporter output. |
| `path` | string | yes | Vault path to test. Matches literally against policy rules. |
| `capabilities` | list[string] | yes | Capabilities to check. May be empty. |
| `expect` | string | yes | `allow` or `deny`. |

### `capabilities`

Any of: `create`, `read`, `update`, `patch`, `delete`, `list`, `sudo`, `deny`, `subscribe`, `recover`.

See [Capabilities](capabilities.md) for what each one means.

An empty list (`capabilities: []`) tests whether any matching rule exists at all, regardless of what it grants. Use it sparingly.

### `expect`

| Value | Passes when |
|---|---|
| `allow` | Every capability in the list is granted after composition |
| `deny` | At least one capability is denied, or no matching rule exists |

Implicit deny (no matching rule at all) counts as `deny` for assertion purposes.

## `analyze`

Configure the static security analyzer. Each entry targets one check by ID.

```yaml
analyze:
  - check: sudo_capability
    disabled: false
    allow_paths:
      - sys/unseal
      - database/config/rotate
    severity: error

  - check: wildcard_paths
    disabled: true
```

### Analyze fields

| Field | Type | Description |
|---|---|---|
| `check` | string | Check ID (`wildcard_paths`, `sudo_capability`, `root_token_create`, `policy_escalation`, `secret_destroy`) |
| `disabled` | bool | Turn this check off entirely |
| `allow_paths` | list[string] | Whitelist rule paths that the check should ignore. Uses Vault glob syntax. |
| `severity` | string | Override the default severity: `info`, `warning`, or `error` |

See [Security analyzers](../analyzers/README.md) for every built-in check.

## Path resolution rules

1. **Absolute** paths in `policies[].path` are used as-is.
2. **Relative** paths in `policies[].path` are resolved against the directory containing the spec file, not the current working directory.
3. `../` segments are allowed. Sibling `policies/` and `specs/` directories work naturally.

Example:

```
vault-policies/
  policies/
    payment-svc.hcl
  specs/
    payment-svc.spec.yaml   <- declares: ../policies/payment-svc.hcl
```

Running `custos test -f specs/payment-svc.spec.yaml` from any directory resolves to `vault-policies/policies/payment-svc.hcl`.

## Strict decoding

custos uses strict YAML decoding. Unknown fields — at any level — cause the spec to be rejected with a source-annotated error. This catches:

- Typos in field names (`capabilites:` instead of `capabilities:`)
- Stale fields from older schema versions
- Accidentally pasted content from unrelated files

If you need a scratch field, put it in a YAML comment.

## Full example

```yaml
version: v1
suite: "payment-service-policies"

policies:
  - path: ../policies/base-readonly.hcl
  - path: ../policies/payment-svc.hcl

tests:
  - name: "can read own secrets"
    path: "secret/data/payment-svc/db-creds"
    capabilities: [read]
    expect: allow

  - name: "cannot read billing secrets"
    path: "secret/data/billing-svc/api-key"
    capabilities: [read]
    expect: deny

  - name: "no sys backend"
    path: "sys/seal"
    capabilities: [sudo]
    expect: deny

analyze:
  - check: sudo_capability
    allow_paths:
      - sys/unseal

  - check: wildcard_paths
    severity: error

  - check: secret_destroy
    disabled: true
```
