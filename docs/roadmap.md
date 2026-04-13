---
layout: default
title: Roadmap
---

[Home](.) |
[Getting Started](getting-started) |
[CLI Reference](cli-reference) |
[Architecture](architecture) |
[**Roadmap**](roadmap) |
[Contributing](contributing)

# Roadmap

Planned evolution of custos from first release through platform maturity.

---

## Principles

1. **Ship `test` first.** A single command that works reliably is worth more than five commands that don't.
2. **Offline is the differentiator.** The ability to test policies without touching Vault is what makes custos unique.
3. **CI is the growth engine.** Adoption happens when someone drops custos into a pipeline and it catches a bad policy on a PR.
4. **Match Vault's behavior exactly.** If custos says "allow" and Vault says "deny," trust is gone.

---

## v0.1.0: "It works offline" — Released

The credibility release. One command, one promise: you can test Vault policies without touching Vault.

- [x] Project scaffolding and CI/CD setup
- [x] HCL policy parser with full field support
- [x] YAML test spec loader and validator
- [x] Offline policy evaluation engine
- [x] `custos test` command with terminal reporter
- [x] Comprehensive evaluation engine tests
- [x] Version command with `--json` flag
- [x] Build and release infrastructure (GoReleaser, Docker, install script, Homebrew)

---

## v0.2.0: "It fits in CI" — Planned

Once the core works, the next unlock is CI/CD integration.

- [ ] JUnit XML reporter (`--format junit`)
- [ ] JSON reporter (`--format json`)
- [ ] Proper exit codes (`--fail-on-warn`)
- [ ] `custos validate` command
- [ ] `custos init --from policy.hcl`
- [ ] Verbose mode (`-v`) improvements

---

## v0.3.0 to v0.5.0: "It's the platform" — Planned

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
| v0.1.0 | **Released** | Offline policy testing |
| v0.2.0 | Planned | CI/CD integration |
| v0.3.0 | Planned | Online mode and scanning |
| v0.4.0 | Planned | Deep analysis |
| v0.5.0 | Planned | Enterprise features |
