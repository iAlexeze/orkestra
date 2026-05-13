
# ONCOP — The Cross‑Operator Protocol of Orkestra*
*Orkestra Project — April 2026*

---

## The orchestration hierarchy

In a distributed system, no operator is an island.  
A single operator manages one CRD, one concern, one bounded context.  
But real systems are ecosystems: deployments depend on queues, queues depend on databases, databases depend on storage, and autoscalers depend on metrics from other operators.

A platform is not a monolith.  
It is a **network of operators**, each responsible for its own domain, yet required to observe and react to the state of others.

Orkestra formalises this with a protocol:

```
Registry   — in‑process, same‑binary observation
    ↓
ONCOP      — cross‑binary, cross‑cluster observation
    ↓
Resolver   — unified field access (metrics, health, info, events)
```

Where Motif is the reusable unit of *construction*,  
**ONCOP is the reusable unit of *observation*.**

It is the protocol that lets one operator read another operator’s state — safely, consistently, and without hard‑coded URLs.

---

## What ONCOP is

ONCOP — the **Orkestra Native Cross‑Operator Protocol** — is a declarative, typed, URL‑inferable protocol for reading another operator’s:

- **metrics** — queue depth, worker count, throughput, lag
- **health** — state, lastError, heartbeat
- **CR detail** — status, spec, children, conditions
- **CRD info** — operator‑level metrics and children
- **events** — CR‑scoped event streams

It is the **observation layer** of Orkestra.

ONCOP is:

- **typed** — `metrics`, `health`, `cr`, `info`, `events`
- **predictable** — URL shape is derived from type + CRD + selector
- **declarative** — expressed in `cross:` blocks
- **cacheable** — each source can define `cacheFor:`
- **composable** — used by autoscale, status.fields, templates, and Motifs
- **fallback‑aware** — registry → ONCOP → raw endpoint

ONCOP is not a replacement for CRDs.  
It is the **protocol for reading CRDs managed by other operators**.

---

## The ONCOP schema

A cross‑operator declaration in a Katalog looks like:

```yaml
operatorBox:
  cross:
    - crd: loader
      selector:
        name: my-loader
        namespace: default
      source:
        host: "http://loader-runtime:8080"
        type: cr
        cacheFor: 10s
      as: loaderCRInfo
```

This declares:

- **what** to observe (`crd: loader`)
- **which instance** (`selector.name: my-loader`)
- **where** to fetch from (`source.host`)
- **how** to interpret the endpoint (`type: cr`)
- **how long** to cache (`cacheFor: 10s`)
- **under what name** to expose it (`as: loaderCRInfo`)

The result becomes available in templates as:

```
.cross.loaderCRInfo.status.phase
.cross.loaderCRInfo.children.deployment.ready
.cross.loaderCRInfo.metrics.queueDepth
```

ONCOP is the bridge between operators.

---

## ONCOP types

ONCOP defines five observation surfaces:

| Type | Meaning | URL shape |
|------|---------|-----------|
| **metrics** | operator‑level metrics | `/katalog/<crd>` |
| **health** | operator health | `/katalog/<crd>/health` |
| **cr** | CR detail (status, spec, children, metrics) | `/katalog/<crd>/cr/<ns>/<name>` |
| **info** | CRD‑level info (list, metrics, children) | `/katalog/<crd>` |
| **events** | CR‑scoped events | `/katalog/<crd>/cr/<ns>/<name>/events` |

These types are **first‑class** in Orkestra:

```go
const (
    ONCOPMetrics ONCOPType = "metrics"
    ONCOPHealth  ONCOPType = "health"
    ONCOPInfo    ONCOPType = "info"
    ONCOPCR      ONCOPType = "cr"
    ONCOPEvents  ONCOPType = "events"
)
```

Each type corresponds to a stable, versioned endpoint in the operator runtime.

---

## URL inference — the heart of ONCOP

ONCOP eliminates hard‑coded URLs.

Given:

