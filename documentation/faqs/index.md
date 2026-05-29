# Frequently Asked Questions

## [Concepts](./01-concepts.md)

Core concepts — what Orkestra is, how it works, and how it compares.

- What is Orkestra?
- Do I need to write Go code?
- How does Orkestra differ from Helm or Kustomize?
- What is a Katalog? What is a Komposer?
- What is the OrkestraRegistry?
- What is the super-operator model?
- Does Orkestra support multi-version CRDs?

---

## [Running](./02-running.md)

Setup, configuration, operations, and RBAC.

- Can Orkestra manage multiple CRDs?
- How do I start Orkestra?
- What does `ork validate` do?
- Does Orkestra require cert-manager?
- What environment variables does Orkestra read?
- What RBAC permissions does Orkestra need?
- How do I debug a CRD in production?
- Is Orkestra safe for production?

---

## [Patterns](./03-patterns.md)

Common usage patterns — built-in kinds, validation, mutation, conditions.

- Can Orkestra manage built-in Kubernetes resources?
- What is the difference between validation and mutation?
- Does `ENABLE_ADMISSION_WEBHOOK=true` block the API server if Orkestra is down?
- How do I use `when:` conditions?
- How does `dependsOn` ordering work?

---

## [Ecosystem](./04-ecosystem.md)

Comparisons and the path forward.

- How does Orkestra compare to kro?
- Can Orkestra manage third-party CRDs?
- What is the path to Kubernetes core?

---

## Why Katalog and Komposer are not CRDs

See [Why Not CRDs](./05-why-not-crds.md) for the full reasoning behind keeping Katalog
and Komposer as plain YAML files rather than Kubernetes CRDs.

---

## Where to go next

- **[Getting Started](../getting-started/index.md)** — your first operator in minutes
- **[Security](../security/index.md)** — admission control, RBAC, namespace protection
- **[Deploying](../deploying.md)** — running Orkestra in a real cluster
- **[Roadmap](../roadmap.md)** — what is shipped and what is coming
