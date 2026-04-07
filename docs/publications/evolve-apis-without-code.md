# Declarative Version Conversion for Kubernetes CRDs

## Abstract

Kubernetes supports multi-version CRDs, but the ecosystem largely avoids them.

The reason is not conceptual complexity — it is implementation cost.
Conversion requires writing code, operating a webhook service, and maintaining
infrastructure that is disproportionate to the problem being solved.

This paper argues that CRD version conversion is not an imperative problem.
It is a structural mapping problem. When treated as such, conversion can be
expressed declaratively and executed by the operator runtime itself.

We present a model where each CRD version is a first-class operator and
conversion rules are defined as YAML mappings evaluated at runtime.
The conversion webhook is embedded in the runtime, eliminating separate
deployment and infrastructure concerns.

Production results show zero-failure conversion at sub-millisecond latency
with no conversion code written. The implication is broader: once operators
are treated as declarative systems, version conversion becomes a natural
extension of reconciliation — not a separate concern.

---

## 1. The Hidden Cost of Versioning

Kubernetes makes API evolution possible.

It does not make it practical.

The conversion webhook model introduces:

* a second system to build and operate
* tight coupling between API design and infrastructure
* a maintenance burden that grows with every version

The result is predictable:
teams avoid versioning.

APIs remain in `v1alpha1` not because they are unstable,
but because stabilizing them is too expensive.

---

## 2. Reframing Conversion

Conversion is typically framed as:

> “Given an object, compute another representation.”

This framing implies code.

But most conversions are not computations.
They are mappings:

* field → field
* field → nested field
* missing field → default
* removed field → drop

This is structure, not behavior.

---

## 3. The Orkestra Model

Orkestra treats each CRD version as an independent operator:

* separate informer
* separate reconcile loop
* shared runtime

This changes the placement of conversion logic.

Conversion is no longer:

* external (webhook)
* imperative (functions)

It becomes:

* internal (runtime)
* declarative (mapping rules)

---

## **4. Conversion as Templates**

Conversion rules reuse the same mechanism as reconciliation:
template evaluation over object state.

This unifies two previously separate concerns:

* “what resources do I create?”
* “how does this version relate to another?”

Both become:

> transformations over structured data

---

## **5. Elimination of Infrastructure**

The most immediate impact is operational:

* no webhook deployment
* no additional binaries
* no separate scaling concerns
* shared TLS and networking

Conversion ceases to be a system.

It becomes a feature.

---

## **6. Production Evidence**

In a live deployment:

* 62 conversions processed
* 0 failures
* sub-millisecond latency
* no conversion code written

The YAML defining conversion rules is under 25 lines.

The equivalent traditional implementation would require:

* conversion functions
* webhook server
* TLS configuration
* CRD wiring

Hundreds of lines of code and infrastructure.

---

## **7. Implications**

### **Versioning becomes cheap**

Teams can introduce versions without committing to infrastructure work.

### **APIs evolve continuously**

Breaking changes no longer require coordination events.

### **Operators become self-contained**

Reconciliation and conversion live in the same system.

---

## **8. Limits**

Not all conversions are declarative.

Cases requiring:

* external data
* complex logic
* conditional branching

still require code.

But these are exceptions.

---

## **9. Conclusion**

CRD versioning has been limited not by capability, but by cost.

Declarative conversion removes that cost by aligning the implementation
with the nature of the problem: structural transformation.

Once conversion is treated as data:

* it composes with reconciliation
* it shares the same runtime
* it requires no additional system

Versioning shifts from an infrastructure burden to a natural part of API design.
