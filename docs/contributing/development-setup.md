---
description: Clone, build, test, and iterate on custos locally.
icon: code
---

# Development setup

custos is a standard Go project with no unusual build requirements. Anything that can build Go 1.25 binaries can work on custos.

## Prerequisites

| Tool | Minimum version | Why |
|---|---|---|
| Go | 1.25 | Language and toolchain |
| make | any | Runs the project's build and test targets |
| git | any | Source control |

Optional but recommended:

- `golangci-lint` — umbrella linter used in CI
- `staticcheck` — extra analysis that `make audit` invokes
- `govulncheck` — Go vulnerability database scan

All three are installed automatically by `make audit` the first time you run it.

## Clone and build

```bash
git clone https://github.com/timkrebs/custos.git
cd custos
make build
./bin/custos version
```

`make build` compiles a binary into `./bin/custos` with the current commit hash and version string baked in via `-ldflags`.

## Run the test suite

```bash
# Unit tests only
make test

# Full audit: fmt, vet, lint, tests with race detector, vuln check
make audit
```

`make audit` is the single command CI runs on every pull request. If it passes locally, your change will almost certainly pass CI.

## Useful Make targets

| Target | What it does |
|---|---|
| `make build` | Compile `./bin/custos` |
| `make test` | Run unit tests (`go test ./...`) |
| `make audit` | Format check, vet, lint, vuln check, race tests |
| `make tidy` | `go mod tidy` and `gofmt -w` |
| `make clean` | Remove `./bin` and coverage artifacts |
| `make install` | `go install` to `$GOBIN` |

## Running custos against its own testdata

The repository ships example policies and specs in `testdata/` that exercise every feature. They are the fastest way to sanity-check a change:

```bash
./bin/custos test -f testdata/specs/payment-svc.spec.yaml
./bin/custos test -f testdata/specs/composed.spec.yaml -v
./bin/custos test -f testdata/specs/admin.spec.yaml
```

If you are adding a feature, adding a `testdata/` sample that exercises it is usually the simplest way to get regression coverage.

## Project layout

```
custos/
  cmd/              # CLI commands (root, test, version)
  pkg/
    analyzer/       # Static security checks
    evaluator/      # Offline evaluation + composition
    parser/         # HCL policy parsing
    reporter/       # terminal, junit, json output
    spec/           # YAML spec loader
    vaultpolicy/    # Canonical capability vocabulary
  testdata/         # Example policies and specs
  version/          # Build-time version info
  docs/             # GitBook documentation (this site)
  .github/workflows # CI pipeline definitions
  main.go           # Entry point
  Makefile          # Developer targets
```

Most contributions land in `pkg/`. The CLI layer in `cmd/` is intentionally thin and should not grow new business logic.

## Debugging

The fastest debugging loop for a failing test is:

```bash
go test ./pkg/evaluator/... -run TestSpecificThing -v
```

For the CLI, use `-v` to see evaluation traces:

```bash
./bin/custos test -f testdata/specs/composed.spec.yaml -v
```

There is no special debug logging to enable. If you need it, add `t.Log` calls to your test or drop a `fmt.Fprintf(os.Stderr, ...)` into the code you are debugging — just remember to remove them before committing.

## Pre-push hook

A common safeguard is a pre-push git hook that runs `make audit`. Save this to `.git/hooks/pre-push` and `chmod +x` it:

```bash
#!/usr/bin/env bash
set -e
make audit
```

## Next steps

- [Development workflow](workflow.md) covers branching, commits, and pull requests.
- [Coding standards](coding-standards.md) is the quick reference for style rules CI enforces.
