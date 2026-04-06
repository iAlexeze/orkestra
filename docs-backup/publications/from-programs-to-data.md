
# Declarative Operators: From Programs to Data

*Orkestra Project — March 2026*

---

## **Abstract**

Kubernetes operators were introduced as programs that encode operational knowledge.
A decade later, the ecosystem has scaled that model — but not questioned it.

This paper argues that the operator model is undergoing a fundamental transition:
from imperative programs to declarative data. The majority of operator behavior
is not inherently procedural. It is structural — the creation, transformation,
and reconciliation of resources based on desired state. The fact that this logic
is implemented as code is an artifact of tooling, not necessity.

We present a model in which operator behavior is expressed as declarative
specifications interpreted by a shared runtime. In this model, operators are no
longer binaries but data structures: versioned, composable, and distributable.
We examine the implications of this shift for system design, ecosystem dynamics,
and the future role of imperative code.

The conclusion is not that all operators become declarative — but that most do.
And once that threshold is crossed, the ecosystem reorganizes around it.

---

## **1. The original operator model**

The operator pattern, introduced in 2016, defined a simple idea:
encode operational knowledge in software.

A controller watches a resource, compares desired state to actual state,
and takes action to converge the two. The implementation model was explicit:
write a program — typically in Go — that performs this reconciliation.

This model worked because it matched the underlying Kubernetes architecture.
Core controllers are programs. Custom controllers followed the same pattern.

The assumption was implicit but powerful:

> Operators are programs.

---

## **2. The mismatch**

At small scale, the assumption holds. At ecosystem scale, it begins to fracture.

Consider what most operators actually do:

* Create a Deployment when a resource is created
* Create a Service exposing that Deployment
* Populate a Secret with credentials
* Update status fields based on resource health
* Handle version upgrades through predictable transformations

This is not arbitrary computation. It is structured transformation.

The logic is:

* deterministic
* repeatable
* largely side-effect bounded within Kubernetes

Yet it is implemented as imperative code:

* reconcile loops
* client calls
* error handling
* retry logic

The majority of this code does not encode domain knowledge.
It encodes *control flow*.

This is the mismatch:
we are using a general-purpose programming model to express
a problem that is largely declarative.

---

## **3. A different lens: operators as data**

If operator behavior is primarily structural, it can be described.

A declarative operator specifies:

* **What resources exist**
* **How they are derived from the CRD spec**
* **How they evolve across versions**
* **What dependencies exist between them**

This is not fundamentally different from how Kubernetes itself operates.
A Deployment is not a program. It is a declaration interpreted by a controller.

Extending this idea:

> An operator is a higher-order declaration — a declaration that produces declarations.

Under this model, the operator is no longer the runtime.
It is the input to a runtime.

---

## **4. The shared runtime model**

Once operators are expressed as data, a new architecture becomes possible.

A single runtime can:

* watch multiple CRDs
* interpret multiple operator specifications
* reconcile all of them within one process

This is not a new idea in Kubernetes.
The `kube-controller-manager` already runs dozens of controllers in one binary.

The difference is that those controllers are compiled in.
Declarative operators externalize them.

This creates a separation:

* **Runtime**: generic, shared, stable
* **Operators**: declarative, versioned, replaceable

The operator becomes configuration.

---

## **5. The 90% threshold**

Not all operators can be declarative.

Some require:

* external API calls
* complex orchestration logic
* non-deterministic workflows

These remain imperative.

But they are the minority.

As declarative systems mature, a pattern emerges:

* The common cases become declarative first
* Repeated imperative patterns are abstracted into primitives
* The boundary of what requires code shrinks over time

This is the same progression seen in:

* infrastructure (shell scripts → Terraform)
* application deployment (scripts → Helm → GitOps)

The critical threshold is not 100%.

It is when **most operators are declarative** — typically 90–99%.

At that point:

* the default changes
* the ecosystem reorganizes
* imperative operators become exceptions, not the norm

---

## **6. Distribution changes everything**

When operators are programs, they are distributed as binaries.

When operators are data, they can be distributed as artifacts.

This has second-order effects:

* Versioning becomes semantic, not tied to builds
* Composition becomes trivial — merge specifications
* Overrides become standard — no forks required
* Discovery aligns with other declarative ecosystems

The operator becomes a dependency, not a deployment.

This is the missing layer in Kubernetes today.

---

## **7. Implications**

### **For platform teams**

Operators become lightweight additions rather than infrastructure decisions.
The cost of adding a new capability approaches the cost of writing YAML.

### **For operator authors**

The unit of contribution shifts from codebases to patterns.
Complexity is pushed to the edges, not required by default.

### **For the ecosystem**

The center of gravity moves:

* from frameworks to registries
* from binaries to specifications
* from runtime isolation to runtime composition

---

## **8. Conclusion**

The operator model is not being replaced.
It is being reframed.

Operators began as programs because programs were the only available tool.
As declarative systems evolve, that constraint disappears.

Most operator behavior is not inherently imperative.
It is declarative structure expressed through imperative means.

Once that structure is made explicit:

* operators become data
* runtimes become shared
* ecosystems become composable

The shift is gradual, then sudden.

Not all operators will become declarative.
But most will.

And when they do, the definition of an operator changes with them.

---

<!-- ## To do
* “Operators are not software. They are configuration pretending to be software.”
* “The industry standardized on Go because it lacked a better abstraction.”
* “The future operator is not compiled.”


* a **KubeCon keynote-style talk**
* a **blog series (3 parts: problem → model → ecosystem)**
* or a **Twitter/X thread that actually spreads** -->
