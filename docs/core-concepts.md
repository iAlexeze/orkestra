# **Core Concepts**

This page explains Orkestra’s core building blocks in a **non‑technical**, beginner‑friendly way.

Think of this as your mental map of how Orkestra works.

## CRD (Custom Resource Definition)

A CRD defines:

- The shape of your API  
- The fields users can set  
- The versions you support  

In Orkestra, a CRD also defines:

- Templates  
- Conversion rules  
- Dependencies  
- Reconciliation behavior  

Your CRD becomes the “blueprint” for your operator.

## CR (Custom Resource)

A CR is an instance of your CRD.

Example:

```yaml
apiVersion: demo.orkestra.io/v1
kind: Website
spec:
  image: nginx
```

Orkestra watches CRs and ensures the cluster matches what the CR describes.

## Katalog

The **katalog** is Orkestra’s internal registry of everything it knows about your CRDs.

It stores:

- CRD metadata  
- Templates  
- Conversion rules  
- Health state  
- Worker status  
- Metrics  
- Dependencies  

You can view it through the Orkestra API:

```
/katalog
/katalog/<crd-name>
/katalog/<crd-name>/health
```

It’s your operator’s “control panel”.

## Komposer

The **komposer** is Orkestra’s template engine.

It takes:

- Your CR  
- Your templates  
- Your values  

And produces:

- Kubernetes manifests  

It’s like Helm, but:

- Embedded  
- Declarative  
- Version‑aware  
- CR‑driven  

You don’t write Go code — you write templates.

## Registry
The **Orkestra Registry** is where reusable building blocks live.

Think of it like a **library of operator behaviors**:

- Some katalogs come from **Orkestra itself** (core behaviors)
- Some come from your **organization**
- Some come from **open‑source registries**
- Some come from **your own Git repositories**

A registry contains **katalogs**, and a katalog describes:

- A CRD  
- Its templates  
- Its reconciliation behavior  
- Its defaults  
- Its dependencies  
- Its versioning rules  

Orkestra loads katalogs from registries and uses them to decide **how to act** when a CRD asks for something.

### Core Registry vs. User Registries

### **Core Registry**
Ships with Orkestra.  
Contains built‑in behaviors like:

- Deployment management  
- Service management  
- Namespace creation  
- Generic reconciliation  
- Health checks  
- Metrics  
- Conversion logic  

### **User Registries**
Defined through Komposers.  
Contain:

- Custom CRDs  
- Custom templates  
- Custom operators  
- Organization‑specific behaviors  

Users can mix and match multiple registries.

It’s the heart of Orkestra’s “operator‑as‑data” model.

## Health

Every CRD has a health endpoint:

```
/katalog/<crd>/health
```

It reports:

- Whether workers are running  
- Whether reconciliation is happening  
- Whether errors occurred  
- When the last reconcile happened  
- How many reconciles succeeded  

It’s your operator’s heartbeat.

## Metrics

Orkestra exposes Prometheus metrics:

- Conversion latency  
- Conversion success/failure  
- Reconcile counts  
- Worker activity  
- Queue depth  
- Go runtime metrics  

You can plug Orkestra into:

- Grafana  
- Prometheus  
- Alertmanager  

Instant observability.

## Informer

An **informer** watches Kubernetes for changes.

Orkestra automatically creates informers for:

- Every CRD version  
- Every resource your templates generate  

You don’t write any code — Orkestra wires everything.

## Queue

Every CRD has a work queue.

When something changes:

- The CR is added to the queue  
- Workers pick up items  
- Reconciliation happens  

This ensures:

- Ordering  
- Retry logic  
- Backpressure  
- Stability  

You never touch the queue — Orkestra manages it.

## Reconciler

The **reconciler** is the heart of an operator.

It takes a CR and:

- Reads its desired state  
- Compares it to the cluster  
- Applies templates  
- Updates resources  

In Orkestra, the reconciler is:

- Auto‑generated  
- Generic  
- Declarative  
- Version‑aware  

You don’t write reconciliation logic — Orkestra does.

## Workers

Workers are the “threads” that process CRs.

Each CRD has:

- A worker pool  
- Configurable concurrency  
- Automatic scaling  
- Automatic error handling  

Workers pick items from the queue and run the reconciler.

## Putting It All Together

Here’s the lifecycle of a CR in Orkestra:

1. You apply a CR  
2. Informer detects it  
3. Conversion happens  
4. CR is added to the queue  
5. Worker picks it up  
6. Reconciler runs templates  
7. Resources are applied  
8. Health + metrics update  
9. Orkestra waits for the next change  

This is the operator loop — fully automated.

