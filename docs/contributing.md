---
layout: default
title: Contributing
---

[Home](.) |
[Getting Started](getting-started) |
[CLI Reference](cli-reference) |
[Architecture](architecture) |
[Roadmap](roadmap) |
[**Contributing**](contributing)

# Contributing

How to set up a development environment, conventions, and the pull request process.

---

## Ways to contribute

- **Report bugs** you encounter while using custos
- **Suggest features** that would make custos more useful
- **Improve documentation** — fix typos, clarify explanations, add examples
- **Write tests** that increase coverage
- **Review pull requests** opened by other contributors
- **Implement features** from the [Roadmap](roadmap)

Look for issues labeled [`good first issue`](https://github.com/timkrebs/custos/labels/good%20first%20issue) or [`help wanted`](https://github.com/timkrebs/custos/labels/help%20wanted).

---

## Development environment

### Prerequisites

- **Go** 1.24 or newer
- **Make** (GNU Make)
- **Git** 2.30+
- A POSIX-like shell (bash, zsh)

Optional:
- [`staticcheck`](https://staticcheck.dev/)
- [`govulncheck`](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
- A [Vault dev server](https://developer.hashicorp.com/vault/docs/get-started/developer-quickstart) for online evaluator work

### Setup

```bash
git clone https://github.com/timkrebs/custos.git
cd custos
go mod download
make build
./bin/custos --help
```

---

## Build targets

| Target | Purpose |
|:-------|:--------|
| `make build` | Build the binary into `./bin/` |
| `make test` | Run tests with race detector |
| `make test/cover` | Run tests and open HTML coverage report |
| `make audit` | Run formatting, vet, staticcheck, govulncheck |
| `make tidy` | Run `go mod tidy`, `go fix`, `go fmt` |
| `make clean` | Remove build artifacts |

**Always run `make audit` before pushing.** CI runs the same checks.

---

## Coding standards

- Format with `gofmt` (run `make tidy`)
- Pass `go vet ./...` and `staticcheck ./...`
- Public APIs must have Go doc comments
- Wrap errors with `fmt.Errorf("...: %w", err)`
- Avoid panics in library code
- Keep packages small and focused
- Justify new dependencies in the PR description

---

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/) as a guideline:

```
parser: support the `+` path glob introduced in Vault 1.16

The `+` glob matches a single path segment and is used in newer Vault
ACL policies. This change extends the matcher and adds table-driven
tests for both `*` and `+` semantics.

Fixes #42
```

- Imperative mood ("add feature", not "added feature")
- Subject line under 72 characters
- Reference issues with `Fixes #N` / `Refs #N`

---

## Pull request process

1. **Fork** and create a feature branch off `main` (`feat/short-description`, `fix/short-description`)
2. Make **small, logically scoped commits**
3. Run `make audit` and `make test` locally
4. Update documentation when behavior changes
5. **Open a PR** and fill in every section of the template
6. Respond to review feedback with additional commits (don't force-push during review)
7. **CI must be green** before merge

---

## License

custos is licensed under [MPL-2.0](https://github.com/timkrebs/custos/blob/main/LICENSE). All contributions are accepted under the same license.
