---
description: How custos decides which rule wins when multiple patterns match a request.
icon: route
---

# Path matching

Vault has a small, precise set of rules for matching a request path against policy rule patterns. custos implements them exactly. This page walks through every case.

## The three pattern kinds

| Kind | Syntax | Example |
|---|---|---|
| Exact | literal path | `sys/seal` |
| Single-segment wildcard | `+` in any segment | `secret/data/+/config` |
| Trailing prefix | `*` at the end | `secret/data/payment-svc/*` |

`*` is only a wildcard when it is the last character. A `*` anywhere else is a literal. `+` is always a single-segment wildcard.

## Precedence

When more than one rule in the *same policy* matches a request, the most specific one wins. Specificity ranking:

1. **Exact match** beats everything else.
2. **Single-segment wildcard match** beats trailing prefix.
3. **Trailing prefix match** is the weakest.

Within the same kind, the rule with more literal (non-wildcard) segments wins. If there is still a tie, the longest rule wins.

## Worked examples

Consider a policy with these three rules:

```hcl
path "secret/data/payment-svc/*" {
  capabilities = ["read", "list"]
}

path "secret/data/payment-svc/prod/db-creds" {
  capabilities = ["deny"]
}

path "secret/data/+/public" {
  capabilities = ["read"]
}
```

And these request paths:

| Request | Matches | Winner | Result |
|---|---|---|---|
| `secret/data/payment-svc/prod/db-creds` | prefix rule + exact rule | **exact** (`.../prod/db-creds`) | deny |
| `secret/data/payment-svc/staging/db-creds` | prefix rule only | prefix | `read, list` |
| `secret/data/billing-svc/public` | single-seg wildcard only | `+/public` | `read` |
| `secret/data/payment-svc/public` | prefix rule + single-seg wildcard | **single-seg wildcard** (`+/public`) | `read` |
| `secret/data/payment-svc` | nothing (no trailing slash) | no match | implicit deny |
| `sys/seal` | nothing | no match | implicit deny |

Row four is the subtle one. The prefix rule would match, but the single-segment wildcard rule is more specific, so it wins. Only its capabilities (`read`) apply.

## Trailing `*` requires a following character

A trailing `*` matches any path that *starts with* the prefix up to the asterisk. It does *not* match the prefix itself without the trailing segment:

- `secret/data/payment-svc/*` matches `secret/data/payment-svc/db-creds`
- `secret/data/payment-svc/*` does **not** match `secret/data/payment-svc` or `secret/data/payment-svc/`

To cover the listing case, write an explicit rule with a trailing slash or a separate rule for list operations.

## list and trailing slashes

Vault list operations are distinct from read operations and must target a path that ends with `/`. If you write:

```yaml
- name: "can list own secrets"
  path: "secret/data/payment-svc/"
  capabilities: [list]
  expect: allow
```

custos matches that against a rule whose pattern also ends with `/` (or a `*` prefix that covers it). A common mistake is writing the test path without the trailing slash, which then looks up the literal entry named `payment-svc` and fails.

## Implicit deny

If no rule in any policy matches the request path, the composed result is deny. custos treats implicit deny identically to an explicit `deny` capability for test assertion purposes: a test expecting `deny` passes, a test expecting `allow` fails.

## What `+` does not do

`+` matches exactly one path segment, not zero and not more. So:

- `secret/+/config` matches `secret/foo/config` but **not** `secret/foo/bar/config` and **not** `secret/config`.
- Two `+` segments require two literal segments: `secret/+/+/config` matches `secret/a/b/config` only.

Use multiple `+` for multiple wildcard segments, and a trailing `*` when the number of trailing segments is variable.

## Across policies

Everything on this page applies *within* a single policy. When multiple policies compose, each policy first selects its own most-specific matching rule, and then custos unions the results with deny as a hard override. See [Policy composition semantics](composition.md).
