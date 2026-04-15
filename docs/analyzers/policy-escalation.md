---
description: Detects policies that can rewrite ACL policies and trivially escalate privilege.
icon: arrow-trend-up
---

# policy\_escalation

**Severity:** error (configurable)

## What it flags

Any rule granting `create` or `update` on paths under `sys/policy/*` or `sys/policies/acl/*`.

```hcl
path "sys/policies/acl/*" {
  capabilities = ["create", "read", "update", "delete"]
}
```

The analyzer emits:

```
policy escalation: update/create on "sys/policies/acl/*" lets this
policy rewrite any ACL policy in Vault, including its own
  at policies/overprivileged.hcl:21
```

## Why it matters

This is the single most dangerous anti-pattern the analyzer catches. A policy that can modify `sys/policies/acl/*` can rewrite *itself* to grant any capability on any path. From that moment on, every other security control in the cluster is moot — the attacker holding this policy can compose a new policy that reads every secret, mints any token, and seals the vault.

Unlike `root_token_create`, there is no "scoped" version of this grant that is safe. You either can write policies or you cannot. There is no middle ground.

## The escalation path

1. Attacker compromises an entity holding a policy with `update` on `sys/policies/acl/my-policy`.
2. Attacker writes a new version of `my-policy` granting `*` on `secret/*`.
3. Attacker renews the token (now inheriting the new policy contents) and reads every secret.

The whole sequence takes two Vault API calls. The only defense is making sure no non-admin policy ever grants these capabilities.

## How to fix it

Remove the rule. If policy management is genuinely needed for this entity — which is rare and should be limited to trusted operators — move it to a dedicated `vault-admin` policy attached to a small, audited group.

```hcl
# Before: inside a general-purpose policy
path "sys/policies/acl/*" {
  capabilities = ["create", "read", "update", "delete"]
}

# After: move to a dedicated admin policy, separate from service policies
```

## Legitimate exceptions

There are genuine cases where a policy needs to write ACL policies — for example, a Terraform service account that manages Vault policies declaratively. For those:

1. Scope the Terraform service account policy to *only* the paths it needs to manage, with no other capabilities.
2. Rotate the Terraform credential frequently.
3. Audit the Terraform plan/apply log as if it were a production deployment.

If you want to whitelist this specific rule:

```yaml
analyze:
  - check: policy_escalation
    allow_paths:
      - sys/policies/acl/*   # terraform-managed, see docs/vault-as-code.md
```

But please read the rationale above first. A blanket whitelist is almost always a mistake.

## Read is fine

Reading policy definitions (`read` on `sys/policies/acl/*`) is not flagged. It does not allow modification and is commonly needed for documentation tools, audits, and the GitHub Actions policy linter. Only `create` and `update` trigger the finding.

## Configuration

```yaml
analyze:
  - check: policy_escalation
    disabled: false
    severity: error
```

Disabling this check is strongly discouraged. If you think you need to, talk to somebody on your security team first.

## Implementation

The check inspects every rule for `create` or `update` in its capabilities list and tests whether the rule path starts with `sys/policies/acl/` or `sys/policy/`. Both the legacy (`sys/policy`) and current (`sys/policies/acl`) paths are covered.
