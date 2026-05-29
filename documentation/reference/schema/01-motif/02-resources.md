# Resources

`resources` is the block where a Motif declares what it creates and manages. The structure mirrors the Katalog `operatorBox` resource blocks. Motif resources are merged into the CRD entry at reconcile time alongside the Katalog's own `operatorBox`.

---

## Block types

```yaml
resources:
  # Special — runs once on CR creation, never on reconcile
  onCreate:
    secrets:
      - ...

  # All other blocks — run on every reconcile
  statefulSets:
    - ...
  deployments:
    - ...
  services:
    - ...
  configMaps:
    - ...
  ingresses:
    - ...
  hpa:
    - ...
  pdb:
    - ...
  namespaces:
    - ...
  serviceAccounts:
    - ...
  roles:
    - ...
  roleBindings:
    - ...
  cronJobs:
    - ...
  custom:
    - ...
```

---

## `resources.onCreate`

Resources declared under `onCreate` are applied exactly once — when the CR is first created. They are never updated on subsequent reconciles.

Use `onCreate` for:
- Secrets with generated values (passwords, API keys) — generate once, never overwrite.
- Seed data that must not be reset on operator restart.

```yaml
resources:
  onCreate:
    secrets:
      - name: "{{ .metadata.name }}-creds"
        once: true
        rotateAfter: "{{ .inputs.passwordRotation }}"
        data:
          password: "{{ randomAlphanumeric 16 }}"
```

`onCreate` and `once: true` are distinct guards:

- `onCreate` — run this block only on CR creation; skip it on all subsequent reconciles.
- `once: true` — check-before-generate idempotency guard. Before evaluating any template, Orkestra checks whether the secret already exists. If it does, skip template evaluation entirely (no-op). If it does not exist, evaluate templates and create.

`once: true` is required when using Orkestra's [random notes](../../orkestra-notes/random.md) — `randomAlphanumeric`, `randomHex`, `randomBase64` — because these are pure non-idempotent functions: they produce a new value on every call. Without `once: true`, the reconcile loop (which runs on every resync) would call the random note on every cycle, generating a new password every 30 seconds and breaking every application using it.

```yaml
resources:
  onCreate:
    secrets:
      - name: "{{ .metadata.name }}-creds"
        once: true
        data:
          password: "{{ randomAlphanumeric 32 }}"
          apiKey:   "{{ randomHex 16 }}"
```

| Condition | Behaviour |
|-----------|-----------|
| Secret does not exist | Evaluate templates → create with generated values |
| Secret already exists | Skip entirely — no template evaluation, no update |

---

## Reconciled resource blocks

All blocks outside `onCreate` are merged into the CRD entry's `onReconcile` phase and run on every reconcile.

A full example from the `postgres` Motif:

```yaml
resources:
  statefulSets:
    - name: "{{ .metadata.name }}-postgres"
      image: "{{ .inputs.image }}"
      replicas: "{{ .inputs.replicas }}"
      serviceName: "{{ .metadata.name }}-postgres-headless"
      port: 5432
      resources:
        profile: "{{ .inputs.resourceProfile }}"
      probes:
        startup:
          type: tcp
          profile: slow-start
        liveness:
          type: tcp
          profile: standard
        readiness:
          type: tcp
          profile: standard
      volumeClaimTemplates:
        - name: data
          storageClass: "{{ .inputs.storageClass }}"
          storageSize: "{{ .inputs.volumeSize }}"
          mountPath: /var/lib/postgresql/data
      env:
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: "{{ .metadata.name }}-creds"
              key: password

  services:
    - name: "{{ .metadata.name }}-postgres-headless"
      headless: true
      port: 5432

    - name: "{{ .metadata.name }}-postgres"
      port: 5432

  deployments:
    - name: "{{ .metadata.name }}-pgadmin"
      image: "{{ .inputs.pgAdminImage }}"
      port: 80
      when:
        - field: inputs.enableUI
          equals: "true"
```

---

## Merge semantics

Resources from all imports are merged into a single CRD entry in declaration order. Name collisions are a hard error — each resource should use `{{ .metadata.name }}` as a prefix to guarantee uniqueness across imports.

| Source | Merge order |
|--------|-------------|
| `imports[0]` resources | First |
| `imports[1]` resources | Second |
| … | … |
| `operatorBox` resources | Last |

The `operatorBox` block always wins on name conflict because it is the Katalog author's own resources.

---

## `status`

`status.fields` declared in a Motif are contributed to the parent CRD's status and written after each reconcile.

```yaml
status:
  fields:
    - path: postgresReady
      value: "{{ allReplicasReady .children.statefulset }}"
    - path: connectionSecret
      value: "{{ .metadata.name }}-creds"
    - path: connectionString
      value: "postgresql://{{ .inputs.username }}@{{ .metadata.name }}-postgres.{{ .metadata.namespace }}.svc.cluster.local:5432/{{ .inputs.database }}"
    - path: pgAdminUrl
      value: "http://{{ .metadata.name }}-pgadmin-svc.{{ .metadata.namespace }}.svc.cluster.local"
```

Status fields from multiple imports are merged. A duplicate `path` is a hard error.

→ Status field schema: [05-status.md](../02-katalog/05-status.md)

---

## `admission`

A Motif can declare its own validation and mutation rules. These are merged with the parent CRD's admission rules.

```yaml
admission:
  validation:
    rules:
      - field: spec.image
        prefix: "myregistry.io/"
        action: deny

  mutation:
    rules:
      - field: spec.replicas
        default: "1"
```

→ Validation schema: [07-validation.md](../02-katalog/07-validation.md)  
→ Mutation schema: [08-mutation.md](../02-katalog/08-mutation.md)

---

→ Next: [03-importing.md](03-importing.md) — importing a Motif into a Katalog  
→ Previous: [01-inputs.md](01-inputs.md)
