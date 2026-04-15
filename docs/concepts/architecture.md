---
description: How custos is put together, from CLI entrypoint to reporter output.
icon: diagram-project
---

# Architecture

custos is a Go binary organised into a small set of packages, each with a single responsibility. The data flow is linear: a test spec goes in, a suite result comes out, and a reporter renders it. Every stage is deterministic and has no network dependencies.

## Pipeline

```
          spec file (YAML)
                |
                v
           pkg/spec  <--- strict YAML decode + path resolution
                |
       +--------+--------+
       |                 |
       v                 v
  pkg/parser         (test cases)
       |                 |
       v                 |
  policy AST             |
       +-----+-----+     |
             |     |     |
             v     v     v
        pkg/analyzer  pkg/evaluator
             |           |
             v           v
         findings    test results
             \           /
              \         /
               v       v
            pkg/reporter
                 |
                 v
         terminal / junit / json
```

## Packages

{% tabs %}
{% tab title="cmd/" %}
The CLI layer. Command definitions, flag parsing, and the glue that wires the other packages together.

- `cmd/cli.go` — root command
- `cmd/cli_start.go` — `custos test` implementation
- `cmd/version_cmd.go` — `custos version` implementation

The CLI layer intentionally contains no business logic. Its only job is to translate flags into function calls and translate results back into reporter output.
{% endtab %}

{% tab title="pkg/spec" %}
Loads and validates test spec YAML files. Implements strict decoding (unknown fields are errors) and resolves policy file paths relative to the spec's own directory.

Defines the `Spec`, `TestCase`, `PolicyRef`, and `AnalyzeCheck` types. This is the contract between humans and custos.
{% endtab %}

{% tab title="pkg/parser" %}
Parses Vault HCL policies using `hcl/v2`. Extracts each `path` block into a `PathRule` struct with the path pattern, capabilities, parameter whitelists, wrapping TTLs, and the source line number.

The parser is strict about what Vault itself accepts and produces source-annotated diagnostics on error.
{% endtab %}

{% tab title="pkg/evaluator" %}
The heart of custos. Two modules:

- **offline.go** — matches a request path against a policy's rules using Vault's precedence rules (exact > single-segment wildcard > trailing wildcard), then checks capabilities.
- **composer.go** — composes multiple policies by unioning grants and applying deny as a hard override.

Both return structured results (`TestResult`, `Composed`) that carry enough provenance to explain any failure.
{% endtab %}

{% tab title="pkg/analyzer" %}
Runs static security checks on parsed policies. Each check is a pure function over the policy AST, independent of test pass/fail state. Five checks ship today: wildcard paths, sudo capability, root token create, policy escalation, and secret destroy.

Findings include file, line, check ID, severity, and the rule capabilities so the reporter can render them next to the offending source.
{% endtab %}

{% tab title="pkg/reporter" %}
Three reporters implementing a common `Reporter` interface:

- **terminal.go** — colored, human-readable output
- **junit.go** — standard Ant JUnit XML for CI
- **json.go** — stable-schema JSON for programmatic use

Reporters consume a `SuiteResult` and write to an `io.Writer`. They have no knowledge of the CLI or the evaluator internals.
{% endtab %}

{% tab title="pkg/vaultpolicy" %}
Canonical Vault capability vocabulary. Owned separately from the parser so that the parser, the spec validator, and the analyzer all draw from one source of truth.
{% endtab %}
{% endtabs %}

## Data flow in one run

{% stepper %}
{% step %}
### Load the spec

`pkg/spec.LoadFile()` reads the YAML, validates it, and resolves every `policies[].path` relative to the spec's directory.
{% endstep %}

{% step %}
### Parse the policies

`pkg/parser.ParseFile()` is called for each policy. The result is a `Policy` struct containing a slice of `PathRule`, each with its source line.
{% endstep %}

{% step %}
### Evaluate the test cases

For each test in the spec, `pkg/evaluator.EvaluateSuite()` calls `Compose()` to merge the contributing policies, checks the expected outcome against the composed result, and builds a `TestResult`.
{% endstep %}

{% step %}
### Run the analyzer

`pkg/analyzer.Analyze()` walks the parsed policies and emits findings. The `analyze` section of the spec configures which checks run, which paths are exempt, and whether warnings are promoted to errors.
{% endstep %}

{% step %}
### Render the report

The chosen reporter (`terminal`, `junit`, or `json`) consumes the `SuiteResult` and writes to stdout. Diagnostics go to stderr so reporter output can be piped cleanly.
{% endstep %}

{% step %}
### Decide the exit code

Non-zero if any test failed, or if warnings were emitted and `--fail-on-warn` was set. Otherwise zero.
{% endstep %}
{% endstepper %}

## Design principles

1. **Offline is non-negotiable.** Every stage is local and deterministic. No network, no Vault, no global state.
2. **One source of truth per concern.** The parser is the only thing that reads HCL. The evaluator is the only thing that decides allow/deny. The reporters are the only thing that formats output.
3. **Match Vault semantics exactly.** If custos says a path is denied, production Vault denies it too. This is a trust boundary.
4. **Explain every failure.** Every test result carries enough provenance to answer "which rule in which policy caused this?" without re-reading the policies.
5. **Strict by default.** Strict YAML decoding, strict HCL parsing, strict spec validation. Typos should fail fast, not silently change behaviour.
