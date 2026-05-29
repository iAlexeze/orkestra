# Support

Orkestra is an open-source project. This page explains how to get help, report
issues, and engage with the community.

---

## Before asking for help

Most questions are answered in:

- **[FAQ](./faqs/index.md)** — the most common questions about how Orkestra works
- **[Troubleshooting section](./getting-started/index.md/#troubleshooting)** in the Getting Started guide
- **Health endpoints** — `curl localhost:8080/katalog/website | jq` often shows the issue directly
- **Orkestra logs** — structured JSON logs with `crd`, `cr`, and `error` fields
- **Existing GitHub issues** — search before opening a new one

The `/katalog/{crd}` endpoint resolves the majority of operational questions without any external help. For a full live view, use the [Control Center](./controlcenter/controlcenter.md).

---

## Getting help

**GitHub Discussions** is the primary support channel for questions, troubleshooting,
and best practices. The maintainers and community respond here.

Use Discussions for:
- "How do I...?" questions
- Troubleshooting unexpected behaviour
- Understanding Orkestra concepts and design decisions
- Sharing what you built and getting feedback

**GitHub Issues** is for bugs and confirmed problems — not questions.

---

## Reporting bugs

When opening a bug report, include:

- Orkestra version: `ork version`
- Kubernetes version: `kubectl version`
- A minimal Katalog that reproduces the issue
- The error message and relevant log lines
- What you expected vs what happened

!!! note "Logs are the fastest path to a resolution"
    Orkestra logs include the CRD name, CR name, and full error details on every
    failure. Set `LOG_LEVEL=debug` to get per-reconcile detail. Attaching these
    logs to a bug report significantly reduces diagnosis time.

Issues with incomplete information take longer to resolve. A minimal reproduction
case is the single most helpful thing you can provide.

---

## Requesting features

Feature requests are welcome. Include:

- The problem you are trying to solve — not just the solution you want
- Why the current behaviour does not work for your use case
- Any alternatives you have considered
- Concrete examples of how you would use the feature

Context about the problem produces better features than requirements about the
solution.

---

## Security issues

!!! warning "Do not open public GitHub issues for security vulnerabilities"

See the [Security](./security.md) page for responsible disclosure instructions.

---

## Commercial support

If your organisation requires SLA-backed support, architecture guidance,
production onboarding, or custom integrations, contact the maintainers to discuss
commercial support options.

---

## Response expectations

Orkestra is maintained by a small team. Response times are best-effort and not
guaranteed. Community support through Discussions is the fastest path to a resolution
for most questions — other users often answer before maintainers do.

The most effective way to get a fast response is to make the problem easy to diagnose:
minimal reproduction, logs attached, version information included.
