# Profiles 02 — Security

One CR. Three Deployments. Each gets a different security posture from a profile name — no manual `securityContext` or pod security fields needed.

**What you learn:** `securityContext.profile` (container security), `podSecurity.profile` (pod-level security), what each profile enforces.

---

## Profiles at a glance

### Container security (`securityContext.profile`)

| Profile | AllowPrivilegeEscalation | RunAsNonRoot | ReadOnlyRootFilesystem | Capabilities |
|---|---|---|---|---|
| `baseline` | false | — | — | Drop: NET_RAW |
| `restricted` | false | true | — | Drop: ALL |
| `hardened` | false | true | true | Drop: ALL |

`restricted` matches the Kubernetes [restricted Pod Security Standard](https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted).

### Pod security (`podSecurity.profile`)

| Profile | RunAsNonRoot | RunAsUser | RunAsGroup | FSGroup |
|---|---|---|---|---|
| `baseline` | false | — | — | — |
| `restricted` | true | 1000 | — | — |
| `hardened` | true | 65534 | 65534 | 65534 |

---

## Step 1 — Validate

```bash
ork validate
```

---

## Step 2 — Start the runtime

```bash
ork run
```

---

## Step 3 — Open the Control Center

In a **separate terminal**:

```bash
ork control
# username:password → orkestra
```

Open [http://localhost:8081](http://localhost:8081). Select **service-security-profiles**, then **Service**.

---

## Step 4 — Apply the CR

```bash
kubectl apply -f ../cr.yaml
```

Verify the security contexts:

```bash
# Container security context
kubectl get deployment my-service-hardened -o jsonpath='{.spec.template.spec.containers[0].securityContext}' | jq
```

Expected for `hardened`:
```json
{
  "allowPrivilegeEscalation": false,
  "capabilities": {"drop": ["ALL"]},
  "readOnlyRootFilesystem": true,
  "runAsNonRoot": true
}
```

```bash
# Pod security context
kubectl get deployment my-service-hardened -o jsonpath='{.spec.template.spec.securityContext}' | jq
```

Expected for `hardened`:
```json
{
  "fsGroup": 65534,
  "runAsGroup": 65534,
  "runAsNonRoot": true,
  "runAsUser": 65534
}
```

---

## Using a profile in your own Katalog

```yaml
deployments:
  - name: "{{ .metadata.name }}"
    image: "{{ .spec.image }}"
    securityContext:
      profile: restricted   # drops ALL capabilities, runAsNonRoot
    podSecurity:
      profile: restricted   # runAsNonRoot: true, runAsUser: 1000
```

---

## E2E

Run the full lifecycle in one command — applies the CR, asserts all three security profile Deployments are created with the correct securityContext fields, then tears down:

```bash
ork e2e
```

This runs everything defined in [e2e.yaml](./e2e.yaml):

```yaml
expect:
  - name: Restricted profile drops all capabilities
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get deployment my-service-restricted -o jsonpath='{.spec.template.spec.containers[0].securityContext.capabilities.drop[0]}'
        outputContains: ALL

  - name: Hardened profile sets read-only root filesystem
    after: cr-applied
    timeout: 60s
    commands:
      - run: kubectl get deployment my-service-hardened -o jsonpath='{.spec.template.spec.containers[0].securityContext.readOnlyRootFilesystem}'
        outputContains: "true"
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
