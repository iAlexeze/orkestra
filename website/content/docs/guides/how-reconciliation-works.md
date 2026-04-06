---
title: "How Reconciliation Works"
weight: 50
description: "Reconciliation is the heart of Orkestra."
---

Reconciliation is the heart of Orkestra.  
It’s the process that ensures the cluster always matches the desired state described in your Custom Resources (CRs).

This page explains how Orkestra performs reconciliation — without diving into low‑level Kubernetes internals.

---

## The Big Idea

Every CR describes a **desired state**.

Example:

```yaml
spec:
  replicas: 3
  image: nginx:1.27
```

Orkestra’s job is to:

1. Read the desired state  
2. Compare it to the actual cluster state  
3. Make changes until they match  

This loop runs continuously.

---

## Step 1 — Something Changes

Reconciliation begins when:

- A CR is created  
- A CR is updated  
- A dependent resource changes  
- A periodic resync occurs  

Orkestra’s informer detects the change and adds the CR to a **work queue**.

---

## Step 2 — A Worker Picks Up the CR

Each CRD has a pool of workers.

A worker:

- Pulls the CR from the queue  
- Loads its katalog  
- Loads its templates  
- Loads its dependencies  

Workers run in parallel, but each CR is processed serially.

---

## Step 3 — Templates Are Rendered

Orkestra uses the **komposer** to render templates.

Templates may generate:

- Deployments  
- Services  
- ConfigMaps  
- Namespaces  
- Any Kubernetes resource  

Templates can reference fields from the CR:

```yaml
image: {{ .spec.image }}
replicas: {{ .spec.replicas }}
```

---

## Step 4 — Resources Are Applied

Orkestra applies the generated manifests using a safe, declarative approach:

- Create missing resources  
- Update changed resources  
- Leave unchanged resources alone  
- Respect finalizers  
- Handle dependencies in order  

---

## Step 5 — Health Is Updated

After applying resources, Orkestra updates:

- Health status  
- Last reconcile time  
- Error counts  
- Metrics  

You can view this at:

```
/katalog/<crd>/health
```

---

## Step 6 — The Loop Continues

Reconciliation is not a one‑time action.

It repeats whenever:

- The CR changes  
- A dependent resource changes  
- A periodic resync occurs  

This ensures the cluster always converges to the desired state.
