# Namespace Provisioner Guide

A namespace provisioner turns a single CR into a fully isolated tenant environment. Apply one resource, get a namespace, resource quotas, network policies, and RBAC — all correctly named, labelled, and wired to the right team.

This guide walks through four variants of the same provisioner:

| Example | Approach |
|---|---|
| [01-explicit](02-explicit.md) | Inline limits in the CR |
| [02-profiles](03-profiles.md) | Built-in profile name (`tier: medium`) |
| [03-user-defined-profiles](04-user-defined-profiles.md) | All six profile classes declared in the Katalog's `profiles:` block |
| [04-motif-profiles](05-motif-profiles.md) | Same profiles, but definitions live in a shared motif — the Katalog holds none |

Examples 01 and 02 produce the same eight resources; the difference is how much the CR author needs to know about Kubernetes quota mechanics. Examples 03 and 04 extend this to all six profile classes and show the two ways to own those definitions: inline (03) or in a motif (04).

The example ships three shared motifs — `tenant-isolation`, `tenant-rbac`, and `tenant-policies` — each sub-example selects the motifs it needs.

---

## Get the example

```bash
ork init --pack use-cases/namespace-provisioner
```

---

## Contents

| Page | What it covers |
|---|---|
| [CRD Design](01-crd.md) | Why the CRD must be cluster-scoped; field choices |
| [Explicit limits](02-explicit.md) | `01-explicit`: validate, simulate, run |
| [Profile-based limits](03-profiles.md) | `02-profiles`: replace hard limits with a tier name |
| [User-defined profiles](04-user-defined-profiles.md) | `03-user-defined-profiles`: all six classes in one `profiles:` block |
| [Profiles via motif](05-motif-profiles.md) | `04-motif-profiles`: delegate the profile registry to a shared motif |

---

→ Start with [CRD Design](01-crd.md)
