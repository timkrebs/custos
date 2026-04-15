---
description: Detects destructive KV v2 operations that permanently remove secret versions or metadata.
icon: trash-can
---

# secret\_destroy

**Severity:** warning (configurable)

## What it flags

Rules that grant either:

- `delete` on `secret/metadata/*`, or
- `update` or `delete` on `secret/destroy/*`

```hcl
path "secret/destroy/*" {
  capabilities = ["update"]
}

path "secret/metadata/*" {
  capabilities = ["delete"]
}
```

The analyzer emits:

```
destructive KV v2 operation on "secret/destroy/*"; this permanently
destroys secret versions and cannot be reversed
  at policies/overprivileged.hcl:35
```

## Why it matters

Vault's KV v2 engine supports three distinct destructive operations, each progressively more permanent:

1. **Soft delete** (`delete` on `secret/data/*`). Marks the current version as deleted. Recoverable with `undelete`.
2. **Version destroy** (`update` on `secret/destroy/*`). Permanently destroys a specific version of a secret. Irreversible.
3. **Metadata delete** (`delete` on `secret/metadata/*`). Permanently destroys *every* version of a secret and its metadata. Irreversible and usually unrecoverable.

The first is a normal operation. The second and third are usually only needed for compliance workflows like GDPR data erasure, and they are routinely granted by mistake because teams see "KV" and copy their `secret/data/*` grants over.

The finding is a warning rather than an error because there are legitimate uses. But it is a warning you should treat seriously: a destroy capability granted to a general-purpose service is a latent data-loss incident.

## Why the distinction matters

Soft deletes can be reversed with a `vault kv undelete` or by restoring from a snapshot. Version destroy and metadata delete cannot: the data is removed from Vault's storage and, unless you have point-in-time backups of the underlying storage engine, it is gone.

## How to fix it

Three options depending on intent:

### 1. You did not mean to grant it

Most common. Remove the rule. Services almost never need to destroy secret versions or metadata — a soft delete is sufficient.

```hcl
# Before
path "secret/destroy/*" {
  capabilities = ["update"]
}
path "secret/metadata/*" {
  capabilities = ["delete"]
}

# After: remove entirely
```

### 2. You meant soft delete

Use `secret/data/*` instead of `secret/metadata/*` or `secret/destroy/*`:

```hcl
path "secret/data/payment-svc/*" {
  capabilities = ["delete"]
}
```

This grants recoverable soft deletes on the service's own secrets, which is almost always what was wanted.

### 3. You genuinely need destructive operations

For compliance workflows (GDPR right-to-erasure, legal holds ending), scope the grant to a dedicated compliance policy, a specific path prefix, and a limited audience. Whitelist it in the spec:

```yaml
analyze:
  - check: secret_destroy
    allow_paths:
      - secret/destroy/user-data/*   # GDPR erasure role
      - secret/metadata/user-data/*
```

Document the policy. Audit quarterly.

## Configuration

```yaml
analyze:
  - check: secret_destroy
    disabled: false
    severity: warning         # default
    allow_paths: []
```

Compliance-heavy environments often promote this to `error`:

```yaml
analyze:
  - check: secret_destroy
    severity: error
```

## Implementation

The check walks every rule and tests whether:

- The path starts with `secret/destroy/` and capabilities include `update` or `delete`, or
- The path starts with `secret/metadata/` and capabilities include `delete`.

Paths outside those prefixes are not flagged by this check. For generic overprivilege on `secret/*`, see [`wildcard_paths`](wildcard-paths.md).
