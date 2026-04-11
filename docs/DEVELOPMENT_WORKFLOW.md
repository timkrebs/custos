# Development Workflow

This guide walks through the full lifecycle of working on a GitHub issue in
custos — from picking up an issue to getting your code merged into `main`.

We use **issue #1** ("Fix 'infragraph' references throughout codebase") as a
concrete example throughout.

---

## Overview

```
  Issue         Branch              PR              main
  ─────         ──────              ──              ────

  #1 open ──► fix/1-infragraph ──► PR #N ──► CI ──► merge ──► close #1
                │                     ▲
                ├─ commit             │
                ├─ commit             │
                └─ push ──────────────┘
```

**Key rule:** `main` is always deployable. All work happens on feature branches
and reaches `main` only through reviewed, CI-green pull requests.

---

## Step-by-step

### 1. Pick an issue and assign yourself

```bash
# View the issue
gh issue view 1

# Assign it to yourself
gh issue edit 1 --add-assignee @me
```

### 2. Create a feature branch

Branch names follow the pattern: `<type>/<issue>-<short-description>`

| Type | When to use |
|------|-------------|
| `fix/` | Bug fixes |
| `feat/` | New features |
| `docs/` | Documentation only |
| `refactor/` | Code restructuring (no behavior change) |
| `ci/` | CI/CD changes |

For issue #1 (a bug fix):

```bash
# Make sure you're on main and up to date
git checkout main
git pull origin main

# Create and switch to the feature branch
git checkout -b fix/1-infragraph-references
```

### 3. Make your changes

```bash
# Edit the files
# cmd/cli.go         — change "infragraph" to "custos"
# main_test.go       — change "infragraph" to "custos"
# version/version.go — change "InfraGraph" to "custos" (already done)
```

### 4. Verify locally before committing

Run the same checks that CI will run:

```bash
# Format, vet, staticcheck, govulncheck
make audit

# Run the full test suite with race detector
make test

# Build the binary and verify it runs
make build
./bin/custos version
```

**Do not push until `make audit` and `make test` both pass.**

### 5. Commit your changes

Use [Conventional Commits](https://www.conventionalcommits.org/) style:

```bash
# Stage specific files (never use `git add .` blindly)
git add cmd/cli.go main_test.go version/version.go

# Commit with a descriptive message referencing the issue
git commit -m "fix(cli): replace infragraph references with custos

The CLI binary name, test helpers, and version string all still
referenced 'infragraph' from the original project scaffold. This
aligns all user-facing strings with the actual project name.

Fixes #1"
```

**Commit message anatomy:**

```
<type>(<scope>): <subject>       ← 72 chars max
                                  ← blank line
<body>                            ← explain WHY, not WHAT
                                  ← blank line
Fixes #1                          ← auto-closes issue on merge
```

### 6. Push the branch to GitHub

```bash
git push -u origin fix/1-infragraph-references
```

### 7. Create a pull request

```bash
gh pr create \
  --title "fix(cli): replace infragraph references with custos" \
  --body "## Summary
- Rename all 'infragraph' references to 'custos' in CLI init, test binary name, and version string

## Related issues
Fixes #1

## Test plan
- [x] \`make audit\` passes
- [x] \`make test\` passes
- [x] \`./bin/custos version\` shows 'custos' not 'infragraph'" \
  --milestone "Sprint 1 - Foundation" \
  --label "bug,sprint:1,size:S,area:cli"
```

### 8. CI runs automatically

When you open a PR targeting `main`, the CI workflow
(`.github/workflows/ci.yml`) triggers automatically:

```
CI Pipeline
───────────────────────────────────────────────
  ┌──────┐     ┌───────┐
  │ Test │     │ Audit │      ← run in parallel
  └──┬───┘     └───┬───┘
     │             │
     └──────┬──────┘
            ▼
       ┌─────────┐
       │  Build   │           ← only if Test + Audit pass
       └─────────┘
```

| Job | What it checks |
|-----|---------------|
| **Test** | `make test` — unit tests with race detector |
| **Audit** | `gofmt`, `go vet`, `staticcheck`, `govulncheck`, `go.mod` tidy |
| **Build** | `make build` + `./bin/custos version` |

### 9. Code review

- A CODEOWNER (currently `@timkrebs`) is automatically requested as reviewer
- Address review feedback with additional commits on the same branch:

```bash
# Make requested changes, then:
git add <files>
git commit -m "fix: address review feedback — rename temp dir pattern"
git push
```

- **Do not force-push during review** — it makes incremental review harder
- The maintainer will squash-merge if appropriate

### 10. Merge to main

Once CI is green and the review is approved:

```bash
# Merge via GitHub (preferred — uses the merge button)
gh pr merge --squash --delete-branch
```

Or via the GitHub web UI: click **"Squash and merge"**.

This:
- Squash-merges your commits into one clean commit on `main`
- Deletes the feature branch
- Auto-closes issue #1 (because the PR body says `Fixes #1`)

### 11. Clean up locally

```bash
git checkout main
git pull origin main
git branch -d fix/1-infragraph-references
```

---

## Releasing to production

Releases are cut from `main` using Git tags:

```bash
# Tag a release (triggers the release workflow)
git tag -a v0.1.0 -m "v0.1.0 — MVP with offline evaluation"
git push origin v0.1.0
```

This triggers `.github/workflows/release.yml`, which:
1. Runs GoReleaser
2. Cross-compiles for linux/darwin/windows (amd64 + arm64)
3. Creates a GitHub Release with binaries and checksums

---

## Quick reference

| Action | Command |
|--------|---------|
| View issue | `gh issue view 1` |
| Create branch | `git checkout -b fix/1-short-name` |
| Run tests | `make test` |
| Run full checks | `make audit` |
| Build binary | `make build` |
| Push branch | `git push -u origin fix/1-short-name` |
| Create PR | `gh pr create --title "..." --body "..."` |
| Check CI status | `gh pr checks` |
| Merge PR | `gh pr merge --squash --delete-branch` |
| Tag release | `git tag -a v0.1.0 -m "..." && git push origin v0.1.0` |

## Branch naming examples

| Issue | Type | Branch name |
|-------|------|-------------|
| #1 Bug: infragraph naming | fix | `fix/1-infragraph-references` |
| #2 Feature: HCL parser | feat | `feat/2-hcl-policy-parser` |
| #7 Test data | feat | `feat/7-example-test-data` |
| #9 JUnit reporter | feat | `feat/9-junit-reporter` |
| Update CONTRIBUTING.md | docs | `docs/update-contributing` |
