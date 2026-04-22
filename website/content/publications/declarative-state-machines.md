---
title: "Declarative State Machines in Kubernetes Operators"
weight: 50
description: "**How Orkestra eliminates the constructor pattern.**"
---

**How Orkestra eliminates the constructor pattern.**

---

## The state machine problem

Many Kubernetes operators implement state machines. A CI/CD pipeline moves
through Pending → Running/build → Running/test → Succeeded or Failed. A
database provisioner moves through Initializing → Provisioning → Ready or
Error. A certificate manager moves through Pending → Requested → Issued → Renewed.

State machines are among the hardest patterns to implement cleanly in Kubernetes.
The Kubebuilder documentation addresses this directly — the CronJob tutorial
shows a simple two-state machine but notes that "the real complexity comes when
you start implementing multi-phase controllers."

The standard answer is a custom reconciler — a Go struct that implements
`reconcile.Reconciler`, reads the current phase from status, and switches on it
to decide what action to take. In Orkestra's terms, this is a **constructor**:
a function that replaces the GenericReconciler entirely and owns the full
reconcile loop.

The constructor works. But it requires Go, a binary build, and a deployment
cycle for every change to the state machine. A new step in the pipeline means
editing Go, rebuilding, pushing an image, rolling the deployment.

Orkestra now provides a declarative alternative that requires none of this.

---

## The Go constructor — what it looked like

The Pipeline operator from Orkestra's example 10 implements a five-state
machine: `Pending → Running → Succeeded | Failed`, with a Job created at each
step transition. Here is the reconcile dispatch from the constructor:

```go
func (r *PipelineReconciler) Reconcile(ctx context.Context, key string) error {
    // ... cache lookup, deep copy ...

    switch pipeline.Status.Phase {
    case "", apiv1.PipelinePhasePending:
        return r.handlePending(ctx, pipeline)
    case apiv1.PipelinePhaseRunning:
        return r.handleRunning(ctx, pipeline)
    case apiv1.PipelinePhaseSucceeded, apiv1.PipelinePhaseFailed:
        return nil // terminal — nothing to do
    default:
        return fmt.Errorf("unknown phase %q", pipeline.Status.Phase)
    }
}
```

And one of the phase handlers:

```go
func (r *PipelineReconciler) handlePending(ctx context.Context, p *apiv1.Pipeline) error {
    if len(p.Spec.Steps) == 0 {
        return r.setPhase(ctx, p, apiv1.PipelinePhaseSucceeded, "no steps defined")
    }

    firstStep := p.Spec.Steps[0]
    jobSpec := orkjobs.Resolve(
        orktypes.JobTemplateSource{
            Name:      fmt.Sprintf("%s-%s", p.Name, firstStep.Name),
            Namespace: p.Namespace,
            Image:     p.Spec.Image,
            Command:   firstStep.Command,
        },
        p.Name,
    )
    if err := orkjobs.Create(ctx, r.kube, p, jobSpec); err != nil {
        return fmt.Errorf("creating step job %q: %w", firstStep.Name, err)
    }

    now := metav1.NewTime(time.Now())
    p.Status.Phase = apiv1.PipelinePhaseRunning
    p.Status.CurrentStep = firstStep.Name
    p.Status.StartTime = &now

    return r.patchStatus(ctx, p)
}
```

The full constructor is ~200 lines of Go. It handles: cache reads, finalizer
management, owner references, status patching, event emission, phase transitions,
Job creation, and completion detection. It works correctly. It is also completely
opaque to anyone who cannot read Go.

---

## The declarative alternative — what it looks like now

The same state machine, declared in a Katalog:

```yaml
operatorBox:
  default: true

  onReconcile:
    jobs:
      # Step 1: create build Job when no phase yet
      - name: "{{ .metadata.name }}-build"
        image: "{{ .spec.image }}"
        command: ["sh", "-c", "{{ index .spec.steps 0 \"command\" }}"]
        when:
          - field: status.phase
            operator: notExists

      # Step 2: create test Job when build succeeded
      - name: "{{ .metadata.name }}-test"
        image: "{{ .spec.image }}"
        command: ["sh", "-c", "{{ index .spec.steps 1 \"command\" }}"]
        when:
          - field: status.phase
            equals: "Running/build"
          - field: children.job.status.succeeded
            operator: gt
            value: "0"

      # Step 3: create notify Job when test succeeded
      - name: "{{ .metadata.name }}-notify"
        image: "{{ .spec.image }}"
        command: ["sh", "-c", "{{ index .spec.steps 2 \"command\" }}"]
        when:
          - field: status.phase
            equals: "Running/test"
          - field: children.job.status.succeeded
            operator: gt
            value: "0"

  status:
    fields:
      - path: phase
        value: "Pending"
        when:
          - field: status.phase
            operator: notExists

      - path: phase
        value: "Running/build"
        when:
          - field: status.phase
            operator: in
            value: "Pending,"

      - path: phase
        value: "Running/test"
        when:
          - field: status.phase
            equals: "Running/build"
          - field: children.job.status.succeeded
            operator: gt
            value: "0"

      - path: phase
        value: "Running/notify"
        when:
          - field: status.phase
            equals: "Running/test"
          - field: children.job.status.succeeded
            operator: gt
            value: "0"

      - path: phase
        value: "Succeeded"
        when:
          - field: status.phase
            equals: "Running/notify"
          - field: children.job.status.succeeded
            operator: gt
            value: "0"

      - path: phase
        value: "Failed"
        when:
          - field: children.job.status.failed
            operator: gt
            value: "0"
```

This is the complete state machine. No Go. No binary build. No deployment cycle.

---

## How it works

The declarative state machine rests on three primitives working together:

