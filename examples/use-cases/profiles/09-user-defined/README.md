# Profiles 09 — User-Defined

One CR. One Deployment. Three NetworkPolicies. Two HPAs. Every profile name is declared in the Katalog's `profiles:` block — none are Orkestra built-ins. The Deployment also picks up team-owned resource limits, probe timings, container security, and pod security from user-defined profiles.

**What you learn:** how to declare all six user-defined profile classes; that `ork validate` enforces references against your registry; that profile names are scoped to the Katalog and can be anything meaningful to your team; that user-defined profiles shadow built-in profiles with the same name.

---

## Profiles declared in this Katalog

| Class | Name | Purpose |
|---|---|---|
| `networkPolicies` | `team-deny-all` | Block all ingress and egress |
| `networkPolicies` | `team-allow-internal` | Allow ingress from pods in the same namespace |
| `networkPolicies` | `team-allow-dns` | Allow outbound UDP/TCP 53 for DNS |
| `hpa` | `team-conservative` | 70% CPU, scale-down one pod per minute |
| `hpa` | `team-responsive` | 50% CPU, scale-down 25% per 15 seconds |
| `resources` | `team-api` | 200m/128Mi requests, 1/512Mi limits |
| `probes` | `team-standard` | 10s delay, 15s period, 3 failures |
| `containerSecurity` | `team-baseline` | Drop NET_RAW, no privilege escalation |
| `podSecurity` | `team-nonroot` | runAsUser 1000, fsGroup 1000 |

---

## Step 1 — Validate

```bash
ork validate
```

## Step 2 — Simulate

```bash
ork simulate
```

---

## Step 3 — Start the runtime

```bash
ork run
```

---

## Step 4 — Apply the CR

In a separate terminal:

```bash
kubectl apply -f ../cr.yaml
```

Verify the resources:

```bash
kubectl get deployment,networkpolicies,hpa
```

Verify the resource profile was applied to the Deployment:

```bash
kubectl get deployment my-service -o jsonpath='{.spec.template.spec.containers[0].resources}'
```

Verify the probe profile was applied:

```bash
kubectl get deployment my-service -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}'
```

---

## Using user-defined profiles in your own Katalog

```yaml
profiles:
  resources:
    - name: my-api
      requests:
        cpu: "200m"
        memory: "128Mi"
      limits:
        cpu: "1"
        memory: "512Mi"

  probes:
    - name: my-standard
      initialDelaySeconds: 10
      periodSeconds: 15
      failureThreshold: 3
      timeoutSeconds: 5

  containerSecurity:
    - name: my-baseline
      allowPrivilegeEscalation: false
      capabilities:
        drop: [NET_RAW]

  podSecurity:
    - name: my-nonroot
      runAsNonRoot: true
      runAsUser: 1000

spec:
  crds:
    mycrd:
      operatorBox:
        onCreate:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              port: "{{ .spec.port }}"
              resources:
                profile: my-api
              probes:
                liveness:
                  type: http
                  path: /healthz
                  profile: my-standard
              securityContext:
                profile: my-baseline
              podSecurity:
                profile: my-nonroot
```

---

## E2E

```bash
ork e2e
```

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
