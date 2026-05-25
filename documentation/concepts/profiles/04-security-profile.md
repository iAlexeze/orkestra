# Security Profile

A security profile is a named preset that expands into a complete `securityContext` and/or `podSecurity` configuration at Katalog load time.

Security profiles follow the [Kubernetes Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/) — `baseline`, `restricted`, and `hardened` map directly to progressively stricter postures.

---

## Profiles

### Container security (`securityContext.profile`)

| Profile | allowPrivilegeEscalation | readOnlyRootFilesystem | runAsNonRoot | capabilities.drop |
|---------|--------------------------|----------------------|--------------|-------------------|
| `baseline` | `false` | — | — | `NET_RAW` |
| `restricted` | `false` | — | `true` | `ALL` |
| `hardened` | `false` | `true` | `true` | `ALL` |

### Pod security (`podSecurity.profile`)

| Profile | runAsNonRoot | runAsUser | runAsGroup | fsGroup |
|---------|-------------|-----------|------------|---------|
| `baseline` | `false` | — | — | — |
| `restricted` | `true` | `1000` | — | — |
| `hardened` | `true` | `65534` | `65534` | `65534` |

`65534` is the `nobody` user — the standard unprivileged UID on Linux.

---

## Usage

Apply a profile to any Deployment, StatefulSet, ReplicaSet, Pod, Job, or CronJob:

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}-api"
      image: "{{ .spec.image }}"
      securityContext:
        profile: restricted
      podSecurity:
        profile: restricted
```

The `securityContext` and `podSecurity` profiles are independent — you can apply them separately or together.

---

## Rules

**Profile or explicit — not both.**

A profile cannot coexist with explicit security fields on the same block.

```yaml
# Valid
securityContext:
  profile: hardened

# Valid
securityContext:
  allowPrivilegeEscalation: false
  runAsNonRoot: true
  capabilities:
    drop: ["ALL"]

# Invalid — rejected at load time
securityContext:
  profile: hardened
  runAsNonRoot: true
```

**Unknown profiles fail fast.** An unrecognized name is a Katalog load error.

**Capability names are validated.** When declaring explicit capabilities, every name in `capabilities.add` and `capabilities.drop` is checked against the known Linux capability set. The special value `ALL` is always accepted. See [Pod Security — validation](../security/07-pod-security.md#validation) for the full capability list.

---

## Profile details

### `baseline`

Minimal hardening. Prevents privilege escalation and drops the most dangerous raw network capability. Suitable for internal services that do not require strict isolation.

**Container:** `allowPrivilegeEscalation: false`, `capabilities.drop: [NET_RAW]`

**Pod:** `runAsNonRoot: false` (not enforced at pod level)

### `restricted`

Matches the Kubernetes **restricted** Pod Security Standard. Drops all capabilities, requires non-root user. Suitable for most production workloads.

**Container:** `allowPrivilegeEscalation: false`, `runAsNonRoot: true`, `capabilities.drop: [ALL]`

**Pod:** `runAsNonRoot: true`, `runAsUser: 1000`

### `hardened`

Maximum isolation. Adds a read-only root filesystem on top of `restricted`. Suitable for high-security workloads.

**Container:** `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, `runAsNonRoot: true`, `capabilities.drop: [ALL]`

**Pod:** `runAsNonRoot: true`, `runAsUser: 65534`, `runAsGroup: 65534`, `fsGroup: 65534`

---

## Choosing a profile

| Situation | Profile |
|-----------|---------|
| Internal service, low isolation requirements | `baseline` |
| Standard production workload | `restricted` |
| High-security workload, full isolation | `hardened` |
| Need specific capabilities (e.g. `NET_BIND_SERVICE`) | explicit fields |

If `hardened` breaks your service (common with apps that write to the container filesystem), use `restricted` and add `readOnlyRootFilesystem: true` only after confirming the app handles a read-only root.
