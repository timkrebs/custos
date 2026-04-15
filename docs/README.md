---
description: >-
  The missing terraform plan for HashiCorp Vault policies. Test Vault ACL
  policies offline, catch overprivileged access, and integrate results into CI.
icon: shield-halved
---

# custos

**custos** (Latin for *guardian*) is a command-line tool that lets you write test specifications for HashiCorp Vault ACL policies, evaluate them offline without a running Vault instance, and detect security anti-patterns before they reach production.

Think of it as the missing `terraform plan` for Vault policies: every change goes through review, but nobody can actually verify *"if I apply this policy, can entity X access path Y?"* until the policy is already live. custos closes that gap.

```
$ custos test -f payment-svc.spec.yaml

  payment-service-policies

    OK   can read its own secrets              (secret/data/payment-svc/db-creds)
    OK   cannot read billing secrets           (secret/data/billing-svc/api-key)
    OK   can issue short-lived certificates    (pki_int/issue/payment-svc)
    FAIL no access to sys backend              (sys/seal)
      expected: deny, got: allow via policy "admin-legacy"

  9 passed, 1 failed, 0 skipped
```

## What custos gives you

{% columns %}
{% column %}
### For developers

Write policy tests the same way you write unit tests. Catch regressions on every pull request. Refactor policies with confidence knowing the old behaviour is pinned by assertions.
{% endcolumn %}

{% column %}
### For security teams

Detect wildcard overreach, sudo leaks, root token minting, policy self-escalation, and destructive KV operations with a static analyzer that runs in under a second on the average policy set.
{% endcolumn %}
{% endcolumns %}

## Quick links

<table data-view="cards">
  <thead>
    <tr>
      <th>Section</th>
      <th data-card-target data-type="content-ref">Target</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Installation</strong> — get custos running in under a minute</td>
      <td><a href="getting-started/installation.md">Installation</a></td>
    </tr>
    <tr>
      <td><strong>Quickstart</strong> — write and run your first policy test</td>
      <td><a href="getting-started/quickstart.md">Quickstart</a></td>
    </tr>
    <tr>
      <td><strong>CLI reference</strong> — every command, flag, and exit code</td>
      <td><a href="reference/cli.md">CLI reference</a></td>
    </tr>
    <tr>
      <td><strong>Spec format</strong> — the full YAML schema</td>
      <td><a href="reference/spec-format.md">Spec format</a></td>
    </tr>
    <tr>
      <td><strong>Security analyzers</strong> — what custos flags and why</td>
      <td><a href="analyzers/README.md">Analyzers</a></td>
    </tr>
    <tr>
      <td><strong>CI integration</strong> — GitHub Actions, GitLab, Jenkins</td>
      <td><a href="guides/ci-integration.md">CI integration</a></td>
    </tr>
  </tbody>
</table>

## Project status

custos is an open-source project under active development. The current release line ships offline policy testing, multi-policy composition, a JSON/JUnit/terminal reporter, and five static security analyzers. The [roadmap](roadmap.md) tracks what is coming next.

{% hint style="info" %}
custos is an independent open-source project licensed under [MPL-2.0](https://github.com/timkrebs/custos/blob/main/LICENSE). It is not affiliated with or endorsed by HashiCorp or IBM.
{% endhint %}
