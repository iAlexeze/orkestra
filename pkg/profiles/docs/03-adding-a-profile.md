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

**7. Add a deployment to the use-case example** in `examples/use-cases/profiles/01-resource/katalog.yaml` — add a deployment with `resources.profile: xlarge` alongside the existing ones, and add a row to the README table.

**8. Update the reference table** in `docs/01-profiles.md`.

**9. Update the concept doc** in `documentation/concepts/operatorbox/06-profiles/index.md` — add a row to the resource profiles table.

---

## Adding a new profile kind

For a complete reference implementation, see `pkg/profiles/hpa.go` (HPA behavior profiles). The steps below use a hypothetical PDB profile kind as the example.

**1. Create `pkg/profiles/pdb.go`** following the same structure as the existing files:

```go
package profiles

import (
    "fmt"
    "strings"
    orktypes "github.com/orkspace/orkestra/pkg/types"
)

type PDBProfile string

const (
    PDBStrict    PDBProfile = "strict"
    PDBRelaxed   PDBProfile = "relaxed"
)

func ApplyPDBProfile(name string) (orktypes.PDBBehavior, error) { ... }
func IsValidPDBProfile(name string) bool { ... }
```

Export only `Apply*Profile` and `IsValid*Profile`. Keep internal config types unexported.

**2. Add the type** to `pkg/types/` (e.g., `pdb_behavior.go`) and add a `Behavior *PDBBehavior` field to `PDBTemplateSource` in `types.go`.

**3. Add a `hooks_pdb.go`** to `pkg/types/` following the pattern in `hooks_hpa.go` — define `PDBProfileEntry`, implement `GetPDBBehavior()` on `PDBTemplateSource`, and add `CollectPDBProfileEntries()` to `CRDEntry`.

**4. Wire validation** into `pkg/katalog` — add `validate_pdb_profile.go` with a `validatePDBBehaviorProfiles()` method on `*Katalog`, call it from `ValidateConfig()`.

**5. Wire resolution** into `pkg/orkestra-registry/pdbs/` — expand the profile in `Resolve()`, convert to the Kubernetes type in the builder.

**6. Add tests** in `pkg/profiles/pdb_test.go`.

**7. Add to the fixture** in `pkg/profiles/fixture/katalog-pdb.yaml`.

**8. Add a use-case example** in `examples/use-cases/profiles/` — create a new numbered directory following the existing pattern (katalog.yaml, README.md, cleanup.sh), add it to `examples/use-cases/profiles/README.md`, and add a Try it block to `documentation/concepts/operatorbox/06-profiles/index.md`.

**9. Document** in `docs/01-profiles.md` and `documentation/concepts/operatorbox/06-profiles/index.md`.

---

## Rules

- `Apply*Profile` must return an error for unknown names, never silently fall back.
- `IsValid*Profile` must stay in sync with the `Apply*Profile` switch — add to both at the same time.
- Profile names are case-insensitive: normalize with `strings.ToLower` at the top of the switch.
- Template expressions (`{{`) are always skipped at load time — do not add validation for them in `pkg/profiles`.
- Profile and explicit fields are mutually exclusive. Enforce this in `pkg/katalog` validation, not here.
