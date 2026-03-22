# Declarative Operators: A New Model for Kubernetes Extensibility

*Orkestra Project — March 2026*

---

## Abstract

Kubernetes operators encode domain knowledge as reconciliation logic. Every
major operator framework to date requires this logic to be written in Go,
compiled into a binary, and deployed as a separate long-running process.
Adjacent problems — admission validation, mutation, policy enforcement — are
solved by separate systems: webhook servers, policy engines, admission
controllers. Each system is correct. Together they create a pattern that is
increasingly expensive to operate.

This paper argues that all of these systems exist to compensate for the same
absence: no permanent observer watching what is being provisioned. When that
observer exists, the external delegation each system provides becomes
unnecessary. We describe Orkestra — a runtime for declarative operators that
acts as that observer — and examine the implications for how Kubernetes
extensibility could be redesigned around a unified, declarative interface for
both operator management and admission policy.

---

## 1. The evolution of the operator pattern

### 1.1 The original model

The operator pattern, introduced in 2016, proposed encoding operational
knowledge as a reconciliation loop: a controller that watches a custom
resource and continuously drives the cluster toward the desired state
declared in it. The concept was sound. The implementation required intimate
familiarity with client-go internals — informers, workqueues, REST mappers,
schemes. Most of that implementation was identical across operators. The
business logic was a small fraction of the total code.

### 1.2 Frameworks reduce boilerplate

Kubebuilder and Operator SDK addressed this by generating the plumbing.
Scaffolding commands produced a working controller skeleton. controller-runtime
wrapped client-go into a higher-level interface. The operator developer could
focus on the reconcile function. The cost of entry dropped meaningfully.

The cost did not reach zero. The generated project still required Go, a
build pipeline, an image registry, and a deployment manifest. Adding a new
CRD meant adding a new type, running code generation, rebuilding the binary,
pushing the image, and rolling the deployment. The development loop was
compressed but not eliminated.

### 1.3 Policy arrives as a second system

As operators proliferated, the policy problem became acute. A user applying
a CR with `spec.replicas: 100` against a platform policy capping replicas
at 10 had no immediate feedback. The operator would reconcile the invalid
object and likely fail. The error surfaced in logs, not at the point of apply.

Admission webhooks solved this synchronously: the API server calls an external
HTTP endpoint during the admission request, before the object is stored. The
endpoint validates or mutates the object. The user receives an immediate
rejection or the object is stored with applied defaults.

The infrastructure required to make this work was substantial: a deployed
webhook server, TLS certificates with rotation, webhook configuration objects
registered with the API server, availability guarantees for the webhook server
as a synchronous dependency of the API server itself. This was a second
project alongside the operator.

### 1.4 Policy engines generalise the solution

OPA, Kyverno, and Gatekeeper emerged to remove per-team webhook development.
A shared policy engine accepts declarative policy and enforces it across the
cluster through a single webhook server. This was the right abstraction for
the general case.

But the platform team now maintained three separate systems: the operator
(CRD reconciliation), the policy engine (admission validation and mutation),
and the CRD definitions themselves. Three observability stories. Three
deployment lifecycles. Three failure domains.

### 1.5 The pattern in the progression

Each step in this evolution was correct. The collective progression reveals
something that no individual step made visible: every system that emerged
was compensation for the same absence.

The API server needed webhooks because it had no visibility into operator
domains. Policy engines were needed because operators had no policy model.
Separate webhook servers were needed because controllers and policy ran in
different processes. All of it — the infrastructure, the certificate
management, the additional deployments — exists because there was no
permanent observer watching what was being provisioned across the managed
surface.

Orkestra is that observer.

---

## 2. The Orkestra model

### 2.1 The operator as a declaration

Orkestra introduces two document kinds: **Katalog** and **Komposer**.

A Katalog declares one or more CRDs — their API types, reconcile behavior,
validation constraints, mutation defaults, and namespace restrictions. A
Komposer composes Katalogs from multiple sources into one runtime.

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
        plural: websites
      validation:
        - field: spec.image
          prefix: "myorg/"
          message: "image must be from the myorg registry"
          action: deny
        - field: metadata.labels.team
          operator: exists
          message: "all resources should declare a team owner"
          action: warn
      mutation:
        - field: spec.replicas
          default: "2"
      reconciler:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              reconcile: true
          services:
            - port: "80"
              targetPort: "{{ .spec.port }}"
              reconcile: true
