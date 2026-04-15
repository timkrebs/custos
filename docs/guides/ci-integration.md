---
description: Recipes for running custos in GitHub Actions, GitLab CI, Jenkins, and more.
icon: arrows-spin
---

# CI integration

custos is designed to run in CI. Every release ships a static binary, a Docker image, and three reporter formats (terminal, JUnit XML, JSON) so it fits into any pipeline without special configuration.

## GitHub Actions

The most common setup. Install custos, run the tests, publish results.

```yaml
name: custos
on:
  pull_request:
    paths:
      - "policies/**"
      - "specs/**"

jobs:
  policy-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install custos
        run: |
          curl -sSfL \
            https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh \
            | bash

      - name: Run policy tests
        run: |
          mkdir -p results
          for spec in specs/*.spec.yaml; do
            name=$(basename "$spec" .spec.yaml)
            custos test -f "$spec" --format=junit > "results/$name.xml"
          done

      - name: Publish results
        if: always()
        uses: dorny/test-reporter@v1
        with:
          name: custos policy tests
          path: results/*.xml
          reporter: java-junit
```

{% hint style="info" %}
The `if: always()` on the reporter step ensures failed tests still publish results. Without it, a failed custos run skips the reporter and the PR shows no test report.
{% endhint %}

### Pinning to a specific version

For reproducible CI, pin the custos version explicitly:

```yaml
- name: Install custos
  run: |
    curl -sSfL \
      https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh \
      | bash -s -- v0.1.0
```

### Commenting on pull requests

Combine the JSON reporter with `jq` to post a summary comment:

```yaml
- name: Summarize
  if: always()
  run: |
    custos test -f specs/payment-svc.spec.yaml --format=json > out.json
    passed=$(jq '.summary.passed' out.json)
    failed=$(jq '.summary.failed' out.json)
    echo "Passed: $passed, Failed: $failed" >> $GITHUB_STEP_SUMMARY
```

## GitLab CI

```yaml
stages:
  - test

custos:
  stage: test
  image: ghcr.io/timkrebs/custos:latest
  script:
    - |
      for spec in specs/*.spec.yaml; do
        name=$(basename "$spec" .spec.yaml)
        custos test -f "$spec" --format=junit > "$name.xml"
      done
  artifacts:
    when: always
    reports:
      junit: "*.xml"
  rules:
    - changes:
        - policies/**/*
        - specs/**/*
```

GitLab renders JUnit reports natively on the merge request page — no extra plugin needed.

## Jenkins

```groovy
pipeline {
  agent any
  stages {
    stage('custos') {
      steps {
        sh '''
          curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash
          mkdir -p results
          for spec in specs/*.spec.yaml; do
            name=$(basename "$spec" .spec.yaml)
            custos test -f "$spec" --format=junit > "results/$name.xml"
          done
        '''
      }
      post {
        always {
          junit 'results/*.xml'
        }
      }
    }
  }
}
```

The `junit` post step ships with Jenkins core, so the test results appear on the build page automatically.

## Buildkite

```yaml
steps:
  - label: "custos policy tests"
    command: |
      curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash
      custos test -f specs/payment-svc.spec.yaml --format=junit > results.xml
    plugins:
      - junit-annotate#v2.4.1:
          artifacts: results.xml
```

## CircleCI

```yaml
version: 2.1
jobs:
  custos:
    docker:
      - image: ghcr.io/timkrebs/custos:latest
    steps:
      - checkout
      - run:
          name: Run custos tests
          command: |
            mkdir -p results
            for spec in specs/*.spec.yaml; do
              name=$(basename "$spec" .spec.yaml)
              custos test -f "$spec" --format=junit > "results/$name.xml"
            done
      - store_test_results:
          path: results
```

## Pre-commit hook

For teams that want policy tests to run locally before a commit:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: custos
        name: custos policy tests
        entry: custos test -f specs/payment-svc.spec.yaml
        language: system
        files: \.(hcl|spec\.yaml)$
        pass_filenames: false
```

## Failing the build on warnings

By default, custos exits non-zero only when an *assertion* fails. Use `--fail-on-warn` to also fail the build on static analyzer warnings:

```bash
custos test -f specs/payment-svc.spec.yaml --fail-on-warn
```

This is the right setting for teams that treat the security analyzer as a blocking gate.

## Common pitfalls

- **Missing `if: always()`** on the publish step means failed runs skip the reporter. Always add it.
- **Absolute paths in specs.** Keep policy paths in spec files relative. CI runners have unpredictable working directories.
- **Shell error handling.** If you loop over specs, remember to track failures: `for spec in specs/*.spec.yaml; do custos test -f "$spec" || fail=1; done; exit ${fail:-0}`.
- **Docker volume permissions.** When mounting a policy directory into the Docker image, the runner UID must be able to read it. `-v "$(pwd):/work:ro"` usually does the trick.
