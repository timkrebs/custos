---
title: Roadmap
layout: default
nav_order: 5
---

# Roadmap
{: .no_toc }

Planned evolution of custos from first release through platform maturity.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Principles

1. **Ship `test` first.** A single command that works reliably is worth more than five commands that don't.
2. **Offline is the differentiator.** The ability to test policies without touching Vault is what makes custos unique.
3. **CI is the growth engine.** Adoption happens when someone drops custos into a pipeline and it catches a bad policy on a PR.
4. **Match Vault's behavior exactly.** If custos says "allow" and Vault says "deny," trust is gone.

---

## NOW — v0.1.0: "It works offline"
{: .d-inline-block }

In Progress
{: .label .label-yellow }

The credibility release. One command, one promise: you can test Vault policies without touching Vault.

**Launch bar:** Someone can `go install`, write a YAML test spec, point it at an HCL policy file, and get pass/fail results in the terminal.

### Must ship

- [x] Project scaffolding and CI/CD setup
- [x] HCL policy parser with full field support
- [x] YAML test spec loader and validator
- [x] Version command
- [ ] Offline policy evaluation engine
- [ ] `custos test` command with terminal reporter
- [ ] Comprehensive evaluation engine tests

### Explicitly not in v0.1.0

Online mode, `scan`, `init`, `validate`, JUnit/JSON reporters, enterprise features.

---

## NEXT — v0.2.0: "It fits in CI"
{: .d-inline-block }

Planned
{: .label .label-blue }

Once the core works, the next unlock is CI/CD integration.

### Must have

- [ ] JUnit XML reporter (`--format junit`)
- [ ] Proper exit codes (`--fail-on-warn`)
- [ ] `custos validate` command

### Should have

- [ ] JSON reporter (`--format json`)
- [ ] `custos init --from policy.hcl`
- [ ] Verbose mode (`-v`)

### Could have

- [ ] First-party GitHub Action (`timkrebs/custos-action@v1`)

---

## LATER — v0.3.0 to v0.5.0: "It's the platform"
{: .d-inline-block }

Planned
{: .label .label-blue }

### v0.3.0 — Online mode and security scanning

- [ ] Online mode (`--vault-addr`, `--vault-token`)
- [ ] `custos scan` command
- [ ] Severity filtering (`--severity`)

### v0.4.0 — Deep analysis

- [ ] Overprivilege detection
- [ ] Policy conflict detection
- [ ] Path coverage reporting

### v0.5.0 — Enterprise

- [ ] Namespace-aware evaluation
- [ ] Sentinel policy integration
- [ ] Timeout and retry configuration

---

## Version history

| Version | Status | Theme |
|:--------|:-------|:------|
| v0.1.0 | In progress | Offline policy testing |
| v0.2.0 | Planned | CI/CD integration |
| v0.3.0 | Planned | Online mode and scanning |
| v0.4.0 | Planned | Deep analysis |
| v0.5.0 | Planned | Enterprise features |
