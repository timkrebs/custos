---
description: Go from zero to a passing custos run in five minutes.
icon: bolt
---

# Quickstart

This guide assumes you have custos installed. If not, start at [Installation](installation.md).

## Step 1: Create a sample policy

{% stepper %}
{% step %}
### Make a working directory

```bash
mkdir custos-demo && cd custos-demo
mkdir policies specs
```
{% endstep %}

{% step %}
### Write a Vault HCL policy

Save the following as `policies/payment-svc.hcl`:

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
```

This is a standard Vault ACL policy. Nothing custos-specific yet.
{% endstep %}

{% step %}
### Write a test spec

Save the following as `specs/payment-svc.spec.yaml`:

```yaml
suite: "payment-service-policies"

policies:
  - path: ../policies/payment-svc.hcl

tests:
  - name: "can read its own secrets"
    path: "secret/data/payment-svc/db-creds"
    capabilities: [read]
    expect: allow

  - name: "cannot read billing secrets"
    path: "secret/data/billing-svc/api-key"
    capabilities: [read]
    expect: deny

  - name: "cannot write to its own secrets"
    path: "secret/data/payment-svc/db-creds"
    capabilities: [create, update]
    expect: deny

  - name: "can issue certificates"
    path: "pki_int/issue/payment-svc"
    capabilities: [create, update]
    expect: allow

  - name: "no sys access"
    path: "sys/seal"
    capabilities: [sudo]
    expect: deny
```

Each test says: *at this path, with these capabilities, I expect Vault to `allow` or `deny`.*
{% endstep %}

{% step %}
### Run custos

```bash
custos test -f specs/payment-svc.spec.yaml
```

Expected output:

```
  payment-service-policies

    OK can read its own secrets              (secret/data/payment-svc/db-creds)
    OK cannot read billing secrets           (secret/data/billing-svc/api-key)
    OK cannot write to its own secrets       (secret/data/payment-svc/db-creds)
    OK can issue certificates                (pki_int/issue/payment-svc)
    OK no sys access                         (sys/seal)

  5 passed, 0 failed, 0 skipped
```

custos exits with code `0` on success and `1` on failure. That is all you need for a basic CI gate.
{% endstep %}
{% endstepper %}

## Step 2: Break it on purpose

Open `policies/payment-svc.hcl` and change the billing rule to allow reads:

```hcl
path "secret/data/billing-svc/*" {
  capabilities = ["read"]
}
```

Re-run:

```bash
custos test -f specs/payment-svc.spec.yaml
```

You should now see a failure with the exact reason:

```
    FAIL cannot read billing secrets         (secret/data/billing-svc/api-key)
      expected: deny, got: allow via rule "secret/data/billing-svc/*"
        in policies/payment-svc.hcl
```

Revert the change to get back to green.

## Step 3: Add it to CI

Drop this into `.github/workflows/custos.yml`:

```yaml
name: custos
on: [pull_request]

jobs:
  policy-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install custos
        run: curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash
      - name: Run policy tests
        run: custos test -f specs/payment-svc.spec.yaml --format=junit > results.xml
      - name: Upload JUnit results
        if: always()
        uses: dorny/test-reporter@v1
        with:
          name: custos policy tests
          path: results.xml
          reporter: java-junit
```

Every pull request that touches a policy now has to survive the suite before it can merge.

## Where to go next

- [Your first test spec](first-test.md) walks through spec file design for real services.
- [Writing test specs](../guides/writing-test-specs.md) covers idioms, test organization, and coverage strategy.
- [Security analyzers](../analyzers/README.md) shows how to turn on static analysis alongside your tests.
- [CI integration](../guides/ci-integration.md) has recipes for GitLab, Jenkins, and Buildkite.
