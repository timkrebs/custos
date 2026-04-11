# Custos Roadmap

> **Custos** — Latin for *guardian*. The missing `terraform plan` for HashiCorp Vault policies.

This document describes the planned evolution of custos from first working release through platform maturity. The roadmap is organized as **Now / Next / Later** to communicate intent without false precision on dates.

## Principles

1. **Ship `test` first.** A single command that works reliably is worth more than five commands that don't.
2. **Offline is the differentiator.** The ability to test policies without touching Vault is what makes custos unique. Protect that.
3. **CI is the growth engine.** Adoption happens when someone drops custos into a pipeline and it catches a bad policy on a PR.
4. **Match Vault's behavior exactly.** If custos says "allow" and Vault says "deny," trust is gone. Fidelity to Vault's evaluation logic is non-negotiable.

---

## NOW — v0.1.0: "It works offline"

The credibility release. One command, one promise: you can test Vault policies without touching Vault.

**Launch bar:** Someone can `go install`, write a YAML test spec, point it at an HCL policy file, and get pass/fail results in the terminal.

### Must ship

- **Rename to custos** — module path, binary name, README, GoReleaser config, CI workflows, all references.
- **Refactor HCL parser** — the current `pkg/parser/hcl.go` parses a generic config schema. It needs to correctly parse Vault policy `path` blocks with `capabilities`, `allowed_parameters`, `denied_parameters`, `min_wrapping_ttl`, `max_wrapping_ttl`, and glob patterns.
- **Offline policy evaluation engine** — given a policy (or set of policies) and a path + capability assertion, return allow/deny. Must handle glob matching (`*`, `+` segments), capability precedence, and deny overrides. Use `hashicorp/vault/sdk` for parsing fidelity.
- **YAML test spec parser** — parse test spec files that define assertions (path, expected capabilities, expected result).
- **`custos test` command with terminal reporter** — colored pass/fail output with clear context on failures (which policy granted/denied, which line).
- **Test coverage on the evaluation engine** — comprehensive tests covering glob patterns, precedence rules, deny semantics, and edge cases.
- **Version command** — `custos version` (already implemented, needs rename).

### Explicitly not in v0.1.0

Online mode, `scan`, `init`, `validate`, JUnit/JSON reporters, enterprise features. All cut. Ship small, ship right.

### Key risk

The HCL parser refactor is the riskiest item. Vault's path matching has nuanced behavior around glob precedence, longest-prefix matching, and capability merging across multiple policies. Spike on this first before touching anything else.

---

## NEXT — v0.2.0: "It fits in CI"

The adoption release. Once the core works, the next unlock is CI/CD integration. This is where someone adds custos to a GitHub Actions workflow and it catches a bad policy change on a pull request.

### Must have

- **JUnit XML reporter** (`--format junit`) — GitHub Actions, GitLab CI, Jenkins, and CircleCI all parse JUnit natively. This is the fastest path to CI integration.
- **Proper exit codes** (`--fail-on-warn`) — CI gates need non-zero exit codes on failure. Configurable severity threshold.
- **`custos validate`** — syntax-check a test spec file without running it. Fast feedback for spec authors and a natural pre-commit hook.

### Should have

- **JSON reporter** (`--format json`) — programmatic consumption, piping to `jq`, integration with custom tooling.
- **`custos init --from policy.hcl`** — generate a test spec skeleton from an existing policy file. Dramatically lowers onboarding friction. With `--all-paths`, generate an assertion for every path in the policy.
- **Verbose mode** (`-v`) — detailed evaluation trace showing which policy matched, which line, and why.

### Could have

- **First-party GitHub Action** (`timkrebs/custos-action@v1`) — high leverage for adoption but can ship as a follow-up.

---

## LATER — v0.3.0 to v0.5.0: "It's the platform"

Each of these opens a new use case or buyer. Sequence based on user demand.

### v0.3.0 — Online mode and security scanning

- **Online mode** (`--vault-addr`, `--vault-token`) — verify assertions against a live Vault instance using `sys/capabilities-self`. Validates that offline results match reality. Supports `--vault-namespace` for Enterprise.
- **`custos scan`** — security scanner that detects dangerous patterns in policy files without requiring a test spec. Finds wildcard paths, sudo capabilities outside `sys/`, overly broad `create`/`delete` grants, and missing deny rules.
- **Severity filtering** (`--severity warning`) — control which findings are reported.

### v0.4.0 — Deep analysis

- **Overprivilege detection** — given a set of policies and a set of test specs, identify capabilities that are granted but never tested. "Policy X grants delete on `secret/data/*` but no test covers this."
- **Policy conflict detection** — find contradictions between policies (one allows, another denies the same path) and surface them with context.
- **Path coverage reporting** — "Your policies cover 73% of declared paths" as a metric for security reviews.

### v0.5.0 — Enterprise

- **Namespace-aware evaluation** — support Vault Enterprise namespaces in offline evaluation and online mode.
- **Sentinel policy integration** — evaluate Sentinel policies alongside ACL policies for Enterprise customers.
- **Timeout and retry configuration** — production-grade online mode with configurable timeouts and retry behavior.

---

## Parked

These ideas are interesting but explicitly not on the roadmap yet. Revisit post-v1.0 or when user demand justifies them.

- **Vault dev server integration** — spin up a Vault dev server for hybrid offline/online testing. The pure offline mode is the differentiator; don't dilute it until there's clear demand.
- **Grafana dashboard template** — export scan results over time for visualization. Polish, not product.
- **Policy-as-code generation** — generate Vault policies from test specs (reverse direction). Interesting but different product.

---

## Version history

| Version | Status | Theme |
|---------|--------|-------|
| v0.1.0 | In progress | Offline policy testing — `custos test` |
| v0.2.0 | Planned | CI/CD integration — reporters, `init`, `validate` |
| v0.3.0 | Planned | Online mode and security scanning |
| v0.4.0 | Planned | Deep analysis — overprivilege, conflicts, coverage |
| v0.5.0 | Planned | Enterprise — namespaces and Sentinel |
