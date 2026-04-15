---
description: A tour of what custos can do today.
icon: list-check
---

# Features

## Offline policy evaluation

custos parses Vault HCL policies locally and evaluates test assertions without touching a Vault server. Path matching, capability checking, and deny overrides follow the same precedence rules as Vault itself, so results are faithful to production behaviour.

See [Offline evaluation engine](../concepts/evaluation-engine.md) for the algorithm and edge cases.

## Multi-policy composition

A Vault entity usually holds more than one policy. custos composes policies using Vault's own semantics:

1. For each policy independently, the most specific matching rule is selected.
2. Capabilities from every contributing policy are unioned.
3. Any `deny` capability on a matching rule overrides every grant, from any policy.

Failures include per-policy provenance so you can see exactly which policy granted or denied each capability. See [Policy composition](../guides/policy-composition.md).

## Static security analysis

Five built-in analyzers run on every test invocation and flag known anti-patterns:

| Check | Severity | Flags |
|---|---|---|
| [`wildcard_paths`](../analyzers/wildcard-paths.md) | warning | Trailing `*` paths with three or more capabilities |
| [`sudo_capability`](../analyzers/sudo-capability.md) | error | `sudo` outside `sys/` and `auth/token/` |
| [`root_token_create`](../analyzers/root-token-create.md) | error | `create` on `auth/token/create` |
| [`policy_escalation`](../analyzers/policy-escalation.md) | error | Mutations on `sys/policies/acl/*` |
| [`secret_destroy`](../analyzers/secret-destroy.md) | warning | Permanent KV v2 destroy or metadata delete |

Every check is configurable per-project: you can disable checks, whitelist specific paths, or promote a warning to an error when compliance requires it.

## CI-ready output

custos ships three reporters covering the environments teams actually run tests in:

- **Terminal** — colored, human-readable, respects `NO_COLOR`. The default.
- **JUnit XML** — standard Ant JUnit format consumed by GitHub Actions' `dorny/test-reporter`, GitLab, Jenkins, and Buildkite out of the box.
- **JSON** — stable schema with `schema_version`, suited to `jq` pipelines, drift detectors, and log ingestion. Ships a `--compact` mode for NDJSON.

See [Output formats](../output/terminal.md).

## Meaningful failures

When a test fails, custos tells you exactly why. A failure message includes:

- The expected and actual result (`allow` vs `deny`)
- The rule that matched and the policy file it came from
- For composition failures, which policies contributed each capability

No more guessing which of the four policies attached to a service account caused the regression.

## No Vault required

custos has no dependency on a live Vault server, no Vault token configuration, and no network access at test time. It runs in air-gapped CI, on read-only build runners, and inside containers that have never spoken to Vault. This is the single most important property of the tool.
