# Security Policy

## Supported versions

| Version | Supported |
| ------- | --------- |
| latest  | Yes       |

## Reporting a vulnerability

**Do not file a public GitHub issue for security vulnerabilities.**

Instead, please report them responsibly by emailing:

**security@custos.dev**

Include the following in your report:

- Description of the vulnerability
- Steps to reproduce or proof-of-concept
- Impact assessment (what an attacker could achieve)
- Suggested fix, if any

You will receive an acknowledgement within **48 hours** and a detailed response
within **5 business days** with next steps.

## Disclosure policy

We follow coordinated disclosure. We will work with you to understand and
address the issue before any public disclosure. We ask that you give us a
reasonable amount of time to fix the vulnerability before disclosing it
publicly.

## Scope

The following are in scope:

- The `custos` CLI binary
- Policy evaluation engine (offline and online)
- Test spec parsing and validation

The following are **out of scope**:

- HashiCorp Vault itself
- Third-party dependencies (report those upstream)
