# Contributing to vaultspec

Thanks for your interest in contributing to **vaultspec**! This document describes
how to set up a development environment, the conventions used in this repository,
and how to get your changes reviewed and merged.

vaultspec is an independent open-source project. It is **not** affiliated with or
endorsed by HashiCorp or IBM.

---

## Table of contents

- [Code of Conduct](#code-of-conduct)
- [Ways to contribute](#ways-to-contribute)
- [Reporting bugs](#reporting-bugs)
- [Suggesting enhancements](#suggesting-enhancements)
- [Security issues](#security-issues)
- [Development environment](#development-environment)
- [Building and testing](#building-and-testing)
- [Coding standards](#coding-standards)
- [Commit messages](#commit-messages)
- [Pull request process](#pull-request-process)
- [Developer Certificate of Origin (DCO)](#developer-certificate-of-origin-dco)
- [License](#license)

---

## Code of Conduct

This project and everyone participating in it is governed by the
[vaultspec Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected
to uphold this code. Please report unacceptable behavior to the maintainers listed
in [MAINTAINERS.md](MAINTAINERS.md).

## Ways to contribute

There are many ways to contribute to vaultspec, and not all of them involve writing
code:

- **Report bugs** you encounter while using vaultspec
- **Suggest features** that would make vaultspec more useful to you or your team
- **Improve documentation** — fix typos, clarify explanations, add examples
- **Write tests** that increase coverage of existing functionality
- **Triage issues** by reproducing bugs or asking clarifying questions
- **Review pull requests** opened by other contributors
- **Implement features** from the roadmap or from accepted feature requests

If you are unsure where to start, look for issues labeled
[`good first issue`](https://github.com/timkrebs/vaultspec/labels/good%20first%20issue)
or [`help wanted`](https://github.com/timkrebs/vaultspec/labels/help%20wanted).

## Reporting bugs

Before filing a bug report:

1. **Search existing issues** to avoid duplicates.
2. **Reproduce the bug** with the latest released version of vaultspec.
3. **Collect diagnostic information**: vaultspec version (`vaultspec version`), Go
   version (`go version`), operating system, and a minimal policy/spec file that
   reproduces the issue.

Open a new issue using the [Bug Report](.github/ISSUE_TEMPLATE/bug_report.yml)
template and include all of the requested information. The more we can reproduce
locally, the faster we can fix it.

## Suggesting enhancements

Open a [Feature Request](.github/ISSUE_TEMPLATE/feature_request.yml) issue and
describe:

- **The problem** you are trying to solve (not just a proposed solution)
- **Who it affects** (yourself, your team, a class of users)
- **Alternatives** you have considered
- **Acceptance criteria** that would let us know the feature is "done"

For larger changes, please open an issue **before** writing code so we can agree on
the design. This avoids wasted work on both sides.

## Security issues

**Do not file public GitHub issues for security vulnerabilities.** Follow the
process described in [SECURITY.md](SECURITY.md).

---

## Development environment

### Prerequisites

- **Go** 1.24 or newer (the version pinned in [go.mod](go.mod) is the minimum)
- **Make** (GNU Make) — the project uses a Makefile for the canonical task runner
- **Git** 2.30+
- A POSIX-like shell (bash, zsh). Native Windows is not officially supported for
  development; use WSL2 instead.

Optional but recommended:

- [`golangci-lint`](https://golangci-lint.run/) — matches the linter set used in CI
- [`staticcheck`](https://staticcheck.dev/) — invoked by `make audit`
- [`govulncheck`](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) — invoked by
  `make audit`
- A locally running [HashiCorp Vault dev server](https://developer.hashicorp.com/vault/docs/get-started/developer-quickstart)
  if you are working on the online evaluator

### Cloning the repository

```bash
git clone https://github.com/timkrebs/vaultspec.git
cd vaultspec
go mod download
```

### Running for the first time

```bash
make build           # produces ./bin/vaultspec
./bin/vaultspec --help
```

## Building and testing

The [Makefile](Makefile) is the source of truth for development tasks. Always
prefer `make <target>` over running `go ...` directly so that local runs match CI.

| Target | Purpose |
|---|---|
| `make build` | Build the `vaultspec` binary into `./bin/` |
| `make run` | Build and run the binary |
| `make test` | Run the full test suite with race detector |
| `make test/cover` | Run tests and open an HTML coverage report |
| `make audit` | Run formatting check, `go vet`, `staticcheck`, and `govulncheck` |
| `make tidy` | Run `go mod tidy`, `go fix`, and `go fmt` |
| `make clean` | Remove build artifacts |
| `make version` | Print the version that will be baked into the binary |

**Before pushing**, you should be able to run `make audit` cleanly. CI will run the
same checks and reject pull requests that fail.

### Writing tests

vaultspec uses standard Go testing (`testing` package) with the race detector
enabled in CI. We do not currently mandate a specific assertion library — match the
style of the package you are editing.

Guidelines:

- **Every new feature ships with tests.** Bug fixes should include a regression
  test that fails before the fix and passes after.
- **Prefer table-driven tests** for parsers, evaluators, and analyzers.
- **Keep test fixtures in `testdata/`** following Go convention. Files inside
  `testdata/` are ignored by the Go toolchain.
- **Do not require network access** for unit tests. Online-mode tests should use a
  test server (`httptest`) or be tagged behind a build tag.

## Coding standards

- Code must be formatted with `gofmt` (run `make tidy`).
- Code must pass `go vet ./...` and `staticcheck ./...` cleanly.
- Public APIs must have Go doc comments. Follow the
  [Go Doc Comments style](https://go.dev/doc/comment).
- Errors are values: wrap with `fmt.Errorf("...: %w", err)` and never swallow
  errors silently.
- Avoid panics in library code. Panic only on programmer error (invariant
  violations), never on user input.
- Keep packages small and focused. Look at the layout in [README.md](README.md) for
  the intended package responsibilities.
- Do not introduce new dependencies without justifying them in the pull request
  description. Prefer the standard library and the existing dependency set.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/) as a soft
guideline. Example:

```
parser: support the `+` path glob introduced in Vault 1.16

The `+` glob matches a single path segment and is used in newer Vault
ACL policies. This change extends the matcher and adds table-driven
tests for both `*` and `+` semantics.

Fixes #42
```

- Use the imperative mood ("add feature", not "added feature").
- Keep the subject line under 72 characters.
- Reference issues with `Fixes #N` / `Refs #N` in the body.

## Pull request process

1. **Fork** the repository and create a feature branch off `main`.
   - Branch naming: `feat/short-description`, `fix/short-description`,
     `docs/short-description`.
2. **Make your changes** in small, logically scoped commits.
3. **Run `make audit` and `make test`** locally and fix anything they report.
4. **Update documentation** when behavior changes — at minimum [README.md](README.md)
   and [CHANGELOG.md](CHANGELOG.md) under the `Unreleased` section.
5. **Open a pull request** using the
   [PR template](.github/PULL_REQUEST_TEMPLATE.md). Fill in every section.
6. **Respond to review feedback** by pushing additional commits to the same branch.
   Do not force-push during review unless asked — it makes incremental review
   harder. The maintainer will squash on merge if appropriate.
7. **CI must be green** before a pull request can be merged.

A maintainer will review your PR. Reviews focus on correctness, tests,
documentation, and fit with the project's direction. Please be patient — this is a
side project.

## Developer Certificate of Origin (DCO)

By contributing to this project you certify that:

> Your contribution is your own original work, or you have the right to submit it
> under the project's license, and you agree that your contribution may be
> distributed under the terms of the project's license (MPL-2.0).

We do not require DCO sign-off in commit messages, but submitting a pull request
constitutes acceptance of the above.

## License

vaultspec is licensed under the [Mozilla Public License 2.0](LICENSE). All
contributions are accepted under the same license. If you are submitting code that
you did not write yourself, you must ensure that the original author has licensed
it under MPL-2.0 or a compatible license, and you must call this out in the pull
request description.
