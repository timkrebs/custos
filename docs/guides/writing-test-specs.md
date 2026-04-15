---
description: Idioms and patterns for test specs that scale past a single service.
icon: pen-ruler
---

# Writing test specs

Once you have a handful of specs, the questions become less about syntax and more about organization. This guide covers patterns that keep a growing spec suite maintainable.

## One spec per policy boundary

Keep the scope of each spec file close to the scope of the policy it tests. Some useful boundaries:

- **Per service.** `specs/payment-svc.spec.yaml` tests `policies/payment-svc.hcl`.
- **Per role.** `specs/readonly-role.spec.yaml` tests the shared read-only policy that many services receive.
- **Per composition.** `specs/payment-svc-composed.spec.yaml` tests the combined behaviour of every policy attached to the payment service entity.

Small, focused spec files give you clearer CI signal when something fails and let you parallelize tests across services.

## Positive and negative assertions

For every allow test, write at least one deny test. Positive tests prove the policy does *something*; negative tests prove the policy does *only* that. Most regressions show up in the negative tests.

```yaml
- name: "payment-svc can read its own secrets"
  path: "secret/data/payment-svc/db-creds"
  capabilities: [read]
  expect: allow

- name: "payment-svc cannot read other services' secrets"
  path: "secret/data/billing-svc/api-key"
  capabilities: [read]
  expect: deny
```

## Always assert the blast radius

Every service should explicitly deny access to the endpoints that compromise the whole cluster. Copy this block into every spec for any service that is not a Vault administrator:

```yaml
- name: "no seal or unseal"
  path: "sys/seal"
  capabilities: [sudo, update]
  expect: deny

- name: "no token minting"
  path: "auth/token/create"
  capabilities: [create]
  expect: deny

- name: "no policy self-escalation"
  path: "sys/policies/acl/payment-svc"
  capabilities: [update, create]
  expect: deny
```

These take no effort to write and catch the most dangerous classes of misconfiguration.

## Name tests like specifications

Test names appear in failure messages, JUnit reports, and CI dashboards. Write them as full sentences describing the guarantee the test proves.

| Good | Less good |
|---|---|
| `can read its own secrets` | `test 1` |
| `cannot issue certs for other services` | `pki denied` |
| `no policy self-escalation on sys/policies/acl/*` | `escalation` |

## Test every capability you care about

Vault enforces capabilities independently. A rule that grants `[read, list]` denies `create`, `update`, `delete`, and `sudo` implicitly. If any of those negative cases matter, write an explicit test so a future capability widening is caught.

```yaml
- name: "read-only: no writes"
  path: "secret/data/shared/config"
  capabilities: [create, update, delete]
  expect: deny
```

## Use the empty capability list sparingly

custos accepts `capabilities: []`. This tests whether the path has *any* matching rule at all, regardless of what it grants. It is useful for path coverage probes but does not replace explicit capability assertions.

## Organize by intent with comments

YAML comments are free. Use them to turn a flat list of tests into a readable document:

```yaml
tests:
  # --- Secrets: own access ---
  - name: "can read own secrets"
    ...

  # --- Secrets: cross-service isolation ---
  - name: "cannot read other services"
    ...

  # --- Blast radius boundaries ---
  - name: "no sys access"
    ...
```

## Pin the analyze section

Once you have decided which security checks apply to this project, commit the `analyze` section to the spec. Teams that do this on day one have an easier time adopting new checks: opting in to a new analyzer is a small, reviewable diff.

```yaml
analyze:
  - check: sudo_capability
    allow_paths:
      - database/config/rotate

  - check: wildcard_paths
    severity: error

  - check: secret_destroy
    disabled: true
```

See [Security analyzers](../analyzers/README.md) for the list of checks and how configuration works.

## Keep tests close to policies

The simplest layout works:

```
vault-policies/
  policies/
    payment-svc.hcl
    billing-svc.hcl
    readonly.hcl
  specs/
    payment-svc.spec.yaml
    billing-svc.spec.yaml
    readonly.spec.yaml
```

Run the whole suite in CI with a tiny shell loop:

```bash
for spec in specs/*.spec.yaml; do
  custos test -f "$spec" --format=junit > "results/$(basename "$spec" .spec.yaml).xml" || fail=1
done
exit ${fail:-0}
```

## Review failing tests like code

When a test fails, the first question is *"was the policy wrong, or was the test wrong?"* The failure message tells you which rule matched and which policy file it came from, so the answer is usually one `git blame` away. Treat assertion changes with the same care as code changes: they encode security intent.
