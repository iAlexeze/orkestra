# FAQ

This page answers the most common questions about Orkestra, how it works, and how to use it effectively.

---

## What is Orkestra?

Orkestra is a **declarative operator runtime** for Kubernetes.  
It turns CRDs into fully functional operators — without writing controllers, reconcilers, or conversion code.

---

## Do I need to write Go code?

No.

Orkestra generates:

- Informers  
- Reconcilers  
- Worker pools  
- Conversion webhooks  
- Health endpoints  
- Metrics  

You only define:

- CRDs  
- Templates  
- Conversion rules  
- Dependencies  

---

## How does Orkestra differ from Helm or Kustomize?

- **Helm** renders templates  
- **Kustomize** patches manifests  
- **Orkestra** runs a full operator loop

Orkestra continuously:

- Watches CRs  
- Reconciles changes  
- Applies templates  
- Tracks health  
- Handles versioning  
- Manages dependencies  

It’s an operator, not a templating tool.

---

## What is a Katalog?

A **katalog** is a declarative package that describes:

- A CRD  
- Its templates  
- Its reconciliation behavior  
- Its dependencies  
- Its conversion rules  

Katalogs are stored in registries and loaded by Orkestra.

---

## What is the Registry?

The **registry** is where katalogs live.

There are two types:

- **Core registry** — built into Orkestra (Deployments, Services, Namespaces, etc.)  
- **User registries** — Git repos, Helm repos, files, URLs  

The registry tells Orkestra *how* to implement a CRD.

---

## What is the Komposer?

The **Komposer** is how users load katalogs from registries.

It supports:

- Git (public/private)  
- GitHub/GitLab  
- Helm repositories  
- Local files  
- URLs  

It lets teams share and reuse operator behaviors.

---

## Does Orkestra support multi‑version CRDs?

Yes — natively.

Orkestra automatically:

- Registers conversion webhooks  
- Handles version negotiation  
- Converts CRs between versions  
- Tracks conversion metrics  
- Supports multiple API versions simultaneously  

You only define the conversion rules.

---

## How do I debug a CRD?

Check:

1. `/katalog/<crd>/health`  
2. `/katalog/<crd>`  
3. Orkestra runtime logs  
4. Kubernetes events  
5. The generated manifests  

These usually reveal the issue quickly.

---

## Does Orkestra require cert‑manager?

No.

You can use:

- Self‑signed certificates  
- cert‑manager  
- Your own PKI  

cert‑manager is recommended for production.

---

## Can Orkestra manage multiple CRDs?

Yes — Orkestra can manage **any number** of CRDs simultaneously.

Each CRD gets:

- Its own informer  
- Its own queue  
- Its own worker pool  
- Its own health endpoint  
- Its own metrics  

---

## Is Orkestra safe for production?

Yes — Orkestra is designed for production workloads, with:

- Leader election  
- Worker pools  
- Backpressure  
- Conversion webhooks  
- Health endpoints  
- Metrics  
- Dependency ordering  