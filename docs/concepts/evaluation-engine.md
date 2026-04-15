---
description: The algorithm that turns a test case and a set of policies into an allow/deny result.
icon: gears
---

# Offline evaluation engine

The evaluation engine is the core of custos. It takes a parsed set of policies and a single test case, and it decides whether the request should be allowed or denied. Everything on this page is deterministic and local — no Vault instance, no network, no caching.

## Inputs and outputs

**Input**

- A list of parsed policies, each with rules (`PathRule`) containing a path pattern, capabilities, and source location.
- A test case: a request path, a list of capabilities to check, and an expected result.

**Output**

- A `TestResult` with:
  - `Pass` (bool): did the composed result match `expect`?
  - `Actual` (`allow` or `deny`): what the engine decided.
  - `Explanation`: a human-readable sentence.
  - `MatchedRule`: the rule that carried the decisive capability, with file and line.
  - `Composed`: the full composition trace — grants, denies, and per-policy contributions.

## The algorithm

{% stepper %}
{% step %}
### For each policy, find the most specific matching rule

For the test case's request path, iterate over the policy's rules and score each match. The highest-scoring match is the policy's contribution.

Specificity ranking (highest to lowest):
1. Exact match
2. Single-segment wildcard (`+`) with more literal segments
3. Trailing prefix (`*`) with more literal segments
4. Longer pattern wins ties

If no rule in a policy matches, that policy contributes nothing to the composed result.

See [Path matching](path-matching.md) for the ranking details.
{% endstep %}

{% step %}
### Union the granted capabilities across policies

Collect every capability granted by every contributing policy's winning rule and union them into a single set. If policy A grants `[read]` and policy B grants `[read, list]`, the union is `{read, list}`.
{% endstep %}

{% step %}
### Apply deny as a hard override

Walk the contributions again. If any winning rule has the `deny` capability, the final result is deny — regardless of what other policies granted. Record which policies contributed the deny so failures can explain it.
{% endstep %}

{% step %}
### Check the requested capabilities

For every capability the test case asks about, check whether it is in the composed grant set and whether the deny override is set.

- If deny is set: the result is deny.
- Else if every requested capability is in the grant set: the result is allow.
- Otherwise: the result is deny.
{% endstep %}

{% step %}
### Compare to the expectation

The test passes when the decided result matches `expect`. Build a `TestResult` with the decision, the matched rule, and the full composition trace. Hand it to the reporter.
{% endstep %}
{% endstepper %}

## An example run

Given two policies:

**`readonly.hcl`**

```hcl
path "secret/*" {
  capabilities = ["read", "list"]
}
```

**`payment-svc.hcl`**

```hcl
path "secret/data/billing-svc/*" {
  capabilities = ["deny"]
}
```

And the test case:

```yaml
- name: "billing denied"
  path: "secret/data/billing-svc/api-key"
  capabilities: [read]
  expect: deny
```

The engine does:

1. In `readonly.hcl`, `secret/*` is the only match. Grants `[read, list]`.
2. In `payment-svc.hcl`, `secret/data/billing-svc/*` is the only match. Carries `deny`.
3. Union grants: `{read, list}`. Deny override: true (from `payment-svc.hcl`).
4. Result: deny.
5. Expected: deny. Test passes.

The `TestResult` records both contributions so the reporter can print:

```
OK billing denied via deny rule "secret/data/billing-svc/*" in payment-svc.hcl
```

## Why this algorithm

The engine faithfully reproduces Vault's own ACL evaluation logic. The three properties that matter are:

1. **Most-specific match per policy.** Matches Vault's precedence rules so tests are not surprised by production behaviour.
2. **Union across policies.** A single entity holding multiple policies gets the union of their grants, which is how Vault handles multi-policy tokens.
3. **Deny is the hard override.** Deny cannot be reversed by any grant. This is the one rule every Vault user already knows.

If custos ever diverges from Vault on any of these, that is a bug to be fixed immediately. Trust in this engine is the single most valuable property custos has.

## Implementation pointers

Curious readers can walk the code directly:

- [`pkg/evaluator/offline.go`](https://github.com/timkrebs/custos/blob/main/pkg/evaluator/offline.go) — single-policy evaluation
- [`pkg/evaluator/composer.go`](https://github.com/timkrebs/custos/blob/main/pkg/evaluator/composer.go) — multi-policy composition
- [`pkg/evaluator/offline_test.go`](https://github.com/timkrebs/custos/blob/main/pkg/evaluator/offline_test.go) — the engine's own test suite, which is the most readable specification of the exact semantics
