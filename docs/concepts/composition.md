---
description: The formal rules custos uses to combine multiple policies attached to a single entity.
icon: code-merge
---

# Policy composition semantics

When a Vault entity holds more than one policy, every request is evaluated against all of them together. custos implements the same semantics offline. This page is the formal specification of the composition rules.

For a practical walk-through with examples, see [Policy composition](../guides/policy-composition.md) in the guides section.

## The contract

Given a request path `P`, a capability set `C`, and a list of policies `[P1, P2, ..., Pn]`:

1. For each policy `Pi`, find the most-specific rule that matches `P`. Call it `Ri` (or null if nothing matches).
2. Let `G = union of Ri.capabilities for every Ri that is not null and does not carry deny`.
3. Let `D = { Pi for which Ri carries the deny capability }`.
4. The composed result is `deny` if `D` is non-empty.
5. Otherwise the composed result is `allow` if and only if every capability in `C` is present in `G`.

## Properties

### Commutativity

The order of policies in the spec does not affect the result. Composition is defined as a union over a set.

### Idempotence

Listing the same policy twice has no effect. The union absorbs duplicates.

### Deny monotonicity

Adding a policy that contributes deny can never make a request more permissive. Adding a policy without deny can never make a request less permissive than the existing composed result, *unless* no earlier policy matched (in which case the new policy goes from implicit deny to an explicit grant).

### Per-policy, not per-rule, matching

Matching happens at the policy level, not across the whole corpus. If policy A has rules `secret/*` and `secret/data/foo/*`, and policy B has rule `secret/data/foo/bar`, then for the request `secret/data/foo/bar`:

- Policy A contributes its own most-specific rule, `secret/data/foo/*`.
- Policy B contributes `secret/data/foo/bar`.
- Both contributions union; deny overrides nothing applies.

custos does *not* pick the globally most-specific rule across policies. It picks per policy, which matches Vault.

## Worked table

Two policies:

**`readonly.hcl`**

```hcl
path "secret/*" {
  capabilities = ["read", "list"]
}
path "sys/health" {
  capabilities = ["read"]
}
```

**`payment-svc.hcl`**

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
```

| Request path | capability | readonly contribution | payment-svc contribution | Union (G) | Deny (D) | Result |
|---|---|---|---|---|---|---|
| `secret/data/payment-svc/db-creds` | read | `{read, list}` | `{read, list}` | `{read, list}` | `{}` | allow |
| `secret/data/billing-svc/api-key` | read | `{read, list}` | deny | `{read, list}` | `{payment-svc}` | **deny** |
| `secret/data/other/foo` | read | `{read, list}` | no match | `{read, list}` | `{}` | allow |
| `secret/data/other/foo` | create | `{read, list}` | no match | `{read, list}` | `{}` | **deny** (create not in G) |
| `pki_int/issue/payment-svc` | create | no match | `{create, update}` | `{create, update}` | `{}` | allow |
| `sys/health` | read | `{read}` | no match | `{read}` | `{}` | allow |
| `sys/seal` | sudo | no match | no match | `{}` | `{}` | **deny** (implicit) |

Implicit deny is what happens when `G` and `D` are both empty and the requested capabilities cannot be satisfied from an empty grant set.

## Contributions in failures

Every composed test result carries a `Contributions` slice recording what each policy offered. When a test fails, the reporter prints those contributions so the cause is obvious:

```
FAIL billing denied despite readonly allowing read
  expected: deny, got: allow
  contributions:
    readonly      secret/*                 GRANT [read, list]
    payment-svc   secret/data/billing-svc/* no match
```

In this hypothetical failure, the `deny` rule in `payment-svc.hcl` has been removed by accident. The contribution trace points straight at the regression.

## Edge cases

### Two policies with conflicting grants

Union semantics mean the more permissive grant wins in the absence of deny. If policy A grants `read` and policy B grants `create`, the composed grant is `{read, create}`. If the effect is unintended, the fix is to add an explicit `deny` on the path in the policy that should win, or to scope the policies more narrowly.

### Deny and sudo on the same path

A rule carrying `deny` overrides any other grant, including `sudo`. There is no way to sudo around a deny.

### Empty capabilities list

A test case with `capabilities: []` passes when *any* matching rule exists after composition, regardless of what it grants. It fails only on implicit deny. Use this for path coverage probes, not for security assertions.

## Why this is load-bearing

Composition is where most real-world Vault regressions hide. A service gets a new policy added, a shared policy is widened "just slightly," and suddenly a path that used to be denied is open. Offline composition testing catches these before they land in production.