```yaml
source:
  host: "http://localhost:8080"
  type: cr
selector:
  name: my-loader
  namespace: default
crd: loader
```

ONCOP constructs:

```
http://localhost:8080/katalog/loader/cr/default/my-loader
```

No developer writes this URL.  
No operator embeds it.  
No autoscaler hard‑codes it.

The protocol infers it.

This is what makes ONCOP **portable**, **composable**, and **safe**.

---

## The resolver — unified access

Once ONCOP fetches data, it is injected into the resolver under `.cross.*`.

A template can reference:

- `.cross.loaderCRInfo.status.phase`
- `.cross.loaderCRInfo.children.deployment.ready`
- `.cross.loaderHealth.state`
- `.cross.loaderCRDInfo.metrics.queueDepth`

Autoscale conditions can reference:

```yaml
when:
  - field: cross.loader.metrics.queueDepth
    greaterThan: "60"
```

Status fields can reference:

```yaml
loaderState: "{{ .cross.loaderCRInfo.status.phase }}"
loaderHealthy: "{{ .cross.loaderHealth.healthy }}"
```

ONCOP makes cross‑operator data feel local.

---

## Resolution priority

ONCOP is not the only observation path.  
It is part of a layered strategy:

```
1. Informer registry (same binary)
2. ONCOP host (cross binary)
3. Raw endpoint (fallback)
4. Empty result (not found)
```

This ensures:

- **fastest path first**  
- **no unnecessary HTTP calls**  
- **compatibility with non‑Orkestra operators**  
- **predictable fallback behavior**

ONCOP is the middle layer — the cross‑binary, cross‑cluster path.

---

## Autoscaling with ONCOP

Autoscalers often depend on metrics from other operators:

- queue depth from a loader
- lag from a consumer
- throughput from a gateway
- worker count from a processor

ONCOP makes this trivial:

```yaml
when:
  - field: cross.loader.metrics.queueDepth
    greaterThan: "60"
    source:
      host: "http://localhost:8080"
      cacheFor: 10s
```

The autoscaler does not know:

- where the loader runs  
- how its metrics are exposed  
- what its URL is  
- whether it is in‑process or cross‑cluster  

ONCOP abstracts all of it.

---

## Status fields with ONCOP

Operators often want to surface cross‑operator state:

```yaml
status:
  fields:
    - path: loaderState
      value: "{{ .cross.loaderCRInfo.status.phase }}"
    - path: loaderHealthy
      value: "{{ .cross.loaderHealth.healthy }}"
    - path: loaderQueueDepth
      value: "{{ .cross.loaderCRDInfo.metrics.queueDepth }}"
```

This gives users a **single pane of glass** — the Processor CR shows the Loader’s health, metrics, and readiness.

ONCOP makes this possible.

---

## Distribution — ONCOP as a protocol, not a library

ONCOP is not a Go package.  
It is not a client library.  
It is a **protocol** implemented by:

- the Orkestra runtime (server)
- the Orkestra reconciler (client)
- the Orkestra resolver (template engine)
- the Orkestra autoscaler (evaluation engine)

It is versioned, documented, and stable.

Operators that implement ONCOP become **first‑class citizens** in the Orkestra ecosystem.

---

## The composition story

```
Operator A exposes metrics, health, CR detail via ONCOP
    ↓
Operator B declares cross: entries
    ↓
ONCOP fetches data from Operator A
    ↓
Resolver injects .cross.*
    ↓
Autoscaler evaluates conditions
    ↓
Status fields surface cross-operator state
```

This is the **observation pipeline** of Orkestra.

Where Motif composes resources,  
**ONCOP composes operators.**

---

## Summary

ONCOP is the missing piece that makes Orkestra a **multi‑operator platform**:

- typed  
- declarative  
- URL‑inferable  
- cacheable  
- composable  
- cross‑binary  
- cross‑cluster  

It is the protocol that lets operators observe each other without coupling, without hard‑coded URLs, and without bespoke integrations.

ONCOP is to observation what Motif is to construction:  
a reusable, declarative, versioned primitive.
