---
description: The problem custos exists to solve, and why offline policy testing matters.
icon: circle-question
---

# Why custos

## The problem

HashiCorp Vault policies are written in HCL, applied to a live Vault instance, and then tested by creating tokens and manually running commands like `vault kv get`. There is no structured way to answer a simple question:

> If I apply this policy, can entity X access path Y?

This shows up as a handful of concrete problems:

1. **No pre-merge validation.** Policy changes go through pull request review, but nobody can confirm correctness until the policy is already deployed.
2. **No regression testing.** A well-intentioned refactor can silently grant or revoke access to paths, and the first signal is usually a production incident.
3. **No static security analysis.** Wildcard paths that leak `delete`, `sudo` on non-system endpoints, and self-escalating `sys/policies/acl/*` grants hide in plain sight.
4. **No CI integration.** Terraform can `plan` infrastructure changes. There is no equivalent for Vault ACL policies.

Teams end up with policies that are technically correct on the day they are written and quietly drift from the intent as services, owners, and backends change.

## What custos does

custos reads your Vault HCL policies and a YAML test spec that describes what access *should* look like, then answers the access question offline, with no running Vault instance required.

```
HCL policies + YAML test spec
            |
            v
     custos evaluate
            |
            v
    pass / fail / findings
```

The evaluation engine mirrors Vault's own ACL matching rules: most-specific match wins, `deny` overrides any grant, and capabilities union across policies that a single entity holds. That means a test that passes in custos will behave the same way in production Vault.

## Why offline matters

Existing tools for testing Vault policies require a Vault instance — either a dev server or a real cluster. That is fine on a laptop, but it falls apart in the two places policy testing is most valuable:

- **Air-gapped and regulated environments.** You cannot stand up a Vault instance in many CI pipelines, and you certainly cannot expose production tokens to a test runner.
- **Pre-merge pull request checks.** Spinning up Vault on every pull request is slow, brittle, and leaks credentials into CI logs.

custos evaluates everything locally, deterministically, and in under a second on the average policy set. That is the differentiator, and it is what makes custos safe to drop into any pipeline.

## Where custos fits

custos is a complement to, not a replacement for, the tools you already use:

- **Terraform** provisions the Vault instance and writes policies to it. custos tests those policies before Terraform applies them.
- **Vault Sentinel** provides fine-grained, runtime-evaluated policies in Vault Enterprise. custos covers the ACL layer underneath and is free.
- **Manual testing** with `vault token create` and `vault kv get` still catches online-only issues like auth method behaviour. custos catches everything that can be decided from the policy document alone, which is the vast majority of mistakes.
