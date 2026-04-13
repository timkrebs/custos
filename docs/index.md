---
title: Home
layout: home
nav_order: 1
---

# Custos

**The missing `terraform plan` for HashiCorp Vault policies.**
{: .fs-6 .fw-300 }

custos (*lat. guardian*) is a CLI tool that lets you write test specifications for your Vault ACL policies, run them offline or against a live Vault instance, and catch misconfigurations, overprivileged access, and policy conflicts — all before they reach production.

[Get Started]({% link getting-started.md %}){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/timkrebs/custos){: .btn .fs-5 .mb-4 .mb-md-0 }

---

```
$ custos test -f payment-svc.spec.yaml

  payment-service-policies

    OK payment service can read its own secrets          (secret/data/payment-svc/db-creds)
    OK payment service cannot read billing secrets       (secret/data/billing-svc/api-key)
    OK payment service cannot delete anything            (secret/data/payment-svc/*)
    OK payment service can issue short-lived certs       (pki_int/issue/payment-svc)
    FAIL no access to sys backend                          (sys/seal)
      → expected: deny, got: allow via policy "admin-legacy" (line 14)

  4 passed · 1 failed · 0 skipped

  Security findings:
    WARN  wildcard path "secret/*" grants [create read update delete list] in admin-legacy.hcl:8
    WARN  sudo capability outside sys/ found in admin-legacy.hcl:14
    INFO  policy path coverage: 5/12 paths tested (41%) — consider adding more tests
```

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

custos is an independent open-source project and is not affiliated with or endorsed by HashiCorp or IBM.