```

This declaration is the complete operator. Apply the `Website` CRD to the
cluster, run `ork run --katalog katalog.yaml`, and:

- Every `Website` CR is watched by an informer
- `spec.image` must have the `myorg/` prefix — CRs that violate this are
  denied reconciliation until corrected
- `metadata.labels.team` is advisory — missing labels surface as active
  warnings without blocking
- `spec.replicas` defaults to 2 on every new CR
- A Deployment and Service are created for each CR and kept in sync
- Deletion cascades via owner references
- Finalizers ensure cleanup completes before the CR is removed
- A health API, Prometheus metrics, and Kubernetes events are provided
  automatically

No Go. No code generation. No webhook server. No policy engine. No additional
deployment.

### 2.2 Two CRD modes, one model

A CRD in Orkestra is either dynamic or typed. The distinction is a single
field: `apiTypes.location`.

**Dynamic** — no location. Orkestra uses the dynamic Kubernetes client.
The CR is represented as `*unstructured.Unstructured`. Template expressions,
conditions, validation, and mutation all operate against the raw field map.
No compiled Go types required. No code generation. `ork run` is the only
command.

**Typed** — location set. Orkestra registers the compiled Go types and uses
a typed REST client. Required only when Go hooks need `obj.(*Website)` — a
concrete type assertion for complex business logic.

The distinction matters less than it appears. Every CRD that ships with Go
API types can run in dynamic mode — the types are not needed for watching,
reconciling, validating, mutating, or applying templates. They are needed
only for type-safe access inside custom Go hooks. For the common case, remove
the location and nothing changes except that no code generation is required.

### 2.3 Managing built-in Kubernetes resources

A Kubernetes Deployment is a CRD that ships with every cluster. It has a
group (`apps`), version (`v1`), kind (`Deployment`), and plural
(`deployments`). Those four values in a Katalog entry are enough for Orkestra
to watch and govern it.

```yaml
- name: deployment-governance
  apiTypes:
    group: apps
    version: v1
    kind: Deployment
    plural: deployments
  validation:
    - field: spec.template.spec.containers[0].image
      prefix: "registry.myorg.io/"
      message: "images must come from the internal registry"
      action: deny
  mutation:
    - field: spec.revisionHistoryLimit
      default: "3"
  restrictedNamespaces:
    - kube-system
    - cert-manager
```

Every Deployment in the cluster now passes through Orkestra's validation
and mutation loop. Images from external registries are denied. The revision
history limit is defaulted. Nothing runs in restricted namespaces. No webhook
infrastructure. No policy engine. No additional tooling.

This is possible because the cluster already holds everything Orkestra needs
to watch a resource. The API server knows the schema of every resource it
serves. Orkestra's discovery client reads the group, version, kind, plural,
and scope from the cluster at startup. The Katalog entry is a pointer — "watch
this" — not a definition of what the resource is.

---

## 3. Validation and mutation as reconciler concerns

### 3.1 Why webhooks exist

Admission webhooks are synchronous HTTP calls from the API server to an
external process. They exist to solve one specific problem: the API server
is general-purpose and knows nothing about your domain. When it needs to ask
"is this object valid?", it must ask externally because the answer lives in
user code.

The infrastructure that surrounds admission webhooks — TLS, separate
deployments, certificate rotation, availability guarantees — all of it exists
to make this external call safe and reliable. The infrastructure is not the
solution. It is the overhead of the coordination pattern.

### 3.2 The observer eliminates the need for external delegation

Orkestra's GenericReconciler sees every CR before any child resources are
created. This makes it a natural validation and mutation point — not an
external one, but one built into the reconcile loop itself.

The observation is straightforward: webhooks are `if/else` and defaults.
Validation is "if field value does not satisfy constraint, reject." Mutation
is "if field absent, set default." Both are evaluations against the CR spec.
Both can be expressed declaratively and evaluated by the same resolver that
evaluates template expressions.

```yaml
validation:
  - field: spec.image
    prefix: "myorg/"
    message: "image must be from myorg registry"
    action: deny     # blocks reconciliation

  - field: metadata.labels.team
    operator: exists
    message: "all resources should declare a team owner"
    action: warn     # advises without blocking

mutation:
  - field: spec.replicas
    default: "1"
  - field: spec.logLevel
    default: "info"
