---
description: How custos compares to other approaches for testing Vault policies.
icon: scale-balanced
---

# Comparison

There are three common ways teams try to verify Vault ACL policies today. This page walks through each and explains where custos fits.

## Feature matrix

| Feature | custos | vault-policy-testing | Manual testing |
|---|:---:|:---:|:---:|
| Offline testing (no Vault server) | Yes | No | No |
| Online verification | Planned (v0.3) | Yes | Yes |
| Multi-policy composition | Yes | No | Partial |
| Static security analysis | Yes | No | No |
| CI-native output (JUnit/JSON) | Yes | No | No |
| Runs in air-gapped environments | Yes | No | No |
| Per-check configuration | Yes | No | No |
| Per-policy failure provenance | Yes | No | No |

## Manual testing

The traditional approach: create a test token with a policy attached, then run Vault commands and observe whether they succeed.

**Pros**

- Zero new tooling.
- Tests the live system exactly as it behaves in production.

**Cons**

- Requires a running Vault instance.
- Not reproducible without scripting.
- No assertions — you eyeball the output.
- Does not scale to hundreds of paths.
- Cannot run pre-merge without exposing Vault credentials to CI.

**When it still helps.** Validating auth method behaviour (LDAP group mapping, OIDC claims) that policies alone cannot describe.

## vault-policy-testing and similar runners

Community runners that spin up a Vault dev server, write policies to it, and use a token to probe access.

**Pros**

- Exercises the real Vault binary.
- Can test auth method bindings alongside policies.

**Cons**

- Requires Vault to be installed and executable on the test runner.
- Slow to start up, especially in ephemeral CI.
- Brittle when Vault versions drift between developer machines.
- No static security analysis.
- No per-failure provenance across composed policies.

## custos

custos evaluates policies entirely offline with an engine that mirrors Vault's own ACL logic.

**Pros**

- No Vault server required. Runs anywhere Go binaries run.
- Deterministic, sub-second evaluation.
- Composition-aware, with per-policy provenance on failures.
- Ships with a static security analyzer and CI-ready output formats.
- Safe to drop into any pull request pipeline — no credentials to leak.

**Cons**

- Cannot test auth method behaviour on its own (use alongside manual or online verification for that).
- Online mode (live `sys/capabilities` verification) is on the v0.3 roadmap, not yet shipped.

**When it is the right choice.** Policy correctness testing, regression guards in pull requests, security reviews, compliance audits, and any environment where spinning up Vault is expensive or forbidden.

## Using them together

custos is designed to complement, not replace. A sensible setup looks like:

1. **Pull request CI:** custos runs on every change to `.hcl` or `.spec.yaml` files. Fast, deterministic, catches the vast majority of mistakes.
2. **Staging pipeline:** Terraform applies the policies to a staging Vault. Integration tests confirm auth method bindings and runtime behaviour.
3. **Production:** Terraform promotes the same policies. custos findings from step one serve as documentation for the security review.
