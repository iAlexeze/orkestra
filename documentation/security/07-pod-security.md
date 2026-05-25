# Pod Security

Orkestra surfaces Kubernetes container and pod security settings directly in the Katalog declaration. You can apply a named security profile in one line, or set individual security fields for fine-grained control.

Security contexts are applied at katalog load time — the runtime sees a fully-expanded spec exactly as if you had written the settings manually.

---

## Quick start

Apply the `hardened` profile to every container in a Deployment:

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}-api"
      image: "{{ .spec.image }}"
      securityContext:
        profile: hardened
      podSecurity:
        profile: hardened
```

---

## `securityContext` — container-level settings

Controls security settings applied to each container.

```yaml
securityContext:
  profile: restricted          # ← named preset (see Profiles below)

  # — or — declare individual fields:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000
  capabilities:
    drop: ["ALL"]
    add: ["NET_BIND_SERVICE"]
```

`profile` and explicit fields are **mutually exclusive** — Orkestra rejects the declaration at load time if both are set.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `profile` | string | Named preset. One of: `baseline`, `restricted`, `hardened`. |
| `allowPrivilegeEscalation` | bool | Whether the process can gain more privileges than its parent. Kubernetes defaults to `true` when unset. |
| `readOnlyRootFilesystem` | bool | Mount the container's root filesystem as read-only. |
| `runAsNonRoot` | bool | Require the container to run as a non-root user. |
| `runAsUser` | int64 | UID to run the container entrypoint. |
| `runAsGroup` | int64 | GID to run the container entrypoint. |
| `capabilities.add` | []string | Linux capabilities to add beyond the default set. |
| `capabilities.drop` | []string | Linux capabilities to remove. Use `["ALL"]` to drop all then selectively add back. |

---

## `podSecurity` — pod-level settings

Controls security settings applied to the pod spec. These apply to all init containers and containers in the pod.

```yaml
podSecurity:
  profile: restricted          # ← named preset (see Profiles below)

  # — or — declare individual fields:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
```

`profile` and explicit fields are **mutually exclusive**.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `profile` | string | Named preset. One of: `baseline`, `restricted`, `hardened`. |
| `runAsNonRoot` | bool | All containers must run as a non-root user. |
| `runAsUser` | int64 | UID for all containers in the pod. |
| `runAsGroup` | int64 | GID for all containers in the pod. |
| `fsGroup` | int64 | Supplemental GID applied to all containers. Pod volumes will be owned by this GID. |

---

## Profiles

Profiles are named presets that expand into a complete security configuration at katalog load time. They follow the [Kubernetes Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/).

### `baseline`

Minimal hardening. Suitable for internal services with low isolation requirements.

| Setting | Container | Pod |
|---------|-----------|-----|
| `allowPrivilegeEscalation` | `false` | — |
| `capabilities.drop` | `["NET_RAW"]` | — |
| `runAsNonRoot` (pod) | — | `false` |

### `restricted`

Matches the Kubernetes **restricted** Pod Security Standard. Suitable for most production workloads.

| Setting | Container | Pod |
|---------|-----------|-----|
| `allowPrivilegeEscalation` | `false` | — |
| `runAsNonRoot` | `true` | `true` |
| `capabilities.drop` | `["ALL"]` | — |
| `runAsUser` (pod) | — | `1000` |

### `hardened`

Maximum isolation. Adds a read-only root filesystem on top of `restricted`.

| Setting | Container | Pod |
|---------|-----------|-----|
| `allowPrivilegeEscalation` | `false` | — |
| `readOnlyRootFilesystem` | `true` | — |
| `runAsNonRoot` | `true` | `true` |
| `capabilities.drop` | `["ALL"]` | — |
| `runAsUser` (pod) | — | `65534` (nobody) |
| `runAsGroup` (pod) | — | `65534` |
| `fsGroup` (pod) | — | `65534` |

---

## Supported resource types

`securityContext` and `podSecurity` are available on:

- `deployments`
- `statefulsets`
- `replicasets`
- `pods`
- `jobs`
- `cronjobs`

---

## Validation

Orkestra validates all security declarations at katalog load time, before any reconcile runs.

**Errors raised at load time:**

- Unknown profile name — only `baseline`, `restricted`, `hardened` are accepted.
- Mixed usage — `profile` declared alongside explicit fields.
- Unknown capability name — every entry in `capabilities.add` and `capabilities.drop` is checked against the known Linux capability set (kernel 6.x). The special value `ALL` is always accepted.
- Template expressions (`{{ .spec.profile }}`, `{{ .spec.cap }}`) are skipped at load time and validated at reconcile time when the expression resolves.

**Known Linux capabilities** (all must be upper-case):

`AUDIT_CONTROL`, `AUDIT_READ`, `AUDIT_WRITE`, `BLOCK_SUSPEND`, `BPF`, `CHECKPOINT_RESTORE`, `CHOWN`, `DAC_OVERRIDE`, `DAC_READ_SEARCH`, `FOWNER`, `FSETID`, `IPC_LOCK`, `IPC_OWNER`, `KILL`, `LEASE`, `LINUX_IMMUTABLE`, `MAC_ADMIN`, `MAC_OVERRIDE`, `MKNOD`, `NET_ADMIN`, `NET_BIND_SERVICE`, `NET_BROADCAST`, `NET_RAW`, `PERFMON`, `SETFCAP`, `SETGID`, `SETPCAP`, `SETUID`, `SYS_ADMIN`, `SYS_BOOT`, `SYSLOG`, `SYS_CHROOT`, `SYS_MODULE`, `SYS_NICE`, `SYS_PACCT`, `SYS_PTRACE`, `SYS_RAWIO`, `SYS_RESOURCE`, `SYS_TIME`, `SYS_TTY_CONFIG`, `WAKE_ALARM`

---

## Example: tiered security by environment

```yaml
onCreate:
  deployments:
    - name: "{{ .metadata.name }}-api"
      image: "{{ .spec.image }}"
      when:
        - field: metadata.labels.env
          operator: eq
          value: production
      securityContext:
        profile: hardened
      podSecurity:
        profile: hardened

    - name: "{{ .metadata.name }}-api"
      image: "{{ .spec.image }}"
      when:
        - field: metadata.labels.env
          notEquals: production
      securityContext:
        profile: baseline
```

---

## Example: custom capabilities

```yaml
securityContext:
  allowPrivilegeEscalation: false
  runAsNonRoot: true
  runAsUser: 1000
  capabilities:
    drop: ["ALL"]
    add: ["NET_BIND_SERVICE"]   # needed for port < 1024
```
