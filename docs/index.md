---
layout: default
title: Custos
---

# Custos

**The missing `terraform plan` for HashiCorp Vault policies.**

custos (*lat. guardian*) is a CLI tool that lets you write test specifications for your Vault ACL policies, run them offline or against a live Vault instance, and catch misconfigurations, overprivileged access, and policy conflicts — all before they reach production.

[Getting Started](getting-started) |
[CLI Reference](cli-reference) |
[Architecture](architecture) |
[Roadmap](roadmap) |
[Contributing](contributing) |
[GitHub](https://github.com/timkrebs/custos)

---

```
$ custos test -f payment-svc.spec.yaml

  payment-service-policies

    OK payment service can read its own secrets          (secret/data/payment-svc/db-creds)
    OK payment service cannot read billing secrets       (secret/data/billing-svc/api-key)
    OK payment service cannot delete anything            (secret/data/payment-svc/*)
    OK payment service can issue short-lived certs       (pki_int/issue/payment-svc)
    FAIL no access to sys backend                        (sys/seal)
      → expected: deny, got: allow via policy "admin-legacy"

  4 passed · 1 failed · 0 skipped
```

---

## Why custos?

Every Vault customer hits the same wall: policies are written in HCL, applied to Vault, and then manually tested by creating tokens and running `vault kv get`. There is no structured way to answer *"if I apply this policy, can entity X access path Y?"* without deploying it live.

This creates real problems:

- **No pre-merge validation** — policy changes go through code review but nobody can verify correctness until they hit Vault
- **No regression testing** — a policy refactor might silently grant access to paths that should be denied
- **No security analysis** — wildcard paths, sudo leaks, and policy escalation vectors hide in plain sight
- **No CI integration** — Terraform can plan infrastructure changes, but there is no equivalent for Vault ACL policies

custos fills this gap.

## Key features

| Feature | Description |
|:--------|:------------|
| **Offline evaluation** | Test policies without a running Vault instance |
| **Online verification** | Verify against live Vault using `sys/capabilities` |
| **Security scanning** | Detect overprivileged access and dangerous patterns |
| **Policy composition** | Test the combined effect of multiple policies |
| **CI/CD-ready output** | JUnit XML, JSON, and colored terminal output |

## Quick install

```bash
# Install script (recommended)
curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash

# Homebrew
brew install timkrebs/tap/custos

# Docker
docker run --rm -v $(pwd):/work ghcr.io/timkrebs/custos test -f /work/spec.yaml

# From source
go install github.com/timkrebs/custos@latest
```

See the full [Getting Started](getting-started) guide for a complete walkthrough.

## Comparison

| Feature | custos | vault-policy-testing | Manual testing |
|:--------|:------:|:--------------------:|:--------------:|
| Offline testing (no Vault) | **Yes** | No | No |
| Online verification | **Yes** | Yes | Yes |
| Policy composition | **Yes** | No | Partial |
| Security scanning | **Yes** | No | No |
| Overprivilege detection | **Yes** | No | No |
| CI/CD output (JUnit/JSON) | **Yes** | No | No |
| Air-gapped environments | **Yes** | No | No |

---

*custos is an independent open-source project licensed under [MPL-2.0](https://github.com/timkrebs/custos/blob/main/LICENSE). It is not affiliated with or endorsed by HashiCorp or IBM.*
