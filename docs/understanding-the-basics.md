# **Understanding The Basics**

Orkestra is a **declarative operator runtime** for Kubernetes.  
If you’re new to Kubernetes operators, CRDs, or reconciliation, this page gives you the foundation you need before diving deeper.

> [!TIP]
> You don’t need to be an expert in Kubernetes internals to use Orkestra.  
> This guide gives you just enough background to understand what Orkestra automates for you.

---

## What Is a Kubernetes Operator?

A Kubernetes operator is a program that:

- Watches your cluster for changes  
- Understands a custom resource (CR)  
- Continuously works to make the cluster match the desired state  

Traditionally, operators require:

- Writing controllers  
- Managing informers  
- Handling queues  
- Implementing reconciliation loops  
- Managing conversion between API versions  
- Handling finalizers  
- Writing boilerplate code  

Orkestra removes all of that.

---

## What Does Orkestra Do?

Orkestra turns Kubernetes into a **declarative operator platform**.

You describe:

- Your CRD  
- Your templates  
- Your conversion rules  
- Your dependencies  

And Orkestra generates:

- Informers  
- Reconcilers  
- Worker pools  
- Conversion webhooks  
- Health endpoints  
- Metrics  
- Dependency ordering  
- Status reporting  

No controller code.  
No boilerplate.  
No client‑go.  
No informers.  
No deep‑copies.  
No conversion functions.

---

## Why CRDs Matter

A **Custom Resource Definition (CRD)** tells Kubernetes:

- “Here is a new type of object”
- “Here is what it looks like”
- “Here is how it should be stored”

Orkestra extends this by letting you attach:

- Templates  
- Conversion rules  
- Dependencies  
- Health checks  
- Reconciliation behavior  

Your CRD becomes a **declarative operator specification**.

---

## Why Custom Resources Matter

A **Custom Resource (CR)** is an instance of your CRD.

Example:

```yaml
apiVersion: demo.orkestra.io/v1
kind: Website
spec:
  image: nginx:1.27
  replicas: 3
```

Orkestra watches these objects and automatically:

- Converts them between versions  
- Reconciles them  
- Applies templates  
- Manages lifecycle  
- Tracks health  

You don’t write any code — Orkestra does the work.

---

## What Happens When You Apply a CR?

When you run:

```bash
kubectl apply -f website.yaml
```

Orkestra:

1. Receives the event  
2. Converts the object to the storage version  
3. Places it in a work queue  
4. Assigns it to a worker  
5. Runs your templates  
6. Applies the generated manifests  
7. Updates health and metrics  

This is the “operator loop”, but fully automated.

---

## Why Orkestra Exists

Traditional operators are:

- Hard to write  
- Hard to maintain  
- Hard to version  
- Hard to test  
- Hard to evolve  

Orkestra makes operators:

- Declarative  
- Versioned  
- Testable  
- Observable  
- Maintainable  
- Zero‑code  

If you understand the basics above, you’re ready for the next page.
