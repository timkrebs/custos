---
description: How to contribute code, docs, and bug reports to custos.
icon: handshake
---

# Contributing

custos is an open source project and contributions are welcome. This section walks through the practical side: setting up a development environment, running the test suite, submitting a change, and the coding standards the project expects.

If you are reporting a bug or requesting a feature, please use the [GitHub issue tracker](https://github.com/timkrebs/custos/issues). Good bug reports include:

- A minimal reproduction (a spec file and any relevant policies)
- The exact custos version (`custos version`)
- What you expected to happen and what actually happened

## Where to start

<table data-view="cards">
  <thead>
    <tr>
      <th>Section</th>
      <th data-card-target data-type="content-ref">Target</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Development setup</strong> — clone, build, run the tests</td>
      <td><a href="development-setup.md">Development setup</a></td>
    </tr>
    <tr>
      <td><strong>Development workflow</strong> — branching, commit style, pull requests</td>
      <td><a href="workflow.md">Development workflow</a></td>
    </tr>
    <tr>
      <td><strong>Coding standards</strong> — what CI expects from every change</td>
      <td><a href="coding-standards.md">Coding standards</a></td>
    </tr>
  </tbody>
</table>

## Ways to contribute

- **Fix a bug.** Look for issues labeled `bug` and `good first issue`.
- **Implement a roadmap item.** See the [roadmap](../roadmap.md). Anything marked "planned" is fair game.
- **Add an analyzer.** New security checks are one of the highest-impact contributions. Each check is a pure function over the parsed policy and fits the pattern in `pkg/analyzer/`.
- **Improve the docs.** Found a typo or a confusing example? Open a pull request against the `docs/` directory. Documentation changes are just as valuable as code.
- **Triage issues.** Reproducing reports and adding clarifying questions helps maintainers move faster.

## Communication

Project discussion happens on GitHub. For questions about how to use custos, open a [discussion](https://github.com/timkrebs/custos/discussions). For proposed changes that need a design conversation, open an issue before writing code so the approach can be agreed before you invest time.

## Licensing

custos is released under [MPL-2.0](https://github.com/timkrebs/custos/blob/main/LICENSE). By submitting a pull request you agree that your contribution is licensed under the same terms.
