---
description: A worked example of writing a test spec for a real-world service policy.
icon: file-pen
---

# Your first test spec

The [quickstart](quickstart.md) gives you a working setup in five minutes. This page is the next layer up: how to approach a test spec for a real service policy, what to cover, and how to keep the spec maintainable as the policy evolves.

## A realistic scenario

Imagine you are writing a policy for `payment-svc`, a service that:

- Reads its own secrets from `secret/data/payment-svc/*`
- Issues short-lived TLS certificates via PKI
- Encrypts and decrypts with a dedicated transit key
- **Must not** read other services' secrets
- **Must not** touch any system backend

The policy is small but already has five distinct "capability surfaces" to cover, and each needs positive *and* negative assertions.

## The policy

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

## The test spec, organized by intent

A good spec reads like a specification of the service's security contract. Group tests by *intent*, not by path, and for every positive assertion write at least one negative assertion.

```yaml
suite: "payment-service-policies"

policies:
  - path: ../policies/payment-svc.hcl

tests:
  # --- Secrets: own service access ---
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

  # --- Secrets: cross-service isolation ---
  - name: "cannot read billing secrets"
    path: "secret/data/billing-svc/api-key"
    capabilities: [read]
    expect: deny

  - name: "cannot list billing secrets"
    path: "secret/data/billing-svc/"
    capabilities: [list]
    expect: deny

  # --- PKI: issue only for self ---
  - name: "can issue certs for self"
    path: "pki_int/issue/payment-svc"
    capabilities: [create, update]
    expect: allow

  - name: "cannot issue certs for other services"
    path: "pki_int/issue/billing-svc"
    capabilities: [create, update]
    expect: deny

  # --- Transit: own key only ---
  - name: "can encrypt with own key"
    path: "transit/encrypt/payment-key"
    capabilities: [update]
    expect: allow

  - name: "cannot access other transit keys"
    path: "transit/encrypt/billing-key"
    capabilities: [update]
    expect: deny

  # --- Blast radius boundaries ---
  - name: "no sys access"
    path: "sys/seal"
    capabilities: [sudo]
    expect: deny

  - name: "no token minting"
    path: "auth/token/create"
    capabilities: [create]
    expect: deny
```

## Principles illustrated by the spec above

1. **One assertion, one intent.** Each test answers one yes-or-no security question. When a test fails, you know exactly which intent was violated.
2. **Deny tests are load-bearing.** The allow tests show the service can do its job. The deny tests show it *cannot* do anything else. Future refactors can break either side.
3. **Test cross-service isolation explicitly.** `billing-svc` is a stand-in for "any other service." If multi-tenancy is a requirement, assert it.
4. **Test the blast radius.** `sys/seal`, `auth/token/create`, and `sys/policies/acl/*` should be denied for every non-admin service. Copy those assertions into every spec.
5. **Group by intent, comment the groups.** The spec is as much documentation as it is executable tests. Future maintainers read it before they read the policy.

## Running it

```bash
custos test -f specs/payment-svc.spec.yaml
```

If every assertion reflects an intentional design decision, the spec becomes an executable statement of what the service is *allowed* to do. The policy can be refactored freely as long as the spec stays green.

## Next step

Read [Writing test specs](../guides/writing-test-specs.md) for idioms that scale to dozens of services and hundreds of assertions.