```

### 3.3 The deny/warn model

Validation rules declare an `action`: `deny` blocks reconciliation, `warn`
advises without blocking. Default is `deny` — new rules block unless advisory
mode is explicitly chosen.

**Deny** — reconciliation halts. A Warning Kubernetes event is recorded on
the CR. The `controller_validation_total{result="denied"}` counter increments.
The workqueue retries with backoff. Child resources are not created or updated
until the CR spec is corrected. The user runs `kubectl describe website
my-site` and sees the violation message directly.

**Warn** — reconciliation continues. A Warning event is recorded. The
`controller_validation_total{result="warned"}` counter increments. The
violation is recorded as an active entry on the `/katalog/{crd}` health
endpoint — platform teams see compliance drift in real time without log
parsing. When the CR is corrected, the active warning is cleared automatically.

This is the difference from admission-time rejection: the CR is stored, the
user can see it, the operator records the violation, and reconciliation
proceeds or halts based on the rule's intent. The feedback loop is
sub-second. No network hop. No TLS. No external availability concern.

### 3.4 Additive composition

Rules follow the same additive composition model as CRD definitions. Rules
from a Komposer are platform-level. Rules at the Katalog level extend them.
Rules at the CRD level override for that specific CRD. A platform team can
declare `spec.image` must have the corporate registry prefix — no CRD-level
override can remove that constraint. A CRD that legitimately needs a higher
replica limit can override the platform maximum for that CRD alone.

```yaml
# Komposer — platform-wide
validation:
  - field: spec.image
    prefix: "myorg/"
    message: "image must be from myorg registry"

# CRD level — this CRD overrides the replica constraint only
- name: high-throughput-processor
  validation:
    - field: spec.replicas
      max: "50"          # overrides a Komposer-level max: "10"
      message: "high-throughput processors may scale to 50"
  # spec.image rule still applies — inherited from Komposer
```

### 3.5 Observable policy

Four Prometheus metrics, all unique to Orkestra:

`controller_validation_total{crd, result="passed|denied|warned"}` — the
aggregate validation rate per CRD. Alert on elevated denied or warned rates.

`controller_validation_violations_total{crd, field, rule, action}` — which
fields are violating which rules, separated by action. Shows whether deny
rules are firing frequently (a signal that the constraint is right but clients
need education) or warn rules are persisting (a signal that advisory rules
should be promoted to deny).

`controller_mutation_total{crd}` — how often defaults are being applied.
High rates signal that client tooling is not setting required fields.

`controller_mutation_applied_total{crd, field, type}` — which specific fields
are being defaulted or overridden. Reveals which defaults are actively needed
versus declared but never triggered.

---

## 4. Conditional provisioning

Template declarations support `when` conditions — evaluated before the
registry is called:

```yaml
reconciler:
  default: true
  onCreate:
    deployments:
      - name: "{{ .metadata.name }}"
        image: "{{ .spec.image }}"
        when:
          - field: spec.enabled
            equals: "true"
    services:
      - name: "{{ .metadata.name }}-lb"
        type: LoadBalancer
        when:
          - field: spec.environment
            equals: production
          - field: spec.exposePublicly
            equals: "true"
```

The resolver evaluates all conditions before calling the registry. Failed
conditions skip the resource silently for this reconcile cycle. Nine operators
are supported: `equals`, `notEquals`, `contains`, `prefix`, `suffix`,
`exists`, `notExists`, `gt`, `lt`.

---

## 5. Namespace restrictions

`restrictedNamespaces` declares namespaces where Orkestra will not create
child resources, regardless of what the CR spec requests. Applied before
templates and hooks run. Additive across composition levels — a namespace
restricted at the Komposer level cannot be unrestricted at the CRD level.

```yaml
restrictedNamespaces:
  - kube-system
  - cert-manager
  - kube-*          # all namespaces starting with kube-
  - "*-system"      # all namespaces ending in -system
```

---

## 6. Composition at scale

### 6.1 The Komposer model

A Komposer resolves CRD definitions from multiple sources — files, Helm
charts, remote URLs, environment variables — into one validated runtime
configuration. Sources are merged by CRD name. Inline `spec.crds` on a
Komposer override source definitions.

```yaml
sources:
  files:
    - ./katalogs/project.yaml
    - https://platform.myorg.io/crds/katalog.yaml
    - url: https://private.myorg.io/crds/secure-katalog.yaml
      auth:
        type: bearer
        fromEnv: PLATFORM_KATALOG_TOKEN
  helm:
    - repo: https://charts.myorg.io
      chart: platform-crds
      version: 2.1.0
