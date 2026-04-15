---
description: Branching, commits, and pull requests for custos contributors.
icon: code-branch
---

# Development workflow

This is the end-to-end lifecycle of a change: from a GitHub issue to a merged pull request. The workflow is deliberately lightweight, but a few conventions are non-negotiable because they keep history readable and CI fast.

## The big picture

```
issue open
   |
   v
feature branch --> commits --> push --> pull request --> CI + review --> merge --> issue closed
```

`main` is always deployable. All work happens on feature branches and reaches `main` only through reviewed, CI-green pull requests.

## Step by step

{% stepper %}
{% step %}
### Pick an issue

Browse the [issue tracker](https://github.com/timkrebs/custos/issues). Anything labeled `good first issue` is a safe starting point. Assign the issue to yourself so others know you are working on it.

```bash
gh issue view 42
gh issue edit 42 --add-assignee @me
```
{% endstep %}

{% step %}
### Create a feature branch

Branch names follow `<type>/<issue>-<short-description>`:

| Prefix | When |
|---|---|
| `feat/` | New feature |
| `fix/` | Bug fix |
| `docs/` | Documentation only |
| `refactor/` | Code restructuring without behaviour change |
| `ci/` | CI or release pipeline |
| `test/` | Test-only changes |

```bash
git checkout main
git pull origin main
git checkout -b feat/42-add-scan-command
```
{% endstep %}

{% step %}
### Make focused commits

Each commit should do one thing and leave the build green. Follow Conventional Commits:

```
feat(scan): add standalone security scan command

Introduces custos scan, which runs the analyzer against
a set of HCL files without requiring a spec file.

Closes #42
```

Good commit subjects:

- `feat(analyzer): add secret_destroy check`
- `fix(parser): handle empty allowed_parameters map`
- `docs: clarify trailing-slash rule for list capability`

Avoid vague messages like `wip`, `fix stuff`, or `updates`.
{% endstep %}

{% step %}
### Run the audit locally

```bash
make audit
```

This runs everything CI will run: gofmt, go vet, staticcheck, govulncheck, and the full race-enabled test suite. If it fails locally, fix it before pushing.
{% endstep %}

{% step %}
### Push and open a pull request

```bash
git push -u origin feat/42-add-scan-command
gh pr create --fill
```

The pull request template asks you to describe what changed, why, and how it was tested. Fill it in — future maintainers will thank you.
{% endstep %}

{% step %}
### Respond to CI and review

CI runs the audit on the pull request. Fix any failures with follow-up commits on the same branch. Reviewers may ask for changes; address them in place rather than force-pushing, so the conversation stays coherent.

```bash
# Address review feedback
git add -p
git commit -m "refactor(scan): move severity parsing into helper"
git push
```
{% endstep %}

{% step %}
### Merge

Once CI is green and the pull request is approved, a maintainer merges with "Squash and merge" so the commit history on `main` stays linear and readable. The squashed commit message uses the pull request title, so make sure the title is a good commit subject.
{% endstep %}
{% endstepper %}

## CI pipeline

Every push runs two parallel jobs:

- **Test** — `go test -race ./...`
- **Audit** — `make audit` (fmt, vet, staticcheck, govulncheck)

When both pass, a **Build** job compiles binaries for darwin, linux, and windows across amd64 and arm64 to make sure cross-compilation still works. Release artifacts are published automatically on tagged commits via GoReleaser.

## Release process

Releases are maintainer-driven and follow semver. The mechanics:

```bash
# From main, after the desired commits have landed
git tag -a v0.2.0 -m "v0.2.0: JUnit, JSON, and analyzer"
git push origin v0.2.0
```

Pushing the tag triggers GoReleaser, which cross-compiles, generates a changelog, publishes a GitHub release, and pushes a Homebrew tap update.

## Commit hygiene checklist

Before pushing, check:

- Commits are focused and named with Conventional Commits style.
- `make audit` passes locally.
- New behaviour has test coverage.
- Public API changes are documented in the appropriate `docs/` page.
- The pull request title reads like a good commit subject.
