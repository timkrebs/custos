---
description: Flags any policy that can mint new Vault tokens on the token backend.
icon: key-skeleton
---

# root\_token\_create

**Severity:** error (configurable)

## What it flags

Any rule that grants `create` capability on `auth/token/create`, `auth/token/create/*`, or any path starting with `auth/token/create/`.

```hcl
path "auth/token/create" {
  capabilities = ["create", "update"]
}
```

The analyzer emits:

```
root token create capability on "auth/token/create"; this grants
privileged token minting and should be tightly scoped
  at policies/overprivileged.hcl:17
```

## Why it matters

`auth/token/create` is the root of Vault's token hierarchy. An entity with `create` on this path can mint arbitrary child tokens, which — depending on the token role configuration — can hold arbitrary policies. Token minting is effectively the ability to escalate to anything any issued token can do.

This is not hypothetical. Token minting is consistently the top avenue for lateral movement in Vault deployments that experience incidents, because:

- It is often granted to "CI systems" with loose scoping.
- It is rarely audited after initial deployment.
- The token role it uses can be modified later to widen the blast radius without touching the policy itself.

## When it can be legitimate

There are cases where token creation is intentional:

- **CI runners** that mint short-lived tokens for downstream jobs (and should be using AppRole + token roles, not this path).
- **Admin policies** for on-call operators.
- **Token roles** that restrict what the minted tokens can do.

In all of these cases, the preferred pattern is to grant access to a *specific token role* rather than the root `auth/token/create` path:

```hcl
# Preferred: grant access to a named token role
path "auth/token/create/ci-runner" {
  capabilities = ["create", "update"]
}
```

That way, the blast radius is bounded by whatever the `ci-runner` role permits, and the analyzer will not flag the rule because the path does not match the root creation endpoint.

## How to fix it

1. Replace `auth/token/create` with a token role path like `auth/token/create/<role-name>`.
2. Define the token role with `vault write auth/token/roles/<role-name>` and constrain its policies, TTL, and renewable flag.
3. Re-run `custos test` to confirm the finding is gone.

## Configuration

If you truly need an admin policy with raw token creation, whitelist the specific rule path:

```yaml
analyze:
  - check: root_token_create
    allow_paths:
      - auth/token/create    # admin-break-glass policy, audited quarterly
```

## Implementation

The check walks every rule with `create` in its capability list and tests whether the rule path matches one of:

- `auth/token/create` (exact)
- `auth/token/create/*` (trailing prefix)
- `auth/token/create/<anything>`

Any match produces a finding. Variations with `update` instead of `create` are not flagged by this check because they target a different operation — use `policy_escalation` or custom rules for those cases.
