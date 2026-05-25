# 03 — Adding a profile

## Adding a new name to an existing profile kind

Example: adding `xlarge` to resource profiles.

**1. Add the constant** in `pkg/profiles/resource.go`:

```go
const (
    // ...existing constants...
    ResourceXLarge ResourceProfile = "xlarge"
)
```

**2. Add the expansion case** in `ApplyResourceProfile`:

```go
case ResourceXLarge:
    return &orktypes.ResourceRequirements{
        Requests: map[string]string{"cpu": "1", "memory": "1Gi"},
        Limits:   map[string]string{"cpu": "4", "memory": "4Gi"},
    }, nil
```

**3. Add to `IsValidResourceProfile`**:

```go
case ResourceTiny, ..., ResourceXLarge:
    return true
```

**4. Update the error message** in `ApplyResourceProfile` to include `"xlarge"` in the allowed list.

**5. Add a test case** in `resource_test.go`:

```go
{"xlarge", "xlarge", false, "1", "1Gi", "4", "4Gi"},
```

**6. Add the profile to the fixture** in `pkg/profiles/fixture/katalog-resource.yaml` — add a deployment using `resources.profile: xlarge` and run `ork run` to verify it creates the Deployment with the correct resource requests.

**7. Update the reference table** in `docs/01-profiles.md`.

---

## Adding a new profile kind

Example: adding an HPA profile kind (`hpa.profile`).

**1. Create `pkg/profiles/hpa.go`** following the same structure as the existing files:

```go
package profiles

type HPAProfile string

const (
    HPABurst          HPAProfile = "burst"
    HPATrafficAware   HPAProfile = "traffic-aware"
    HPACostOptimized  HPAProfile = "cost-optimized"
)

func ApplyHPAProfile(name string) (*orktypes.HPASpec, error) { ... }
func IsValidHPAProfile(name string) bool { ... }
```

Export only `Apply*Profile` and `IsValid*Profile`. Keep internal config types unexported.

**2. Wire validation** into `pkg/katalog` — add a `validateHPAProfile()` method on `*Katalog`, call it from `Katalog.Validate()`.

**3. Wire resolution** into `pkg/orkestra-registry/common` — add `ResolveHPA` (or inline in the HPA builder), following the pattern in `resource.go` and `security.go`.

**4. Add tests** in `pkg/profiles/hpa_test.go`.

**5. Add to the fixture** in `pkg/profiles/fixture/katalog-hpa.yaml`.

**6. Document** in `docs/01-profiles.md`.

---

## Rules

- `Apply*Profile` must return an error for unknown names, never silently fall back.
- `IsValid*Profile` must stay in sync with the `Apply*Profile` switch — add to both at the same time.
- Profile names are case-insensitive: normalize with `strings.ToLower` at the top of the switch.
- Template expressions (`{{`) are always skipped at load time — do not add validation for them in `pkg/profiles`.
- Profile and explicit fields are mutually exclusive. Enforce this in `pkg/katalog` validation, not here.
