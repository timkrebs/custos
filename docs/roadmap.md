---
description: Where custos is going, organized as Now, Next, and Later.
icon: map
---

# Roadmap

The roadmap is organized as **Now**, **Next**, and **Later** to communicate intent without committing to dates that are impossible to keep. The source of truth for the current state is [`ROADMAP.md`](https://github.com/timkrebs/custos/blob/main/ROADMAP.md) in the repository.

## Guiding principles

1. **Ship `test` first.** A single command that works reliably is worth more than five commands that do not.
2. **Offline is the differentiator.** The ability to test policies without touching Vault is what makes custos unique. Protect that property first.
3. **CI is the growth engine.** Adoption happens when someone drops custos into a pipeline and it catches a bad policy on a pull request. Every release prioritizes CI ergonomics.
4. **Match Vault's behaviour exactly.** If custos says allow and Vault says deny, trust is gone. Fidelity to Vault's evaluation logic is non-negotiable.

## Version overview

| Version | Status | Theme |
|---|---|---|
| v0.1.0 | Released | Offline policy testing — `custos test` |
| v0.2.0 | In progress | CI integration, reporters, static analyzer |
| v0.3.0 | Planned | Online mode and security scanning |
| v0.4.0 | Planned | Deep analysis — overprivilege, conflicts, coverage |
| v0.5.0 | Planned | Enterprise — namespaces and Sentinel |

## Now — v0.1.0 and v0.2.0

### v0.1.0: "It works offline" (released)

The credibility release. One command, one promise: test Vault policies without touching Vault.

- HCL parser for Vault policy `path` blocks (capabilities, allowed/denied parameters, wrapping TTLs, glob patterns)
- Offline evaluation engine with Vault-accurate precedence, deny semantics, and multi-policy composition
- YAML test spec parser with strict field validation
- `custos test` command with colored terminal reporter
- Comprehensive test coverage on the evaluation engine

### v0.2.0: "It fits in CI" (in progress)

The adoption release. Once the core works, the next unlock is CI integration — someone adds custos to a GitHub Actions workflow and it catches a bad policy change on a pull request.

Already shipped as part of this release line:

- JUnit XML reporter (`--format=junit`)
- JSON reporter (`--format=json`, `--compact`)
- Proper exit codes with `--fail-on-warn`
- Verbose mode with multi-policy provenance
- Overprivilege analyzer with five built-in checks
- Per-check configuration via the `analyze:` section

Still planned for this release line:

- `custos validate` — syntax-check a spec file without evaluating it
- `custos init --from policy.hcl` — generate a spec skeleton from an existing policy
- First-party GitHub Action

## Next — v0.3.0

### Online mode and standalone scanning

Online mode verifies custos's offline results against a live Vault instance using `sys/capabilities-self`. It is the proof point that custos's offline engine matches production behaviour, and it unlocks a second use case: "audit my existing Vault."

- `custos test --vault-addr --vault-token` — dual-mode evaluation (offline plus online verification)
- `custos scan` — standalone security scan that runs the analyzer without a spec file
- Namespace support for Vault Enterprise
- Severity filtering (`--severity warning`)

## Later — v0.4.0 and v0.5.0

### v0.4.0: Deep analysis

- **Overprivilege detection.** Given a set of policies and a set of tests, identify capabilities granted but never asserted on. "Policy X grants `delete` on `secret/data/*` but no test covers this."
- **Policy conflict detection.** Find contradictions between policies and surface them with context.
- **Path coverage reporting.** "Your tests cover 73% of the capability surface in your policies" as a security review metric.

### v0.5.0: Enterprise

- **Namespace-aware evaluation.** Vault Enterprise namespaces in both offline and online mode.
- **Sentinel policy integration.** Evaluate Sentinel alongside ACL policies.
- **Production-grade online mode.** Timeouts, retries, and audit-friendly output.

## Parked

Ideas that are interesting but explicitly not on the roadmap. Revisit post-1.0 or when user demand justifies them.

- **Vault dev server integration.** Spinning up a Vault dev server for hybrid offline/online testing. The pure offline mode is the differentiator; diluting it is a mistake until there is clear demand.
- **Grafana dashboard template.** Export scan results over time for visualization. Polish, not product.
- **Policy-as-code generation.** Generate Vault policies *from* test specs (the reverse direction). Interesting but a different product.

## How to influence the roadmap

Open a [discussion](https://github.com/timkrebs/custos/discussions) with the use case you care about. Concrete scenarios — "my team ran into this problem; custos could solve it by..." — beat abstract feature requests. Bug reports and pull requests both move items up the priority list faster than anything else.
