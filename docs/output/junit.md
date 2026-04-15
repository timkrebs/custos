---
description: Standard Ant JUnit XML for CI test reporters.
icon: file-xml
---

# JUnit XML

The JUnit reporter emits standard Ant JUnit XML that every major CI system knows how to consume. Use it when you want your policy tests to show up in the same UI as your regular test suites.

## Enable it

```bash
custos test -f specs/payment-svc.spec.yaml --format=junit > results.xml
```

## Schema

custos follows the de-facto Ant JUnit schema: `<testsuites>` containing one `<testsuite>` per spec file, each containing one `<testcase>` per test assertion. Failures are represented with nested `<failure>` elements.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="custos" tests="9" failures="1" errors="0" time="0.001234">
  <testsuite name="payment-service-policies"
             tests="9"
             failures="1"
             errors="0"
             skipped="0"
             time="0.001234"
             timestamp="2026-04-15T10:00:00Z">
    <testcase name="can read its own secrets"
              classname="payment-service-policies"
              time="0.000021" />
    <testcase name="no access to sys backend"
              classname="payment-service-policies"
              time="0.000018">
      <failure message="expected deny, got allow at path sys/seal"
               type="AssertionError">
        Expected: deny
        Got:      allow
        Path:     sys/seal
        Capabilities: [sudo]
        Matched rule: "sys/*" in policy "admin-legacy"
        Explanation: explicitly allowed by rule "sys/*" in admin-legacy.hcl
      </failure>
    </testcase>
  </testsuite>
</testsuites>
```

## Field semantics

| Element / attribute | Meaning |
|---|---|
| `<testsuites name="custos">` | Top-level container. Always named `custos`. |
| `tests` | Total test count across all suites. |
| `failures` | Number of tests that did not match their expected result. |
| `errors` | Number of tests that could not be evaluated (invalid spec, unreachable policy file, etc.). |
| `time` | Total duration in seconds, float with microsecond precision. |
| `<testsuite name="...">` | Named after the `suite:` field in the spec. |
| `<testcase name="...">` | Named after the `name:` field of the test assertion. |
| `<testcase classname="...">` | Set to the suite name for compatibility with Java-oriented reporters. |
| `<failure message="...">` | Short one-line summary. |
| `<failure>` chardata | Detailed multi-line body with expected, actual, path, capabilities, and matched rule. |
| `timestamp` | ISO 8601 UTC timestamp of suite start. |

All user-supplied strings (test names, paths, explanations) are XML-escaped so special characters in spec files do not break the output.

## CI integration examples

### GitHub Actions with dorny/test-reporter

```yaml
- name: Run custos tests
  run: custos test -f specs/payment-svc.spec.yaml --format=junit > results.xml

- name: Publish test results
  if: always()
  uses: dorny/test-reporter@v1
  with:
    name: custos policy tests
    path: results.xml
    reporter: java-junit
```

### GitLab CI

```yaml
policy-tests:
  script:
    - custos test -f specs/payment-svc.spec.yaml --format=junit > results.xml
  artifacts:
    when: always
    reports:
      junit: results.xml
```

GitLab renders the report natively on the merge request page.

### Jenkins

```groovy
post {
  always {
    junit 'results.xml'
  }
}
```

Jenkins core ships with JUnit support; no plugin needed.

## Multiple spec files

For projects with many specs, run each one separately and publish the results together. The reporter wraps its output in a `<testsuites>` element, so tools that expect multiple files handle it naturally:

```bash
mkdir -p results
for spec in specs/*.spec.yaml; do
  name=$(basename "$spec" .spec.yaml)
  custos test -f "$spec" --format=junit > "results/$name.xml"
done
```

Most CI reporters accept a glob pattern (`results/*.xml`) to aggregate them.

## Exit code and `<failure>`

A failing assertion produces both a `<failure>` element *and* a non-zero exit code. They carry the same information; CI systems that want to fail the build on any failure can rely on either.

## What JUnit does not carry

JUnit XML is intentionally minimal. It does not include:

- Full composition traces (use [JSON](json.md) for that)
- Analyzer findings (they print to the terminal alongside the test results, but not to the XML)
- Policy file contents

If you need more detail in your CI, run custos twice: once with `--format=junit` for the report, and once with `--format=json` saved as an artifact.
