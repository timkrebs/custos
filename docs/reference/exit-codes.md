---
description: Exit code reference for scripting and CI gating.
icon: right-from-bracket
---

# Exit codes

custos uses conventional Unix exit codes: `0` on success, non-zero on failure. The codes below are stable across releases and safe to rely on in CI scripts.

| Exit code | Meaning |
|---|---|
| `0` | All tests passed. Either no warnings were emitted or `--fail-on-warn` was not set. |
| `1` | At least one test failed, or `--fail-on-warn` was set and warnings were emitted, or the spec/policy files could not be loaded. |

## Decision table

| Tests | Warnings | `--fail-on-warn` | Exit |
|---|---|:---:|:---:|
| all pass | none | off | 0 |
| all pass | none | on | 0 |
| all pass | some | off | 0 |
| all pass | some | on | **1** |
| any fail | any | off | **1** |
| any fail | any | on | **1** |

## Scripting patterns

**Basic gate**

```bash
custos test -f specs/payment-svc.spec.yaml
if [ $? -ne 0 ]; then
  echo "custos failed"
  exit 1
fi
```

**Loop over many specs, collect failures**

```bash
fail=0
for spec in specs/*.spec.yaml; do
  custos test -f "$spec" || fail=1
done
exit $fail
```

Without the `|| fail=1`, the script exits on the first failure and skips remaining specs. The pattern above ensures every spec runs even if an earlier one fails, which is usually what you want in CI.

**Collect results into a summary**

```bash
mkdir -p results
fail=0
for spec in specs/*.spec.yaml; do
  name=$(basename "$spec" .spec.yaml)
  custos test -f "$spec" --format=junit > "results/$name.xml" || fail=1
done
exit $fail
```

## Why only two codes

Fine-grained exit codes (one for parse errors, another for assertion failures, a third for warnings) are tempting but in practice no CI system uses them. Every consumer just cares about zero or non-zero. Keeping two codes keeps the contract simple and reliable.

When you need to distinguish between failure modes, parse the JSON reporter output:

```bash
custos test -f spec.yaml --format=json | jq '.summary'
```
