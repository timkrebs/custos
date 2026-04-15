---
description: How custos evaluates the combined effect of multiple policies on one entity.
icon: layer-group
---

# Policy composition

A single Vault entity almost always holds more than one policy. A service might receive a shared `readonly` policy, a service-specific policy, and a team policy, and Vault evaluates all three together for every request. custos composes policies using the same semantics, so test results match production.

## The rules

custos composes policies in three steps:

{% stepper %}
{% step %}
### Match each policy independently

For the request path, each policy is checked on its own. Within a policy, the most specific matching rule wins — an exact path beats a single-segment wildcard (`+`), which beats a trailing prefix (`*`).
{% endstep %}

{% step %}
### Union the granted capabilities

Capabilities from every matching rule across every contributing policy are combined into a single set. If policy A grants `[read]` and policy B grants `[list, create]`, the composed grant is `[read, list, create]`.
{% endstep %}

{% step %}
### Apply deny as a hard override

If *any* matching rule in *any* contributing policy carries the `deny` capability, the composed result is deny. No grant from any other policy can reverse it.
{% endstep %}
{% endstepper %}

## A worked example

Consider two policies attached to the same entity.

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

Here is how custos composes them for a few representative requests:

| Request | readonly contribution | payment-svc contribution | Composed result |
|---|---|---|---|
| `secret/data/payment-svc/db-creds` read | grants `read, list` | grants `read, list` | allow |
| `secret/data/billing-svc/api-key` read | grants `read, list` | **denies** | **deny** |
| `secret/data/other/foo` read | grants `read, list` | no match | allow |
| `pki_int/issue/payment-svc` create | no match | grants `create, update` | allow |
| `sys/seal` sudo | no match | no match | deny (implicit) |

The fifth row illustrates implicit deny: if no policy has a rule that matches the path, the request is denied. This is how Vault works, and custos reproduces it.

## Writing the composed spec

Point the spec at both policies and write tests that cover the composed behaviour:

```yaml
suite: "composed-policy-evaluation"

policies:
  - path: ../policies/readonly.hcl
  - path: ../policies/payment-svc.hcl

tests:
  - name: "readonly grants read on any secret"
    path: "secret/data/some-service/config"
    capabilities: [read]
    expect: allow

  - name: "billing denied despite readonly allowing read"
    path: "secret/data/billing-svc/api-key"
    capabilities: [read]
    expect: deny

  - name: "can issue certs via payment-svc policy"
    path: "pki_int/issue/payment-svc"
    capabilities: [create, update]
    expect: allow

  - name: "cannot write via readonly alone"
    path: "secret/data/some-service/config"
    capabilities: [create]
    expect: deny
```

Run it the same way:

```bash
custos test -f specs/composed.spec.yaml -v
```

The `-v` flag prints a contribution trace for every test, so you can see which policy granted or denied each capability.

## Per-policy provenance on failures

When a composed test fails, custos tells you which policies contributed what:

```
    FAIL billing denied despite readonly allowing read
      expected: deny, got: allow
      contributions:
        readonly     secret/*                 GRANT [read, list]
        payment-svc  secret/data/billing-svc/* no match
```

If the `deny` rule in `payment-svc.hcl` were accidentally removed, the failure message points straight at it. This is the single biggest win of composition-aware testing.

## Tips

- **Order does not matter.** The composition algorithm is commutative. You can list policies in any order in the spec.
- **Most-specific wins per policy.** If a policy has both `secret/*` and `secret/data/payment-svc/*`, the second wins for payment-svc paths *within that policy*. Composition across policies happens on top of per-policy matching.
- **Deny is the big hammer.** Use it when you genuinely want to block a path even when another policy grants it. Do not use it to "comment out" a rule.

See [Policy composition semantics](../concepts/composition.md) for the formal algorithm and edge cases.
