---
description: Static security checks custos runs against your Vault policies.
icon: shield-check
---

# Security analyzers

The custos analyzer runs a set of static security checks against every parsed policy and emits findings for anti-patterns it recognizes. Analysis is independent of test pass/fail: a policy can pass every assertion and still produce warnings, and a policy can fail tests without producing any findings.

Analysis runs automatically as part of `custos test`. There is no separate command to invoke.

## Built-in checks

| Check | Severity | What it flags |
|---|---|---|
| [`wildcard_paths`](wildcard-paths.md) | warning | Trailing `*` paths that grant three or more capabilities |
| [`sudo_capability`](sudo-capability.md) | error | `sudo` on any path outside `sys/` or `auth/token/` |
| [`root_token_create`](root-token-create.md) | error | `create` capability on `auth/token/create` |
| [`policy_escalation`](policy-escalation.md) | error | `create` or `update` on `sys/policies/acl/*` or `sys/policy/*` |
| [`secret_destroy`](secret-destroy.md) | warning | Permanent KV v2 destroy or metadata delete |

Click through to each check for the full detection logic, rationale, and remediation guidance.

## Finding anatomy

Every finding carries:

- **Check ID** — the stable identifier (`wildcard_paths`, `sudo_capability`, etc.)
- **Severity** — `info`, `warning`, or `error`
- **Message** — a human-readable explanation
- **File and line** — the 1-based source location in the HCL file
- **Path** — the rule path that triggered the finding
- **Capabilities** — the capabilities on the offending rule

Findings are deterministically sorted by file, line, then check ID so reports stay stable across runs.

## Configuration

Per-check configuration lives in the spec file under the `analyze` key:

```yaml
analyze:
  - check: sudo_capability
    disabled: false
    allow_paths:
      - sys/unseal
      - database/config/rotate
    severity: error

  - check: wildcard_paths
    disabled: true
```

### Options

| Option | Type | Effect |
|---|---|---|
| `check` | string | The check ID to configure |
| `disabled` | bool | Turn this check off entirely |
| `allow_paths` | list[string] | Whitelist rule paths. Uses Vault glob syntax. Findings that would otherwise fire on a matching path are suppressed. |
| `severity` | string | Override the default severity. Accepts `info`, `warning`, or `error`. |

The `allow_paths` list uses the same pattern syntax as Vault policies themselves. A whitelist entry `database/config/*` covers every rule under that path.

## Failing builds on findings

By default, analyzer findings are informational: they print in the reporter but do not affect the exit code. To fail the build when any warning or error is emitted, pass `--fail-on-warn`:

```bash
custos test -f spec.yaml --fail-on-warn
```

This is the right setting for teams that treat static analysis as a blocking gate. Teams that want to phase the analyzer in gradually can leave `--fail-on-warn` off and use the findings as guidance.

## Adding exceptions responsibly

Every `allow_paths` entry is a suppressed finding. Treat the `analyze` section of your spec as a living audit log: each exception should have a comment explaining why it is safe.

```yaml
analyze:
  - check: sudo_capability
    allow_paths:
      # legitimate: rotate database root creds at boot
      - database/config/rotate
      # break-glass only; reviewed quarterly
      - sys/unseal
```

Resist the temptation to silence a finding by adding `disabled: true`. Prefer narrow path exceptions so new occurrences of the anti-pattern still get caught.
