# Hook Visitor Pattern

## The problem it solves

Orkestra katalog validation needs to inspect specific fields across every resource template — Deployments, ReplicaSets, StatefulSets, Pods, Services, Jobs, and more — declared across three reconcile phases (onCreate, onReconcile, onDelete).

The naive approach repeats the same loop three times:

```go
// Repeated for onCreate, onReconcile, onDelete — fragile and verbose
if crd.HasOnCreate() {
    for _, d := range crd.OperatorBox.OnCreate.Deployments {
        if !isValidProtocol(d.Protocol) { return err }
    }
    for _, rs := range crd.OperatorBox.OnCreate.ReplicaSets {
        if !isValidProtocol(rs.Protocol) { return err }
    }
    for _, ss := range crd.OperatorBox.OnCreate.StatefulSets {
        if !isValidProtocol(ss.Protocol) { return err }
    }
    // ... same again for onReconcile and onDelete
}
```

This breaks every time a new resource type is added. It requires every validator to know about every resource type. It is easy to miss a type and silently skip validation.

---

## The solution: VisitResources + typed interface

`HookTemplates.VisitResources` iterates every resource slice in declaration order and calls a function for each one:

```go
// pkg/types/hook_temp.go
func (h *HookTemplates) VisitResources(fn func(res interface{})) {
    for _, x := range h.Deployments  { fn(x) }
    for _, x := range h.ReplicaSets  { fn(x) }
    for _, x := range h.StatefulSets { fn(x) }
    for _, x := range h.Pods         { fn(x) }
    for _, x := range h.Services     { fn(x) }
    // ... all other resource types
}
```

A validator defines a small interface for the field it cares about, implements it on the relevant template source types, and uses a type assertion inside `VisitResources`:

```go
// Define the interface — only resources that carry this field implement it
type ported interface {
    GetProtocol() string
}

// Wire it to the four types that have a Protocol field
func (t DeploymentTemplateSource) GetProtocol() string  { return t.Protocol }
func (t ReplicaSetTemplateSource) GetProtocol() string  { return t.Protocol }
func (t StatefulSetTemplateSource) GetProtocol() string { return t.Protocol }
func (t PodTemplateSource) GetProtocol() string         { return t.Protocol }
```

The collector iterates across all phases in one place:

```go
func (c *CRDEntry) CollectPortProtocolEntries() []PortProtocolEntry {
    var out []PortProtocolEntry

    collect := func(phase string, ht *HookTemplates) {
        ht.VisitResources(func(res interface{}) {
            p, ok := res.(ported)      // only fires for types that implement ported
            if !ok { return }

            proto := p.GetProtocol()
            if proto == "" { return }  // empty = use default, nothing to validate

            out = append(out, PortProtocolEntry{
                Phase:    phase,
                Protocol: proto,
            })
        })
    }

    if c.HasOnCreate()    { collect("onCreate",    c.OperatorBox.OnCreate)    }
    if c.HasOnReconcile() { collect("onReconcile", c.OperatorBox.OnReconcile) }
    if c.HasOnDelete()    { collect("onDelete",    c.OperatorBox.OnDelete)    }

    return out
}
```

The validator then becomes trivial:

```go
func (k *Katalog) validatePortProtocols() error {
    for crdName, crd := range k.enabledCRDs {
        for _, e := range crd.CollectPortProtocolEntries() {
            if isTemplateExpr(e.Protocol) { continue }
            if !orktypes.IsValidProtocol(e.Protocol) {
                return errInvalidProtocolForResource(crdName, e.ResourceName, e.Phase, e.Protocol)
            }
        }
    }
    return nil
}
```

---

## The pattern generalised

Any cross-hook field validation follows this structure:

| Step | File | What goes here |
|---|---|---|
| 1. Define the entry type | `pkg/types/hooks_<field>.go` | `type <Field>Entry struct { Phase, ResourceName, ... }` |
| 2. Define the interface | same file | `type <fielded> interface { Get<Field>() ... }` |
| 3. Implement on each type | same file | `func (t DeploymentTemplateSource) Get<Field>() ... { return t.<Field> }` |
| 4. Write the collector | same file | `func (c *CRDEntry) Collect<Field>Entries() []<Field>Entry` |
| 5. Write the validator | `pkg/katalog/validate_<field>.go` | Call `crd.Collect<Field>Entries()`, validate each entry |
| 6. Register | `pkg/katalog/validate.go` | Add `k.validate<Field>()` as the next step |

Existing examples:
- `hooks_rolling_update.go` + `validate_rolling_update_profile.go` — validates `rollingUpdate.profile` on Deployments, ReplicaSets, StatefulSets
- `hooks_sleep.go` — validates `sleep:` duration syntax across all resource types
- `hooks_protocol.go` + `validate_protocol.go` — validates `protocol:` on Deployments, ReplicaSets, StatefulSets, Pods

---

## Why not use reflect?

Reflect would let us scan any struct for fields by name without interface declarations. The interface approach is preferred because:

- **Compile-time safety.** If you add `Protocol` to a new template source type but forget to implement `GetProtocol()`, the type just won't appear in the collector — silently skipped but not broken. With reflect, you might accidentally pick up unrelated fields with the same name.
- **Explicit opt-in.** Only types that actually have the field implement the interface. Services have `Protocol` with different semantics (service-level, not container-level) — they implement `ported` only if that is correct.
- **Readable.** The interface declaration documents intent: "any resource that carries a port protocol field".
- **Testable.** You can test each `Get<Field>()` implementation directly without reflection ceremony.

---

## When to use this pattern

Use it when you need to validate, collect, or transform a specific field that appears on a subset of resource template types, across all reconcile phases. The pattern is specifically for **cross-hook, cross-resource-type** operations.

For single-resource-type or single-phase operations, direct field access is cleaner and more obvious.
