---
description: Every Vault capability custos recognizes and what it means.
icon: key
---

# Capabilities

Vault capabilities are the verbs of the ACL system. A policy rule grants a set of capabilities on a set of paths. custos recognizes every capability Vault itself supports.

## The full list

| Capability | HTTP verb(s) | Description |
|---|---|---|
| `create` | `POST` | Create a new entry. Does not update if the entry already exists. |
| `read` | `GET` | Read an existing entry. |
| `update` | `POST`, `PUT` | Update an existing entry. Vault treats `create` and `update` as distinct, so rules often list both. |
| `patch` | `PATCH` | Partial update (newer Vault versions, KV v2). |
| `delete` | `DELETE` | Delete an entry. |
| `list` | `LIST` | List entries under a path. Requires a trailing slash on the request path. |
| `sudo` | n/a | Grants access to sudo-protected endpoints (mostly `sys/` operations). |
| `deny` | n/a | Hard-deny override. Any matching `deny` blocks the request regardless of other grants. |
| `subscribe` | n/a | Subscribe to the event stream. |
| `recover` | n/a | Recover deleted KV v2 versions. |

## Capability semantics

### create vs update

Vault enforces `create` and `update` separately. A rule that grants only `create` cannot modify an existing entry, and a rule that grants only `update` cannot create a new one. Most real-world rules grant both:

```hcl
path "secret/data/payment-svc/*" {
  capabilities = ["create", "read", "update", "delete"]
}
```

Test them independently when the distinction matters:

```yaml
- name: "can create new secrets"
  path: "secret/data/payment-svc/new-secret"
  capabilities: [create]
  expect: allow

- name: "can update existing secrets"
  path: "secret/data/payment-svc/existing-secret"
  capabilities: [update]
  expect: allow
```

### list requires a trailing slash

To list keys under a path, the request path must end with `/` and the policy must grant `list` on the matching rule:

```yaml
- name: "can list secret keys"
  path: "secret/data/payment-svc/"  # trailing slash
  capabilities: [list]
  expect: allow
```

Without the trailing slash, you are asking for a read on a specific entry named `payment-svc`, which is a different thing.

### sudo is for sys only

`sudo` unlocks a small set of privileged endpoints, almost all under `sys/`. `sudo` on any other path is almost certainly a mistake, and the [`sudo_capability`](../analyzers/sudo-capability.md) analyzer will flag it.

Legitimate `sudo` paths include:

- `sys/unseal`
- `sys/rotate`
- `sys/audit`
- `auth/token/renew`

### deny is a hard override

`deny` is not "do nothing." It is a hard override that blocks the request regardless of what any other rule or policy grants. Use it when you genuinely want to block a path that another policy would otherwise allow.

```hcl
path "secret/data/billing-svc/*" {
  capabilities = ["deny"]
}
```

## Testing multiple capabilities at once

A test with multiple capabilities passes when *every* listed capability is granted. This is useful for "can the service fully manage this entry?" assertions:

```yaml
- name: "can fully manage own secrets"
  path: "secret/data/payment-svc/db-creds"
  capabilities: [create, read, update, delete]
  expect: allow
```

For deny assertions, the test passes when *any* capability is denied. If your policy grants read but denies write, this passes:

```yaml
- name: "cannot modify shared config"
  path: "secret/data/shared/config"
  capabilities: [create, update, delete]
  expect: deny
```

## Implicit deny

If no rule matches the request path at all, the result is implicit deny. This counts as `deny` for assertion purposes:

```yaml
- name: "no access to unrelated paths"
  path: "some/totally/unrelated/path"
  capabilities: [read]
  expect: deny
```

Implicit deny is how Vault handles any path that is not explicitly covered by a policy. custos reproduces it exactly.