```

### 6.2 Multi-team ownership

Each team owns their Katalog. The platform Komposer composes them. Policy
is declared at the Komposer level and applies to all CRDs. Environment-specific
overrides are declared inline. The same Katalog runs in development with
`workers: 2` and in production with `workers: 8` — the override is declared,
not hardcoded.

This is the pattern that Helm brought to deployment configuration, applied
to operator behavior. Platform teams publish operator definitions. Application
teams consume and selectively override. The inheritance is explicit, auditable,
and composable.

---

## 7. The Kubernetes-native future

### 7.1 What the architecture implies

Every capability in Orkestra is a declaration interpreted at runtime. The
observer pattern — Orkestra watching every CR it manages — makes external
delegation unnecessary. The composition model — Katalog and Komposer as
first-class YAML documents — makes operator distribution work like package
management.

The logical conclusion is that this observer belongs inside Kubernetes, not
outside it.

### 7.2 A native meta-controller

If Katalog and Komposer became Kubernetes-native resource kinds, registered
by the cluster itself, the Orkestra runtime could run as a core controller
inside `kube-controller-manager`. Every cluster would have an operator runtime
without installation. Platform teams would write Katalogs and Kubernetes would
manage them.

For built-in Kubernetes resources — Deployments, Namespaces, Pods — the Kind
is enough. Kubernetes already holds the schema. The Katalog is a pointer:
"watch Deployments, apply this validation, apply these defaults." No
`apiTypes.location`. No compiled types. Kubernetes knows.

For custom resources, the CRD definition provides the schema at cluster apply
time. The Katalog entry declares Group, Version, Kind, and Plural — the same
information the cluster already has. Orkestra reads it from the discovery
API. Nothing is duplicated.

### 7.3 A unified interface for admission and CRD management

The existing Kubernetes admission model has two entry points: the API server
(webhooks, admission controllers) and the operator runtime (reconcile loops).
They solve related problems with different mechanisms, different infrastructure
requirements, and different observability stories.

When the operator runtime is trusted and in-process, these two entry points
can converge. The same validation declaration that governs reconcile-time
behavior can govern admission-time behavior. The API server calls the
Orkestra validation engine directly — in-process, no network hop, no TLS.
The `ValidatingAdmissionPolicy` mechanism introduced in 1.26 demonstrates
that the Kubernetes project is already moving in this direction. Orkestra's
condition model is the natural evolution: the full declarative operator
stack, composable through Komposers, observable through the health API.

The result is one interface for both problems:
- Admission validation and mutation for any resource
- CRD reconciliation and lifecycle management
- Composable policy inherited across Komposer hierarchies
- Consistent observability through a standard health API

Less tooling. Fewer CRD types per resolution. One native meta-controller for
admission and CRD management — unified, composable, and observable by default.

### 7.4 The path there

This is not immediate. It is the direction. The path runs through production
adoption, through CNCF Sandbox, through a Kubernetes Enhancement Proposal,
through alpha and beta behind a feature gate, to a future release where
`kubectl apply -f my-katalog.yaml` is the complete interaction with an
operator runtime that ships with every cluster.

Every Katalog written today is evidence for the proposal. Every webhook
server decommissioned is evidence that the model works. Every platform team
that simplifies their operator surface is evidence that the direction is right.

The solution will speak for itself.

---

## 8. What this replaces

| Traditional | Orkestra equivalent |
|---|---|
| Go operator binary | Katalog YAML |
| Kubebuilder scaffolding | `ork init` |
| `ork generate runtime` | Generated only for custom Go hooks |
| ValidatingWebhookConfiguration | `validation:` block |
| MutatingWebhookConfiguration | `mutation:` block |
| Webhook TLS + deployment | Built into the reconciler |
| OPA / Kyverno policy | Validation rules in the Katalog |
| Helm chart per operator | Komposer |
| Per-operator metrics | Unified per-CRD metrics |
| Per-operator health checks | Unified health API |
| Admission controller for built-ins | Katalog entry with Kind only |

---

## 9. Conclusion

The operator pattern is the right abstraction for Kubernetes extensibility.
The requirement to implement it in Go, backed by a webhook server for policy
and a policy engine for admission control, has been a constraint of convention
rather than necessity.

All three of these systems exist to compensate for the same absence: no
permanent observer watching the managed surface. Orkestra provides that
observer. When the observer exists, the external delegation each system
provides becomes unnecessary.

Declarative operators are not simpler operators. They are a different model:
operators as data rather than code, composable like any other Kubernetes
resource, governed by the same policy primitives as the resources they manage,
observable through a consistent interface that ships with every operator
automatically.

Kubernetes made infrastructure declarative.
Orkestra makes the operators that extend Kubernetes declarative.
The same principle, applied one level up.
It was always possible.
It just needed someone to build it.

---

*Orkestra — Declarative Operators for Kubernetes*
*March 2026*
*https://github.com/iAlexeze/orkestra*