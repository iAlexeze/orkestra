---
title: "Why Operators Are Over-Permissioned — And How We Fixed It"
date: 2026-05-16
weight: 1
---

Most Kubernetes operators are massively over-permissioned. Not slightly, not accidentally — structurally. Inspect the RBAC of a typical production operator and you will find something like this:

```yaml
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
```

Cluster-wide. Full access. And it is not because the engineers who wrote them were careless. It is because the system makes it easier to be unsafe than correct.

---

The operator model evolved around one assumption: an operator is a program. You write Go, scaffold controllers, wire informers. Somewhere in that process you add RBAC. Then you add more. Then you hit a permissions error in production at 2am and do the fastest thing possible — widen the scope until it works. And it stays that way. Not because it is right, but because RBAC is painful to maintain, operators evolve, and nobody wants to debug fine-grained permissions under pressure. So teams converge on a pattern: over-permission once, never touch it again.

The real problem is not RBAC. RBAC is doing exactly what it was designed to do. The problem is that RBAC is disconnected from intent. Permissions are written by hand, but operators are dynamic systems — they reconcile different resources as they evolve, they compose with other operators, their scope changes. The RBAC drifts. And drift in security always goes in one direction: more access, not less.

This compounds as operator count grows. A typical cluster might run ten community operators and five internal ones, each with its own RBAC, its own assumptions, its own blast radius. No single person has a complete picture. Security becomes "we trust these operators" rather than "we understand what they can do."

---

The industry has treated RBAC as something you write. But an operator already tells you everything you need to know about what it needs: what CRDs it watches, what resources it creates, what resources it updates. That information is already present in the operator's declaration. The permission model is implicit in the behavior model. We just have not been using it.

Orkestra makes it explicit. You write a Katalog — a declarative definition of your operator — and Orkestra derives the permissions from what the Katalog declares:

```bash
ork generate bundle
```

The output is a ClusterRole containing exactly the resources declared in the Katalog's `onCreate`, `onReconcile`, and `onDelete` blocks. If your operator creates Deployments and Services, it gets permissions for Deployments and Services. Nothing more. The permissions are not a parallel document you maintain — they are a derivative of the declaration you already wrote.

This changes the security posture in a few ways that matter. First, security becomes the default. You do not need to think about RBAC to be secure — you define what your operator does and the permissions follow. Second, there is no silent expansion. When the Katalog changes, the generated bundle changes deterministically. You see exactly what is being added or removed before it reaches the cluster, diffable in any GitOps workflow. Third, composition stays safe. Orkestra runs multiple CRDs in one runtime, and without precise per-CRD permissions that would be dangerous — one CRD could inadvertently expand the access available to another. Derived permissions prevent this because each Katalog defines a bounded capability.

---

This is not only about RBAC. It is about changing the unit of trust. The traditional model trusts the operator binary — a compiled program with whatever permissions it was given. Orkestra's model trusts the declaration of behavior, which is a much smaller, clearer surface. The declaration is readable, diffable, auditable. The binary is not.

The industry default today is: start permissive, tighten later — maybe. Orkestra takes the opposite approach: start minimal, expand only when required. That is the model Kubernetes itself uses internally for its own controllers. We are applying it one level up, to the operators that extend Kubernetes.

Operators were meant to encode domain knowledge. Somewhere along the way they also became the thing you trusted with your entire cluster. Fixing that is not about better RBAC templates. It is about aligning permissions with intent — and making that alignment automatic.

Orkestra does not ask you to secure your operator. It makes your operator secure by construction.
