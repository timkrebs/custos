---
title: Getting Started
layout: default
nav_order: 2
---

# Getting Started
{: .no_toc }

Install custos and run your first policy test in under five minutes.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Installation

### Install script (recommended)

```bash
curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash
```

Installs the latest release to `~/.local/bin`. Use `-b /usr/local/bin` for system-wide install.

### Homebrew (macOS/Linux)

```bash
brew install timkrebs/tap/custos
```

### From release binaries

Download the latest release for your platform from the [Releases](https://github.com/timkrebs/custos/releases) page.

### Docker

```bash
docker run --rm -v $(pwd):/work ghcr.io/timkrebs/custos test -f /work/spec.yaml
```

### From source (requires Go 1.22+)

```bash
go install github.com/timkrebs/custos@latest
```

---

## Quick start

### 1. Write a Vault policy

Create a file `policies/payment-svc.hcl`:

```hcl
path "secret/data/payment-svc/*" {
  capabilities = ["read", "list"]
}

path "secret/data/billing-svc/*" {
  capabilities = ["deny"]
}

path "pki_int/issue/payment-svc" {
  capabilities = ["create", "update"]
}

path "transit/encrypt/payment-key" {
  capabilities = ["update"]
}

path "transit/decrypt/payment-key" {
  capabilities = ["update"]
}
```

### 2. Write a test spec

Create a file `payment-svc.spec.yaml`:

```yaml
suite: "payment-service-policies"

policies:
  - path: policies/payment-svc.hcl

tests:
  - name: "can read its own secrets"
    path: "secret/data/payment-svc/db-creds"
    capabilities: [read]
    expect: allow

  - name: "can list its own secret keys"
    path: "secret/data/payment-svc/"
    capabilities: [list]
    expect: allow

  - name: "cannot write to its own secrets"
    path: "secret/data/payment-svc/db-creds"
    capabilities: [create, update]
    expect: deny

  - name: "cannot read billing secrets"
    path: "secret/data/billing-svc/api-key"
    capabilities: [read]
    expect: deny

  - name: "can issue certificates"
    path: "pki_int/issue/payment-svc"
    capabilities: [create, update]
    expect: allow

  - name: "no sys access"
    path: "sys/seal"
    capabilities: [sudo]
    expect: deny

analyze:
  - check: overprivilege
    warn_on:
      - wildcard_paths
      - sudo_capability

  - check: coverage
    min_coverage: 80%
```

### 3. Run tests

```bash
custos test -f payment-svc.spec.yaml
```

### 4. Add to your CI pipeline

```yaml
# .github/workflows/vault-policies.yml
name: Vault policy tests
on:
  pull_request:
    paths: ['policies/**', '*.spec.yaml']

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install custos
        run: go install github.com/timkrebs/custos/cmd/custos@latest

      - name: Run policy tests (offline)
        run: custos test -f payment-svc.spec.yaml --format=junit > results.xml

      - name: Publish test results
        uses: dorny/test-reporter@v1
        if: always()
        with:
          name: Vault policy tests
          path: results.xml
          reporter: java-junit
```

---

## Test spec reference

A test spec is a YAML file with the following structure:

| Field | Type | Required | Description |
|:------|:-----|:---------|:------------|
| `suite` | string | Yes | Name of the test suite |
| `policies` | list | Yes | Policy files to load |
| `policies[].path` | string | Yes | Path to an HCL policy file |
| `tests` | list | Yes | Test assertions |
| `tests[].name` | string | Yes | Human-readable test name |
| `tests[].path` | string | Yes | Vault path to test |
| `tests[].capabilities` | list | Yes | Capabilities to check |
| `tests[].expect` | string | Yes | Expected result: `allow` or `deny` |
| `analyze` | list | No | Security analysis configuration |

### Valid capabilities

`create`, `read`, `update`, `patch`, `delete`, `list`, `sudo`, `deny`, `subscribe`, `recover`

---

## Next steps

- Read the full [CLI Reference]({% link cli-reference.md %}) for all commands and flags
- Understand the [Architecture]({% link architecture.md %}) of the evaluation engine
- Check the [Roadmap]({% link roadmap.md %}) for upcoming features
