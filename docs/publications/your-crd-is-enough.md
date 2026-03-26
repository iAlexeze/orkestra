# YOUR CRD IS ENOUGH

## The Zero‑Code Future of Kubernetes Operators

---

For years, the Kubernetes community has lived with an unspoken contradiction.

We introduced Custom Resource Definitions (CRDs) to make Kubernetes extensible. The promise was simple: describe your API, and Kubernetes would store it, validate it, and serve it. The hard part—the business logic—would be yours to implement.

But somewhere along the way, the promise inverted. Instead of focusing on the CRD, the ecosystem focused on everything around it.

We built frameworks to generate code for us. We built tools to scaffold projects for us. We built libraries to abstract the plumbing. Each generation made the code easier to write, but none removed the need to write it at all.

The result is a generation of platform engineers who spend their days writing infrastructure for their infrastructure. Operators become software projects. CRDs become an afterthought. The tail wags the dog.

This paper argues for a return to first principles. The CRD was always enough. The rest is plumbing.

---

## The Contradiction

Kubernetes was designed for declarative infrastructure. You write YAML. The platform makes it so.

This promise holds everywhere — until you need to extend Kubernetes itself.

The moment you need a custom resource, you leave the declarative world. You write Go. You scaffold controllers. You wire informers and schemes. You manage reconcile loops, retries, and backoff. You build images. You write deployment manifests for your controller. You maintain a project whose primary purpose is to watch another project.

Every major operator framework to date has accepted this as the cost of entry. Kubebuilder, Operator SDK, Metacontroller, Kopf — they each make the code easier, not unnecessary.

This creates a paradox: to make Kubernetes more declarative, you must write imperative code.

Orkestra breaks that paradox.

---

## What "Your CRD Is Enough" Actually Means

The phrase is not a slogan. It is a technical assertion with profound implications.

It means that the CRD contains everything Kubernetes needs to know about your resource.

Kubernetes already:
- Stores the CRD as unstructured JSON
- Knows its schema (if you provided one)
- Validates incoming requests against that schema
- Persists objects in etcd
- Emits events when they change
- Triggers watches for controllers

The only missing piece has always been a runtime that can reconcile CRDs without requiring the user to write code that translates the CRD into Kubernetes API calls.

That runtime is Orkestra.

Orkestra treats your CRD as the API and your Katalog as the operator. No types. No structs. No reconcile loops. No controllers. No webhooks. Just declarations.

---

## The 80% Problem

Every operator ever written contains the same 80% of logic:

- Create a Deployment
- Update a Service
- Sync a Secret
- Merge a ConfigMap
- Set owner references
- Detect drift
- Handle retries
- Emit events
- Expose metrics

This logic is identical across operators. A PostgreSQL operator and a Redis operator do the same things, with different field names. Yet every team rewrites it. Every framework expects you to write it again.

Orkestra provides that 80% as a built‑in library. You declare what you want. Orkestra handles the how.

The remaining 20% — your business logic — you write as hooks. But even there, Orkestra's declarative templates cover most cases. For the Website CRD, creating a Deployment and Service is 10 lines of YAML. No hooks needed.

---

## What We've Been Building Instead

Every operator framework before Orkestra was built around typed CRD structs. Typed structs require:

- code generation
- deep‑copy functions
- scheme registration
- client‑go libraries
- reconcile loops
- patch logic
- drift detection
- error handling
- boilerplate

This entire stack exists because we assumed operators must be written in Go. But Kubernetes itself does not use typed structs for CRDs. It stores everything as `map[string]interface{}`. The typed layer is a convenience for developers—not a requirement for the platform.

Orkestra operates directly on unstructured CRDs—the same way Kubernetes does internally. This eliminates the entire class of problems created by typed APIs:

- no dot‑notation failures
- no schema drift
- no type mismatches
- no code generation
- no compile‑time coupling
- no need for a language at all

Your CRD becomes the source of truth. Your Katalog becomes the behavior. Orkestra becomes the runtime.

---

## The Model

```
CRD → Katalog → Orkestra → Kubernetes
```

- **CRD** defines *what* the resource is
- **Katalog** defines *how* it should behave
- **Orkestra** reconciles it
- **Kubernetes** stores and enforces it

This is the simplest possible operator model—and the most powerful.

---

## What This Unlocks

### 1. Zero‑code operators

Not "less code." Zero code.

A user writes:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
spec:
  crds:
    - name: website
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
      reconciler:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
```

And runs:

```bash
ork run --katalog katalog.yaml
```

That is the entire operator.

### 2. Operators for CRDs you don't own

You can manage any third‑party CRD without touching its code:

- Prometheus
- ArgoCD
- Istio
- External Secrets
- Cert‑Manager
- Any Helm chart
- Any open‑source operator

If Kubernetes accepts the CRD, Orkestra can reconcile it. No fork. No rewrite. No reverse engineering. Just a Katalog.

### 3. Operators that evolve at runtime

No recompilation. No redeployment. No image builds. No version drift. Update the Katalog—the operator changes instantly.

This is the difference between build‑time and runtime.

### 4. True version management

When a CRD has multiple versions, Kubernetes stores only one. The user sees the storage version, not the version they created.

Orkestra's informer watches the version the user wrote. The reconciler sees the original intent. The `ork get --version` command shows what was created.

This means teams can:
- Use different versions simultaneously
- See what they actually wrote
- Upgrade at their own pace
- Write version‑specific templates
- Trust what they see

### 5. Declarative version conversion

The standard approach to multi‑version CRDs requires writing a conversion webhook in Go, deploying it as a separate service, managing TLS certificates, and maintaining conversion logic across versions.

Orkestra eliminates this. Conversion rules are declared in the Katalog:

```yaml
conversion:
  - kind: Website
    storageVersion: v1
    paths:
      - from: v1alpha1
        to: v1
        spec:
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          autoscaling:
            enabled: false
      - from: v1
        to: v1alpha1
        spec:
          image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
```

Orkestra's built‑in webhook applies these rules. No Go. No separate deployment. No TLS management.

### 6. A shared operator ecosystem

The final form of this vision is the OrkestraRegistry—a global registry of versioned operator patterns. A registry entry looks like this:

```yaml
# registry entry: prometheus@v2.45
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: prometheus
  description: Prometheus monitoring stack

crds:
  - name: prometheus
    group: monitoring.coreos.com
    version: v1
    kind: Prometheus
    plural: prometheuses

templates:
  prometheus:
    onCreate:
      deployments:
        - image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          # production‑hardened defaults
```

Teams consume it like this:

```yaml
sources:
  registry:
    katalog: prometheus
    version: v2.45
```

No controller code. No reconcile loops. No operator project. Just a reference to a versioned operator pattern.

This is the difference between a tool and an ecosystem.

---

## The Economics

| Cost Center | Traditional | Orkestra |
|-------------|-------------|----------|
| **Operator development** | Weeks | Minutes |
| **Operator resources** | 50 operators × 200MB = 10GB | 1 runtime × 200MB = 200MB |
| **Version management** | Full‑time engineers | Declarative rules in Katalog |
| **Cluster duplication** | Separate clusters per version | One cluster, all versions |
| **Debugging** | Hours finding storage version | Seconds with `ork get --version` |

The savings are not incremental. They are orders of magnitude.

---

## The One‑Sentence Truth

Every operator framework before Orkestra reduced the code you write. Orkestra removes the need to write code at all.

**Your CRD is enough.**