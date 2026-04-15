---
layout: default
title: Architecture
---

[Home](.) |
[Getting Started](getting-started) |
[CLI Reference](cli-reference) |
[**Architecture**](architecture) |
[Roadmap](roadmap) |
[Contributing](contributing)

# Architecture

How custos evaluates Vault policies and the design decisions behind it.

---

## Overview

custos is built around three core subsystems:

```
                    ┌─────────────┐
                    │  Test Spec  │  YAML
                    │   (.yaml)   │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  Spec Loader│  pkg/spec
                    │  & Validator│
                    └──────┬──────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
   ┌──────▼──────┐ ┌──────▼──────┐ ┌───────▼──────┐
   │  HCL Parser │ │  Evaluator  │ │   Analyzer   │
   │  pkg/parser │ │  (offline/  │ │  (security   │
   │             │ │   online)   │ │   scanning)  │
   └──────┬──────┘ └──────┬──────┘ └───────┬──────┘
          │                │                │
          └────────────────┼────────────────┘
                           │
                    ┌──────▼──────┐
                    │  Reporter   │  terminal / junit / json
                    └─────────────┘
```

## Project structure

```
custos/
├── cmd/                    # CLI commands and routing
│   ├── cli.go              # CLI initialization
│   ├── cli_start.go        # Test command implementation
│   └── version_cmd.go      # Version command
├── pkg/
│   ├── parser/             # HCL policy file parsing
│   │   └── hcl.go
│   ├── evaluator/          # Offline policy evaluation engine
│   │   └── offline.go
│   ├── reporter/           # Terminal output with colors
│   │   └── terminal.go
│   └── spec/               # Test specification handling
│       ├── spec.go
│       ├── loader.go
│       └── validate.go
├── version/                # Build-time version info
├── testdata/               # Example policies and specs
│   ├── policies/           # HCL policy fixtures
│   └── specs/              # YAML test spec fixtures
├── .build/                 # Build and install scripts
├── .release/               # Docker and release config
├── main.go                 # Binary entrypoint
├── Makefile                # Build tasks
└── .goreleaser.yml         # Release automation
```

## HCL policy parsing

The parser (`pkg/parser`) uses HashiCorp's own `hcl/v2` library to parse Vault ACL policy files. Each policy file contains `path` blocks:

```hcl
path "secret/data/myapp/*" {
  capabilities = ["read", "list"]
  allowed_parameters = {
    "version" = []
  }
}
```

The parser extracts:

| Field | Description |
|:------|:------------|
| `path` | Vault path pattern (supports `*` and `+` globs) |
| `capabilities` | List of allowed operations |
| `allowed_parameters` | Parameter allow-list |
| `denied_parameters` | Parameter deny-list |
| `required_parameters` | Mandatory parameters |
| `min_wrapping_ttl` | Minimum response wrapping TTL |
| `max_wrapping_ttl` | Maximum response wrapping TTL |

## Offline evaluation engine

The offline evaluator (`pkg/evaluator`) mirrors Vault's actual ACL evaluation logic:

1. **Path resolution** — match the test path against all policy path rules using Vault's glob/prefix matching semantics (`*` matches any characters including `/` separators, `+` matches exactly one path segment)

2. **Most specific match** — Vault uses longest-prefix-match; exact paths beat globs, globs beat prefixes

3. **Capability evaluation** — check whether requested capabilities exist in the matched rule's capability set

4. **Deny override** — `deny` capability on any matching path overrides all other grants

5. **Multi-policy composition** — when multiple policies are loaded, capabilities are unioned across policies, then deny rules apply as overrides

> **Note:** This mirrors Vault's evaluation order as documented in the [Vault ACL policy documentation](https://developer.hashicorp.com/vault/docs/concepts/policies).

## Security analysis

The analyzer (`pkg/analyzer`) performs static analysis on policy HCL
independently of test assertions. Findings carry a `check`, `severity`,
`message`, `file`, `line`, `path`, and the offending rule's
capabilities, so editors and CI annotators can jump straight to the
violating `path` block.

| Check | Detection | Severity |
|:------|:----------|:---------|
| `wildcard_paths` | Paths ending in `*` with 3+ capabilities | Warning |
| `sudo_capability` | `sudo` on any path not under `sys/` or `auth/token/` | Error |
| `root_token_create` | `create` on `auth/token/create` | Error |
| `policy_escalation` | `update` / `create` on `sys/policy/` or `sys/policies/acl/` | Error |
| `secret_destroy` | Destructive ops on `secret/destroy/` or `secret/metadata/` | Warning |
| `coverage` | Percentage of paths with test assertions | Info (planned) |
| `conflicts` | Overlapping allow/deny across policies | Warning (planned) |

Operators configure the analyzer via the `analyze:` section of the spec
YAML. Each entry is keyed by `check:` and supports:

- `disabled: true` — turn the check off entirely.
- `allow_paths: [...]` — per-check path exceptions with Vault-style
  glob matching (`*` trailing prefix, `+` single segment, otherwise
  exact). This is how a break-glass admin policy can keep a legitimate
  `sudo` grant or wildcard without drowning the rest of the report in
  noise.
- `severity: error|warning|info` — override the default severity (for
  example, bumping `secret_destroy` to `error` in a tightly regulated
  environment).

```yaml
analyze:
  - check: sudo_capability
    allow_paths:
      - database/config/rotate
  - check: wildcard_paths
    disabled: true
  - check: secret_destroy
    severity: error
```

## Key dependencies

| Dependency | Purpose |
|:-----------|:--------|
| `hashicorp/hcl/v2` | HCL file parsing |
| `zclconf/go-cty` | Type system for HCL value decoding |
| `fatih/color` | Colored terminal output |
| `timkrebs/gocli` | CLI framework |
| `gopkg.in/yaml.v3` | YAML test spec parsing |
