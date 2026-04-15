---
description: The subset of Vault HCL policy syntax that custos understands.
icon: brackets-curly
---

# HCL policies

custos parses Vault ACL policy files using HashiCorp's official `hcl/v2` library. The supported syntax is a faithful subset of what Vault itself accepts.

## Anatomy of a path block

```hcl
path "secret/data/payment-svc/*" {
  capabilities = ["read", "list"]

  allowed_parameters = {
    "ttl" = []
  }

  denied_parameters = {
    "root_token" = []
  }

  required_parameters = ["cas"]

  min_wrapping_ttl = "1m"
  max_wrapping_ttl = "5m"
}
```

## Supported fields

| Field | Type | Purpose |
|---|---|---|
| `capabilities` | list[string] | Required. Operations this path allows. See [Capabilities](capabilities.md). |
| `allowed_parameters` | map | Whitelist of parameter values. Keys are parameter names, values are allowed value lists. |
| `denied_parameters` | map | Blacklist of parameter values. |
| `required_parameters` | list[string] | Parameters that must be present on the request. |
| `min_wrapping_ttl` | duration string | Minimum response wrapping TTL. |
| `max_wrapping_ttl` | duration string | Maximum response wrapping TTL. |

All fields other than `capabilities` are parsed and preserved but are not currently evaluated. They will be honored when request parameter matching lands in a future release — see the [roadmap](../roadmap.md).

## Path glob semantics

custos implements Vault's exact matching rules:

| Pattern | Meaning | Matches | Does not match |
|---|---|---|---|
| `secret/data/payment-svc` | exact | `secret/data/payment-svc` only | anything else |
| `secret/data/payment-svc/*` | trailing prefix | any path starting with `secret/data/payment-svc/` | `secret/data/payment-svc` (no trailing slash) |
| `secret/data/+/config` | single-segment wildcard | `secret/data/foo/config`, `secret/data/bar/config` | `secret/data/foo/bar/config` |
| `secret/+/+/config` | multiple single-segment | `secret/foo/bar/config` | `secret/foo/config` |

The `*` wildcard only works at the end of a path. An inline `*` is a literal character. `+` is a single-segment wildcard and may appear anywhere.

See [Path matching](../concepts/path-matching.md) for the precedence rules custos applies when multiple patterns match a request.

## Deny capability

`deny` is a capability like any other, with one special behaviour: if a matching rule carries `deny`, the composed result is deny regardless of any other grants.

```hcl
path "secret/data/billing-svc/*" {
  capabilities = ["deny"]
}
```

In a spec, asserting that a path is denied via this rule looks like:

```yaml
- name: "billing denied"
  path: "secret/data/billing-svc/api-key"
  capabilities: [read]
  expect: deny
```

## Source line tracking

custos records the 1-based source line of every parsed `path` block. Analyzer findings include that line, so when you run:

```
wildcard path "secret/*" grants 6 capabilities
  at policies/overprivileged.hcl:6
```

you can jump straight to the offending rule.

## Parser diagnostics

Syntax errors produce source-annotated output:

```
policies/broken.hcl:4,3-15: Missing required argument;
  The argument "capabilities" is required, but no definition was found.
```

The parser uses `hcl/v2` for diagnostics so the error format matches what Vault and other HCL tools produce.

## What is not supported

- **Templated paths** (`{{identity.entity.id}}`). Vault Enterprise templated policies are a planned feature; today, custos will parse them but will not expand templates.
- **Sentinel policies.** custos is ACL-layer only. Sentinel is a planned integration on the v0.5 roadmap.
- **Policy-as-code generators.** custos reads HCL; it does not generate it. Pair with Terraform for generation.
