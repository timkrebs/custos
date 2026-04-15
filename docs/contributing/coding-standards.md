---
description: Style and design conventions CI enforces on every pull request.
icon: ruler-triangle
---

# Coding standards

custos is written in idiomatic Go. The project trusts the standard toolchain: `gofmt`, `go vet`, `staticcheck`, and `govulncheck`. Everything in this document is either enforced by one of those tools or by review.

## The golden rule

```bash
make audit
```

If that passes locally, your change is almost certainly acceptable. The rest of this page is context for *why* the rules exist.

## Formatting

- **`gofmt -s` is non-negotiable.** CI fails on unformatted code. Your editor should run `gofmt` on save.
- **Imports** are grouped: standard library, third-party, local. Use `goimports` to keep them sorted.
- **Line length** is not enforced. Prefer clear names and short functions over artificial wrapping.

## Naming

- Exported identifiers have doc comments starting with the identifier name: `// Evaluate returns the composed result ...`
- Package names are short, lowercase, and singular: `parser`, not `parsers`.
- Test function names are `TestXxx_<situation>` or `TestXxx/<subtest>` with `t.Run`.

## Errors

- Return errors; do not panic. A panic in any reachable code path is a bug.
- Wrap errors with `fmt.Errorf("doing X: %w", err)` when the caller benefits from context. Do not wrap when the error is already descriptive.
- Do not create custom error types unless callers need to match on them.

## Testing

- Every new package, exported function, or code path should have tests. The test suite is the single most important thing custos ships.
- Prefer table-driven tests for anything with more than two cases.
- Use `testdata/` fixtures for realistic policy and spec examples. Keep fixtures minimal and focused.
- `go test -race ./...` must pass. CI enforces this.

## Comments

- Default to writing no comments. Well-named functions and types explain themselves.
- Add a comment when the *why* is non-obvious: a tricky invariant, a workaround for a known bug, behaviour that would surprise a reader.
- Do not write comments that describe *what* the code does. If a reader needs that, the code needs better names.
- Exported identifiers need doc comments per `go vet -s`. Keep them short.

## Public API stability

The CLI surface and the JSON output schema are stable contracts. Breaking changes to either need a major version bump and a migration note in the changelog. In-package Go APIs under `pkg/` are internal for now and may change without notice.

## Package boundaries

custos keeps package responsibilities narrow:

- `pkg/parser` is the only code that reads HCL.
- `pkg/evaluator` is the only code that makes allow/deny decisions.
- `pkg/reporter` is the only code that writes output.
- `cmd/` is CLI glue with no business logic.

Resist the urge to put a helper in `pkg/util`. If it does not fit any existing package, it probably belongs in a new, purpose-named package.

## Dependencies

- **Add a dependency** only when the functionality is non-trivial and well-maintained.
- Prefer the standard library where it is good enough.
- Any new dependency needs a one-line justification in the pull request description.

## Logging

There is no global logger. Reporters write to their configured `io.Writer`. Everything else that wants to emit diagnostics writes to `os.Stderr`. Do not introduce `log` package calls or structured logging libraries without discussion.

## What CI enforces

| Check | Tool | Blocking |
|---|---|---|
| Format | `gofmt` | yes |
| Vet | `go vet` | yes |
| Static analysis | `staticcheck` | yes |
| Vulnerabilities | `govulncheck` | yes |
| Tests (with race) | `go test -race` | yes |
| Cross-compile | `goreleaser build --snapshot` | yes, on tag |

If any of the above fails, CI fails. Fix the underlying issue; do not disable the check.

## What humans enforce in review

- Clear naming
- Focused commits and pull requests
- Tests for new behaviour
- Documentation updates where behaviour changes
- No dead code, no commented-out code
- No `TODO` comments without an issue number

## When in doubt

When you are not sure whether a pattern is idiomatic, look at the rest of `pkg/` for prior art. If nothing similar exists, open a draft pull request and ask in the description. Maintainers would rather discuss an approach early than review a finished change that needs to be reworked.
