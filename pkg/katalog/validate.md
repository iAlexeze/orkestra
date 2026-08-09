# Katalog Validator System

`pkg/katalog` runs a sequential validation pipeline on every loaded Katalog. Each
validator is a method on `*Katalog` (or a package-level function) in its own
`validate_*.go` file. `ValidateConfig` in `validate.go` is the single entry point
— it calls them in order and returns on the first error.

## Pipeline order

`ValidateConfig` runs validators in 41 numbered steps. Broadly:

1. **Struct validation** — field-level constraints via `konfig.Validate`
2. **Structural consistency** — uniqueness, dependsOn cycles, GVK, defaults
3. **Reconciler mode** — typed/dynamic, hooks/constructor requirements
4. **Resource wiring** — reconcilers, runtime objects, hooks, status
5. **Profile expansion** — autoscale, resources, probes, HPA, PDB, rolling update,
   network policy, resource quota, limit range — profiles are validated then
   expanded in-place so the runtime always sees a fully-formed spec
6. **Workload validation** — autoscale/HPA conflict, port protocols, envFrom suffix
7. **Security** — profiles, capabilities, gateway requirement, deletion/namespace
   protection, admission/mutation operators
8. **Serve** — fields, paths, order, namespace, tokens, target, aliases
9. **Gateway tokens** — duplicates, source exclusivity, OIDC rules
10. **External calls, notes, user profiles** — uniqueness and template validity

## Conventions

**Error display** — every `return fmt.Errorf(...)` in a validator must start with
`failureMark()` as the first format argument. `failureMark()` emits a red `✗`
prefix so all validation errors have a uniform visual marker.

```go
// correct
return fmt.Errorf("%s crd %q: ...", failureMark(), crdName)

// wrong — no failureMark
return fmt.Errorf("crd %q: ...")
```

**Multi-field errors** — errors with several related fields use the boxed block
format from [`validate_gateway_tokens.go`](./validate_gateway_tokens.go): a `──────` top/bottom border, the mark
and headline on the first line, then indented key-value detail lines and a `Fix:`
block. Single-condition errors are plain one-liners.

**No functional changes** — validators must not mutate the Katalog except for
profile expansion (deliberate in-place replacement) and adding warnings. All other
mutations belong in `setDefaults` or the enrichment layer.

**Template expressions** — profile and protocol fields that contain `{{` are
skipped at load time (`orktypes.IsTemplate`). They are evaluated at reconcile time
by the resolver.

## Adding a validator

1. Create `validate_<topic>.go` in `pkg/katalog`.
2. Write the method (or function) with the `failureMark()` convention on every
   error return.
3. Add a call in `ValidateConfig` at the appropriate step number.
4. Write a `validate_<topic>_test.go` with at least one valid and one invalid fixture.
