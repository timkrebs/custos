---
title: Architecture
layout: default
nav_order: 4
---

# Architecture
{: .no_toc }

How custos evaluates Vault policies and the design decisions behind it.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

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
│   │   └── hcl.go          # Vault ACL policy parser
│   └── spec/               # Test specification handling
│       ├── spec.go          # Data structures
│       ├── loader.go        # YAML loading
│       └── validate.go      # Validation logic
├── version/                # Build-time version info
│   └── version.go
├── testdata/               # Test fixtures
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

The offline evaluator mirrors Vault's actual ACL evaluation logic:

1. **Path resolution** — match the test path against all policy path rules using Vault's glob/prefix matching semantics (`*` matches any single path segment, `+` matches one segment in newer Vault versions)

2. **Most specific match** — Vault uses longest-prefix-match; exact paths beat globs, globs beat prefixes

3. **Capability evaluation** — check whether requested capabilities exist in the matched rule's capability set

4. **Deny override** — `deny` capability on any matching path overrides all other grants

5. **Multi-policy composition** — when multiple policies are loaded, capabilities are unioned across policies, then deny rules apply as overrides

{: .note }
This mirrors Vault's evaluation order as documented in the [Vault ACL policy documentation](https://developer.hashicorp.com/vault/docs/concepts/policies).

## Online verification

When `--vault-addr` is provided, custos uses the Vault API:

| Endpoint | Purpose |
|:---------|:--------|
| `POST sys/capabilities` | Evaluate capabilities for a specific token |
| `POST sys/capabilities-self` | Evaluate capabilities for the calling token |
| `GET sys/policy/{name}` | Retrieve policy definitions for scanning |

Online mode captures effects that offline evaluation cannot model:
- Sentinel policies (Enterprise)
- Identity group membership and entity aliases
- Namespace chroot listeners
- MFA enforcement

## Security analysis

The analyzer performs static analysis on policy HCL:

| Check | Detection | Severity |
|:------|:----------|:---------|
| `wildcard_paths` | Paths ending in `*` with 3+ capabilities | Warning |
| `sudo_capability` | `sudo` on any path not under `sys/` | Error |
| `root_token_create` | `create` on `auth/token/create` | Error |
| `policy_escalation` | `update` on `sys/policy/*` | Error |
| `secret_destroy` | `delete` on `secret/destroy/*` | Warning |
| `coverage` | Percentage of paths with test assertions | Info |
| `conflicts` | Overlapping allow/deny across policies | Warning |

## Test spec format

Test specifications are YAML files that define:

- **Suite name** — identifies the test suite
- **Policy references** — paths to HCL policy files to load
- **Test cases** — assertions about path/capability combinations
- **Analysis config** — optional security analysis settings

The spec loader (`pkg/spec`) validates:
- Required fields are present (suite name, at least one test)
- Capabilities use valid Vault capability names
- Expectations are either `allow` or `deny`
- Referenced policy files exist

## Key dependencies

| Dependency | Purpose |
|:-----------|:--------|
| `hashicorp/hcl/v2` | HCL file parsing |
| `zclconf/go-cty` | Type system for HCL value decoding |
| `timkrebs/gocli` | CLI framework |
| `gopkg.in/yaml.v3` | YAML test spec parsing |
