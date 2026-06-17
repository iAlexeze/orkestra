# Foundations

These are the decisions that shaped everything else. Not rules imposed from outside — convictions earned while building, tested against real problems, and refined until the design held together.

Writing them down is the starting point. For the engineers who use Orkestra and want to understand why it behaves the way it does. For the contributors who want to push it further without breaking what makes it work. For anyone who arrives here wondering whether this is a system worth investing in.

They are not features. They are not configuration options. They are the load-bearing assumptions the whole design rests on. When something in Orkestra feels unexpected, the explanation is usually here.

The principles are not independent. Blindness makes do-one-thing-well possible — a layer can focus on its single job because it has no surface area with any other layer. Do-one-thing-well makes deliberate configuration meaningful — if each layer has one job, the contract that locks in at startup is small and knowable. Deliberate configuration is what makes production-as-default safe — you cannot accidentally promote a partially-updated runtime because the update requires an explicit restart. And secure-by-design is what makes the whole composable system trustworthy — each layer's security posture is derived from its declared intent, not from what the author remembered to configure.

They compound. Read them in order if you can.

---

**[Operators for Everyone](00-operators-for-everyone.md)** — The mission underneath all of it. Operators were hard to write not because the concept is hard, but because the wall between the idea and a working implementation was too high. Orkestra removes that wall — through the runtime, through the patterns in the registry, through learning materials that ship with the CLI and evolve with it. Common problems solved collectively and once, so that every team can focus on the problem they actually came to solve.

→ [Read: Operators for Everyone](00-operators-for-everyone.md)

---

**[Blindness by Design](01-blindness-by-design.md)** — Every layer in Orkestra is intentionally opaque to the layers around it. A Motif has no knowledge of the Katalog that imports it. A Katalog has no knowledge of the Komposer that composes it. The runtime has no knowledge of the registry that published the pattern. This blindness is not a limitation — it is the property that makes composition safe across teams, trust boundaries, and registries.

→ [Read: Blindness by Design](01-blindness-by-design.md)

---

**[Do One Thing Well](02-do-one-thing-well.md)** — Each Orkestra artifact has exactly one responsibility. A Motif is a resource template. A Katalog is an operator declaration. A Komposer is a composition. None does more than its role, and none knows more than its role requires. This is why you can add a Katalog without touching any other Katalog, upgrade a Motif without rewriting the operator that imports it, and compose patterns from registries you do not control.

→ [Read: Do One Thing Well](02-do-one-thing-well.md)

---

**[Configuration is Deliberate](03-no-autosync.md)** — Orkestra reads your Katalog at startup and shuts the door. No polling. No background updates. No automatic re-sync when a source changes. Reconciliation is continuous — the runtime watches your CRs and corrects drift forever. But configuration is a contract that locks in at start time, and changing it is a deliberate act: update the bundle, restart the runtime.

→ [Read: Configuration is Deliberate](03-no-autosync.md)

---

**[Secure by Design](04-secure-by-design.md)** — Security in Orkestra is structural, not a layer bolted on after the fact. Least-privilege RBAC is derived from what you declared — not requested. Admission gates enforce the same rules at two independent points. The production binary cannot run developer commands. These properties hold by default; you do not opt into them.

→ [Read: Secure by Design](04-secure-by-design.md)

---

**[Production Mode as Default](05-production-mode-as-default.md)** — Orkestra does not have a relaxed local mode and a strict production mode. Every run — local, CI, staging, production — starts the same runtime, applies the same validation, enforces the same rules. The only difference between environments is your configuration. This is what makes behavior consistent and promotions predictable.

→ [Read: Production Mode as Default](05-production-mode-as-default.md)
