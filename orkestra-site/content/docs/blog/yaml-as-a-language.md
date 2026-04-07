---
title: "Yaml As A Language"
weight: 7
---

# YAML as a Language in Orkestra

_Composing Operators with Anchors, Aliases, and Merge Keys_

Orkestra Katalogs are written in YAML — but they are not just configuration files.

They are **declarative programs**.

YAML provides a set of native composition primitives — anchors (`&`), aliases (`*`), and merge keys (`<<`) — that allow you to define reusable building blocks and assemble operator behaviour without writing code.

In Orkestra, these are not optional conveniences.
They are the foundation of how complex operators remain **readable, reusable, and evolvable**.

---

## The Shift in Thinking

Traditional operator development uses:

* Go code for abstraction
* Functions and libraries for reuse
* Controllers for behaviour

Orkestra replaces this with:

* YAML structures as primitives
* Anchors as reusable components
* Merge keys as composition

> You are not writing configuration.
> You are composing behaviour.

---

## The Core YAML Primitives

### Anchors (`&`) — Define once

```yaml
baseContainer: &baseContainer
  name: "{{ .metadata.name }}"
  image: "{{ .spec.image }}"
  port: "{{ .spec.port }}"
```

---

### Aliases (`*`) — Reuse anywhere

```yaml
deployment:
  <<: *baseContainer
```

---

### Merge Keys (`<<`) — Compose structures

```yaml
deployment:
  <<: *baseContainer
  replicas: "{{ .spec.replicas }}"
```

---

## Building an Operator from Composable Parts

Instead of defining a full reconciler in one block, break it into **small, reusable pieces**.

```yaml
# ============================================================
# Base building blocks
# ============================================================

# Container definition
baseContainer: &baseContainer
  name: "{{ .metadata.name }}"
  image: "{{ .spec.image }}"
  port: "{{ .spec.port }}"

# Scaling behaviour
scaling: &scaling
  replicas: "{{ .spec.replicas }}"

# Deployment base
deploymentBase: &deploymentBase
  <<: *baseContainer
  <<: *scaling
  reconcile: true

# Service base
serviceBase: &serviceBase
  port: "{{ .spec.port }}"
  targetPort: "{{ .spec.port }}"
  reconcile: true
```

These are not “snippets”.
They are **operator primitives**.

---

## Composing the Reconciler

Now assemble them into behaviour:

```yaml
commonReconciler: &commonReconciler
  reconciler:
    default: true
    onCreate:
      deployments:
        - <<: *deploymentBase
      services:
        - <<: *serviceBase
```

No duplication.
No repetition.
Just composition.

---

Now combine them into richer variants with validation:

```yaml
prodReconciler: &prodReconciler
  reconciler:
    default: true
    onCreate:
      deployments:
        - <<: *deploymentBase
      services:
        - <<: *serviceBase
      validation:
        rules:
          - field: spec.image
            prefix: "myorg/"
            message: "images must come from the internal registry (myorg/)"
            action: deny
```

---

---

## Applying Composition to Multi-Version CRDs

The same approach applies to versioned CRDs.

```yaml
# Versions
storageVersion: &storageVersion v1
alpha: &alpha v1alpha1

# Shared type
commonType: &commonType
  group: demo.orkestra.io
  kind: Website
  plural: websites
```

```yaml
spec:
  crds:
    # v1alpha1 – minimal behaviour
    website-v1alpha1:
      <<: *commonReconciler
      apiTypes:
        <<: *commonType
        version: *alpha

    # v1 – composed, production-ready behaviour
    website-v1:
      <<: *prodReconciler
      apiTypes:
        <<: *commonType
        version: *storageVersion
```

Different versions now express **different levels of capability**, not just schema changes.

---

## Declarative Conversion with Reusable Structures

The same composition model applies to conversion rules.

```yaml
commonSpec: &commonSpec
  image: "{{ .spec.image }}"
  replicas: "{{ .spec.replicas }}"
  port: "{{ .spec.port }}"
```

```yaml
conversion:
  storageVersion: *storageVersion
  paths:
    - from: *alpha
      to: *storageVersion
      spec:
        <<: *commonSpec
        seo:
          enabled: false

    - from: *storageVersion
      to: *alpha
      spec:
        <<: *commonSpec
        theme: "default"
```

---

## What This Enables

Using YAML this way turns a Katalog into a **declarative composition system**:

* Shared primitives across CRDs
* Version-aware behaviour
* Feature layering without duplication
* Safe overrides without forking
* Consistent structure across large Katalogs

There is no need for:

* helper functions
* shared Go libraries
* code generation
* external templating engines

Everything is expressed in **native YAML + Orkestra templates**.

---

## Anti-Patterns

### Copy-paste instead of anchors

If two blocks differ by one field, they should share a base anchor.

---

### Over-composition

Too many nested merges reduce readability.
Prefer **clear, named building blocks**.

---

### Hidden overrides

When merging, be explicit about fields you override.
Clarity matters more than cleverness.

---

## Mental Model

Think of a Katalog as:

> A system of composable building blocks evaluated at runtime

* Anchors = components
* Aliases = reuse
* Merge keys = composition
* Templates = dynamic values

---

## Summary

| Concept     | YAML Feature  | Role in Orkestra              |
| ----------- | ------------- | ----------------------------- |
| Reuse       | Anchors (`&`) | Define shared building blocks |
| Reference   | Aliases (`*`) | Reuse structures              |
| Composition | Merge (`<<`)  | Combine behaviour             |
| Abstraction | Templates     | Dynamic evaluation            |

---

## Final Thought

> YAML is not a limitation in Orkestra.
> It is the language.

And once you start composing operators this way,
you stop thinking in terms of “writing YAML” —
and start thinking in terms of **building systems**.
