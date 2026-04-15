---
description: The default human-readable colored reporter.
icon: display
---

# Terminal

The terminal reporter is the default output format. It prints colored, human-readable results intended for running in a developer shell. It is what `custos test` uses when no `--format` flag is passed.

## Example output

```
  payment-service-policies

    OK   can read its own secrets              (secret/data/payment-svc/db-creds)
    OK   can list its own secret keys          (secret/data/payment-svc/)
    OK   cannot write to its own secrets       (secret/data/payment-svc/db-creds)
    OK   cannot read billing secrets           (secret/data/billing-svc/api-key)
    OK   can issue certificates                (pki_int/issue/payment-svc)
    FAIL no access to sys backend              (sys/seal)
      expected: deny, got: allow via rule "sys/*"
        in policies/admin-legacy.hcl

  5 passed, 1 failed, 0 skipped

  WARNING  wildcard path "secret/*" grants 5 capabilities
    at policies/admin-legacy.hcl:2

  1 warning
```

## Layout

1. **Suite header.** The `suite:` field from the spec file.
2. **Test list.** One line per test, indented under the suite. Each line shows the status (`OK` or `FAIL`), the test name, and the path in parentheses.
3. **Failure details.** For every failing test, a second indented line explains why: expected vs actual, the matched rule, and the policy file.
4. **Summary.** Counts of passed, failed, and skipped tests.
5. **Findings.** Any analyzer warnings or errors from the current run, each with file and line.
6. **Findings summary.** Total count of warnings and errors.

## Verbose mode

Pass `-v` or `--verbose` to get an evaluation trace for every test, not just failures:

```
    OK   can read its own secrets              (secret/data/payment-svc/db-creds)
      matched: "secret/data/payment-svc/*"
        in policies/payment-svc.hcl:1
      granted: [read, list]
      composed: allow
```

In composed suites with multiple policies, verbose mode prints a contribution trace:

```
    FAIL billing denied despite readonly allowing read  (secret/data/billing-svc/api-key)
      expected: deny, got: allow
      contributions:
        readonly      secret/*                 GRANT [read, list]
        payment-svc   secret/data/billing-svc/* no match
```

This is the single most useful mode when debugging why a composed test does not behave the way you expect.

## Colors

The reporter uses ANSI colors:

- **Green** for `OK`
- **Red** for `FAIL`, errors
- **Yellow** for warnings
- **Dim** for metadata (file paths, line numbers, matched rule details)

Colors are automatically disabled when the output is not a terminal. You can also turn them off explicitly by setting the standard `NO_COLOR` environment variable:

```bash
NO_COLOR=1 custos test -f spec.yaml
```

## When to use it

- Interactive development — the default, no configuration needed.
- Local debugging with `-v` to see the full trace.

For CI, prefer [JUnit XML](junit.md) or [JSON](json.md), which are designed to be consumed by tooling.