**1. `when:` conditions on resource declarations**

Resource creation is gated by conditions evaluated against the current CR state.
`when: [{field: status.phase, operator: notExists}]` means "create this Job
only on the first reconcile, when no phase has been written yet." The condition
is checked before the resource is created — not after. If the condition does not
pass, the resource is skipped entirely.

**2. `when:` conditions on status fields**

Status fields are written conditionally. The same `when:` syntax used for
resource gating now applies to individual status field declarations. The last
field whose conditions pass wins — this is the override semantics that makes
the phase progression work.

Declare terminal states last:

```yaml
- path: phase
  value: "Running/build"   # written first
  when: [...]

- path: phase
  value: "Succeeded"       # written last — overrides Running/build if conditions pass
  when: [...]
```

**3. Children in the resolver context**

Child resources (Jobs, Deployments, CronJobs) are read after every reconcile
and injected into the template resolver under `.children.<lowercase-kind>`.
`children.job.status.succeeded` accesses the `succeeded` field from the child
Job's status. This is how the state machine knows a step completed.

The combination: gated resource creation + gated status writes + child status
access = a complete declarative state machine.

---

## The reconcile loop — how progression happens

Each reconcile cycle does one unit of work. The queue fires again on resync or
on a watch event. The progression is automatic:

```
Reconcile 1:
  status.phase = "" (not yet written)
  → when: notExists passes for build Job → create build Job
  → status field: phase="Pending" written (notExists passes)

Reconcile 2 (triggered by Job creation watch event):
  status.phase = "Pending"
  → when: notExists does NOT pass → build Job skipped (already exists)
  → status field: phase="Running/build" written (in: "Pending," passes)

Reconcile 3..N (resync every 10s):
  status.phase = "Running/build"
  children.job.status.succeeded = 0
  → when: gt 0 does NOT pass → test Job NOT created
  → phase stays "Running/build"

Reconcile N+1 (when build Job completes):
  status.phase = "Running/build"
  children.job.status.succeeded = 1
  → when: gt 0 passes → create test Job
  → status field: phase="Running/test" written

... and so on to Succeeded or Failed
```

This is the Kubernetes controller pattern — level-triggered reconciliation,
where each reconcile reads current state and moves toward desired state by
one step. The controller never "remembers" what it did last. Every decision
is made fresh from current state. This is correct by design.

---

## What Orkestra still provides

The declarative state machine does not replace Orkestra's runtime. Everything
the GenericReconciler provides still applies:

- Informer watching the Pipeline CRD
- Workqueue with deduplication and rate-limited backoff
- Worker pool (configurable concurrency)
- `safeReconcile` — panics caught and logged
- Finalizer management — CR protected from dirty deletion
- Owner references — child Jobs cascade-deleted when CR is deleted
- Kubernetes events — emitted per reconcile
- Status Layer 1 — `Ready` condition written after every reconcile
- Prometheus metrics — reconcile total, duration, queue depth, error rate

The declarative state machine adds: conditional resource creation and
conditional status writes. The runtime does all the heavy lifting.

---

## Comparison

| | Go Constructor | Declarative Katalog |
|---|---|---|
| **Lines of code** | ~200 Go | ~60 YAML |
| **Build required** | Yes | No |
| **Deployment required** | Yes | No |
| **Readable by non-Go engineers** | No | Yes |
| **Auditable in PR review** | Difficult | Trivial |
| **Publishable to OrkestraRegistry** | No | Yes |
| **Override with Komposer** | No | Yes |
| **Finalizer management** | Manual | Automatic |
| **Owner references** | Manual | Automatic |
| **Events** | Manual | Automatic |
| **Metrics** | Partial (must wire metrics) | Automatic |
| **When to use** | External I/O, side effects | Pure Kubernetes resource management |

---

## When the constructor is still right

The declarative model covers state machines that manage Kubernetes resources.
The constructor remains the right answer when:

- **The state machine calls external APIs.** If step 2 of your pipeline
  is "call the payment processor API and wait for confirmation", that is
  a side effect. Declarative templates cannot express side effects. Write a hook.

- **The transitions require complex Go business logic.** If deciding whether
  to advance from Running/build to Running/test requires parsing a JSON report
  and checking 15 fields, a note may not cover it. Write a hook.

- **You are migrating an existing controller-runtime reconciler.** The
  constructor's `Reconcile(ctx, key) error` signature maps directly from
  `Reconcile(ctx, req) (ctrl.Result, error)`. The migration is mechanical.

For everything else — phased execution, sequential Jobs, status-driven
conditionals, multi-step workflows over Kubernetes resources — the declarative
model is simpler, more readable, and faster to iterate on.

---

## The historical record

The Go constructor implementation for the Pipeline operator is preserved in
the Orkestra repository at:

→ `examples/advanced/10-constructor/reconciler/pipeline_reconciler.go`

This file exists as a reference implementation and migration guide. It shows
how the same behavior is implemented in Go, and why the declarative alternative
is preferable for most use cases.

The Git history at the time of the declarative state machine introduction
(April 2026) shows the full before-and-after: the constructor being
superseded by the Katalog declaration in `examples/phases/katalog.yaml`.

---

## Conclusion

State machines in Kubernetes operators are hard because the reconcile loop
is stateless — it reads current state every time and must derive what to do
next purely from what it observes. Implementing this correctly in Go requires
careful attention to finalizers, owner references, error handling, and event
emission on every state transition.

Orkestra's declarative state machine moves the complexity into the runtime.
The Katalog declares what should happen in each state. Orkestra executes it,
handles all the Kubernetes machinery, and makes the progression automatic.

The result is an operator that any engineer on the team can read, understand,
modify, and deploy — without rebuilding anything.
