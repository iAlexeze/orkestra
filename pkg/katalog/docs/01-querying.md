# 01 — Querying the Katalog

`*Katalog` is the single source of truth for all CRD metadata at runtime. It is constructed once at boot and never mutated — all reads are safe without synchronisation.

## Common accessors

```go
// All enabled CRDs (enriched, validated)
k.EnabledCRDs()  // map[string]orktypes.CRDEntry

// Single CRD by name
entry, err := k.Get("cronjob")  // name is the Katalog map key (lowercase, no spaces)

// Check existence without fetching
k.Exists("cronjob")

// All CRDs including disabled ones
k.AllCRDs()

// Metadata (name, description, version, author, license)
k.Metadata()       // orktypes.KatalogMeta
k.Meta()           // alias
```

## Listing CRDs

```go
// Names of all enabled CRDs
k.CRDNames()  // []string

// Human-readable summary of one CRD
summary, err := k.Describe("cronjob")

// Technical explanation of how Orkestra handles the CRD
explanation, err := k.Explain("cronjob")
```

## Dependency queries

```go
// All CRDs this one depends on
k.Depends("worker", "queue")  // true if worker → queue

// All CRDs that depend on this one
k.Dependents("queue")  // []string

// Startup and shutdown ordering
k.Order()  // []string — topological, dependency-safe

// Full graph: name → dependency names
k.Graph()  // map[string][]string
```

## UI / YAML export

```go
// Clean representation for the Control Center /katalog/raw endpoint.
// Excludes internal fields (Scheme, GVK, etc.) that have json:"-" tags.
k.ToUI()  // *orktypes.KatalogForUI
```

## Controllers

```go
// CRDs that have a non-default reconciler constructor
k.Controllers()  // []string
```

→ Next: [02-dependencies.md](02-dependencies.md)
