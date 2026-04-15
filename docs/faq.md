---
description: Common questions and troubleshooting for custos users.
icon: circle-question
---

# FAQ

## General

### What does custos mean?

*Custos* is Latin for *guardian*. The tool guards your Vault policies against misconfiguration.

### Is custos affiliated with HashiCorp or IBM?

No. custos is an independent open-source project released under MPL-2.0. It uses HashiCorp's open-source `hcl/v2` library to parse policies, but it has no official relationship with HashiCorp or IBM (HashiCorp's parent).

### Does custos require a running Vault instance?

No. Everything custos does today is offline. Online verification against a live Vault is planned for v0.3 — see the [roadmap](roadmap.md) — but the core value proposition is running without Vault.

### What versions of Vault does custos support?

custos supports any Vault version whose ACL policy syntax is a subset of what HashiCorp's `hcl/v2` parses. In practice that means every currently supported Vault release (1.x). If you hit a Vault-specific policy construct custos does not recognize, please [open an issue](https://github.com/timkrebs/custos/issues).

## Installation

### The install script failed. What do I do?

Read the script ([install.sh on GitHub](https://github.com/timkrebs/custos/blob/main/.build/install.sh)) and run the steps manually. The most common failure is PATH not including the install directory — typically `/usr/local/bin` or `$HOME/.local/bin`.

### Can I pin to a specific version?

Yes. The install script accepts a version argument:

```bash
curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash -s -- v0.1.0
```

Homebrew users can use `brew install custos@0.1.0` once a versioned formula is published. Docker users pin the tag (`ghcr.io/timkrebs/custos:v0.1.0`).

## Writing tests

### A test is failing but the policy looks right. How do I debug it?

Run with `-v`:

```bash
custos test -f spec.yaml -v
```

Verbose mode prints the matched rule, the policy file, and for composed suites the contribution of every policy. Nine times out of ten the issue is obvious from the trace.

### My list test is failing even though the policy grants list. Why?

Vault list operations target paths that end with `/`. A list test should look like:

```yaml
- name: "can list own secrets"
  path: "secret/data/payment-svc/"   # note the trailing slash
  capabilities: [list]
  expect: allow
```

Without the trailing slash, you are asking for a read on the literal entry named `payment-svc`, which is a different thing.

### Can I test capabilities individually?

Yes. Vault enforces every capability independently. Write separate tests for each:

```yaml
- name: "can create new secrets"
  path: "secret/data/payment-svc/new"
  capabilities: [create]
  expect: allow

- name: "can update existing secrets"
  path: "secret/data/payment-svc/existing"
  capabilities: [update]
  expect: allow
```

### How should I test implicit deny?

Any test that expects `deny` on a path not covered by any policy rule tests implicit deny. You do not need to do anything special:

```yaml
- name: "no access to unrelated paths"
  path: "some/unrelated/path"
  capabilities: [read]
  expect: deny
```

### Can I test multiple policies together?

Yes. Multi-policy composition is a first-class feature. List every policy the entity will hold under `policies:` and write tests against the composed behaviour. See [Policy composition](guides/policy-composition.md).

## Analyzers

### The analyzer flagged a rule we genuinely need. Can I suppress the warning?

Yes. Use the `analyze:` section of your spec with `allow_paths`:

```yaml
analyze:
  - check: sudo_capability
    allow_paths:
      - database/config/rotate
```

Prefer narrow whitelisting over blanket disabling. Every entry is a suppressed finding and should be documented.

### Can I add my own analyzer?

Not through configuration today — custom checks require a Go patch against `pkg/analyzer/`. Contributions of new checks are welcome. See [Contributing](contributing/README.md).

### Why is `policy_escalation` an error but `wildcard_paths` only a warning?

Policy escalation is exploitable in two API calls and has no mitigation other than removing the grant. Broad wildcards are risky but often have legitimate scopes (admin policies, break-glass roles). The defaults reflect that difference, but both are configurable per-project.

## CI

### My CI run reports zero tests even though custos ran. What happened?

Most likely the CI reporter is looking in the wrong place. Double-check the path in your workflow's test reporter step matches where custos wrote the JUnit XML. A missing `if: always()` on the reporter step is another common cause — without it, a failed run skips the reporter.

### Can custos run as a pre-commit hook?

Yes. See the [CI integration](guides/ci-integration.md) page for a sample `.pre-commit-config.yaml`.

### How do I run custos against many spec files in CI?

Loop over them and collect failures:

```bash
fail=0
for spec in specs/*.spec.yaml; do
  custos test -f "$spec" --format=junit > "results/$(basename "$spec" .spec.yaml).xml" || fail=1
done
exit $fail
```

### Can I get a pull request comment with the results?

Yes, via the JSON reporter and a step that posts to the PR. GitHub Actions example:

```yaml
- name: Summary
  if: always()
  run: |
    custos test -f spec.yaml --format=json > out.json
    echo "Passed: $(jq .summary.passed out.json)" >> $GITHUB_STEP_SUMMARY
    echo "Failed: $(jq .summary.failed out.json)" >> $GITHUB_STEP_SUMMARY
```

## Edge cases

### What if two rules in the same policy match the same path?

The most specific match wins. Exact path beats single-segment wildcard (`+`), which beats trailing prefix (`*`). See [Path matching](concepts/path-matching.md) for the precise ranking.

### What if two policies disagree?

Composition rules apply: each policy contributes its own most-specific match, the grants union, and any `deny` is a hard override. See [Policy composition semantics](concepts/composition.md).

### Can I use templated paths like `{{identity.entity.id}}`?

custos parses them without error but does not expand the template. Templated policy evaluation is a future feature — track it on the [roadmap](roadmap.md).

### What about Sentinel policies?

custos is ACL-layer only today. Sentinel integration is planned for v0.5.

## Other

### How do I report a bug?

Open an issue at [github.com/timkrebs/custos/issues](https://github.com/timkrebs/custos/issues) with a minimal reproduction, the output of `custos version`, and what you expected to happen.

### Where can I ask a question?

[GitHub Discussions](https://github.com/timkrebs/custos/discussions). Open a new discussion thread with as much context as you can share.

### Is there a Slack / Discord / mailing list?

Not yet. For now, discussions and issues on GitHub are the canonical channel.
