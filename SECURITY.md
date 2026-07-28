# Security Policy

## Supported Versions

taksuMCP is pre-1.0. Security fixes are made against the latest `main`
and the most recent tagged release only.

| Version | Supported |
| ------- | --------- |
| main    | ✅        |
| < 0.1.0 | ❌        |

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security
vulnerabilities.

Instead, report privately via [GitHub Security Advisories]
(https://github.com/Tharun-bot/taksuMCP/security/advisories/new)
or email [tharunkrishna1611@gmail.com].

Please include:
- A description of the vulnerability and its potential impact
- Steps to reproduce
- Affected version/commit

We aim to acknowledge reports within 5 business days and to disclose
publicly, with credit if desired, once a fix is available.

## Scope notes

taksuMCP issues task-scoped credentials for stateless auth
(`internal/auth`). Issues around token scoping, idempotency-key
collisions, or storage-layer race conditions are in scope and taken
seriously.