# **Multi‑Version CRD Conversion**

Orkestra supports **declarative CRD version conversion**, enabling safe evolution of your APIs without breaking existing users.  
You can define:

- **Up‑conversion** (older → newer)  
- **Down‑conversion** (newer → older)  
- **Defaulting** (populate missing fields)  
- **Field renames**  
- **Structural transformations**  
- **Type changes**  
- **Multi‑step version pipelines**  

This allows you to ship new CRD versions while keeping older clients fully functional.

---

## **Why Conversion Matters**

!!! note
    Conversion is essential for long‑lived operators.  
    It lets you evolve your API without forcing all clients to upgrade at once.

Use cases include:

- Renaming fields  
- Moving fields into nested structures  
- Splitting one field into multiple  
- Introducing new defaults  
- Removing deprecated fields  
- Supporting multiple versions simultaneously  

---

# **Basic Example — Renaming Fields**

```yaml
conversion:
  from: v1alpha1
  to: v1beta1
  mapping:
    spec.replicas: spec.scale.replicas
    spec.image: spec.container.image
```

This means:

- `spec.replicas` → `spec.scale.replicas`
- `spec.image` → `spec.container.image`

!!! tip
    Orkestra handles both **up‑conversion** and **down‑conversion** automatically using the same mapping.

---

# **Defaulting Example**

You can set defaults when fields are missing in older versions:

```yaml
conversion:
  from: v1alpha1
  to: v1beta1
  defaults:
    spec.scale.min: 1
    spec.scale.max: 5
```

!!! note
    Defaults apply only when the target field is absent — they do not override user‑provided values.

---

# **Removing Deprecated Fields**

If a field is removed in a newer version, simply omit it from the mapping:

```yaml
conversion:
  from: v1alpha1
  to: v1beta1
  mapping:
    spec.image: spec.container.image
  drop:
    - spec.legacyMode
```

!!! caution
    Dropped fields are **not** preserved when converting back down.  
    Use this only when the field is truly deprecated.

---

# **Type Change Example**

Convert a string into a structured object:

```yaml
conversion:
  from: v1alpha1
  to: v1beta1
  mapping:
    spec.region: spec.location.region
  transforms:
    spec.location:
      type: object
      fields:
        region: string
        zone: string
```

!!! tip
    Type transforms allow you to migrate from flat specs to structured ones without breaking older clients.

---

# **Multi‑Version Pipeline Example**

Orkestra supports multi‑hop conversion:

```
v1alpha1 → v1beta1 → v1
```

Define each step:

```yaml
conversion:
  from: v1alpha1
  to: v1beta1
  mapping:
    spec.image: spec.container.image
```

```yaml
conversion:
  from: v1beta1
  to: v1
  mapping:
    spec.container.image: spec.runtime.image
```

Orkestra automatically chains them:

```
v1alpha1 → v1beta1 → v1
```

!!! note
    You do **not** need to define direct conversion between non‑adjacent versions.

---

# **Selecting Versions with a Komposer**

Different environments can choose different CRD versions without forking the operator.

```yaml
# komposer.yaml
spec:
  crds:
    - name: application
      apiTypes:
        group: platform.myorg.io
        version: v1beta1
        kind: Application
```

Production can run `v1`, development can run `v1beta1`, and older clusters can stay on `v1alpha1`.

!!! tip
    Conversion ensures all versions behave consistently at runtime.

---

# **Runtime Behavior**

When a CR is read:

1. Orkestra detects its version  
2. Converts it to the **internal version** (highest available)  
3. Reconciles using the internal version  
4. Converts back to the **requested version** when writing status  

This ensures:

- Reconciliation always uses the newest schema  
- Older clients still receive the version they expect  
- Status fields remain consistent across versions  

---

# **Full Example — Realistic Application CRD Evolution**

### **v1alpha1**
```yaml
spec:
  image: nginx:1.0
  replicas: 2
```

### **v1beta1**
```yaml
spec:
  container:
    image: nginx:1.0
  scale:
    replicas: 2
```

### **v1**
```yaml
spec:
  runtime:
    image: nginx:1.0
  scale:
    min: 2
    max: 5
```

### **Conversion Rules**

```yaml
# v1alpha1 → v1beta1
conversion:
  from: v1alpha1
  to: v1beta1
  mapping:
    spec.image: spec.container.image
    spec.replicas: spec.scale.replicas
```

```yaml
# v1beta1 → v1
conversion:
  from: v1beta1
  to: v1
  mapping:
    spec.container.image: spec.runtime.image
  defaults:
    spec.scale.min: 1
    spec.scale.max: 5
```

!!! tip
    This is a real‑world pattern: flatten → structured → stable.

---

## Related Documentation

- **Concept:** [CRD Versioning](../concepts/versioning.md)
- **Reference:** [Conversion Rules](../reference/katalog-schema.md#conversion)
- **Next Use Case:** [Go Hooks](./hooks.md)
