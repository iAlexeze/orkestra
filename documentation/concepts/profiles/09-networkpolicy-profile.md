# NetworkPolicy Profile

A NetworkPolicy profile is a named preset that expands into a complete set of ingress/egress rules and policy types at reconcile time.

You write a profile name. Orkestra writes the rules.

---

## Profiles

| Profile | Ingress | Egress | Use case |
|---------|---------|--------|----------|
| `deny-all` | Blocked | Blocked | Baseline isolation — no traffic in or out |
| `deny-all-ingress` | Blocked | Unmanaged | Block all inbound, leave egress open |
| `deny-all-egress` | Unmanaged | Blocked | Block all outbound, leave ingress open |
| `allow-same-namespace` | Same namespace only | Unmanaged | Allow pods within the namespace to talk to each other |
| `allow-dns-egress` | Unmanaged | UDP/TCP port 53 only | Allow DNS resolution (combine with other policies) |

---

## Usage

Set `profile` on any `networkPolicies` entry:

```yaml
onCreate:
  networkPolicies:
    - name: "{{ .metadata.name }}-baseline"
      podSelector: {}
      profile: deny-all
      reconcile: true
```

Profiles compose by declaring multiple policies:

```yaml
onCreate:
  networkPolicies:
    - name: "{{ .metadata.name }}-deny-all"
      podSelector: {}
      profile: deny-all
      reconcile: true
    - name: "{{ .metadata.name }}-allow-dns"
      podSelector: {}
      profile: allow-dns-egress
      reconcile: true
```

This is the standard layered approach: start with `deny-all`, then add back only what the workload needs.

---

## How `deny-all` works

Kubernetes interprets an empty ingress or egress slice as "block all". The `deny-all` profile sets:

```yaml
policyTypes:
  - Ingress
  - Egress
ingress: []
egress: []
```

The explicit `policyTypes` declaration is required — without it, an empty `egress` slice does not block egress traffic.

---

## Rules

**Profile or explicit — not both.**

`profile` and explicit `ingress`/`egress`/`policyTypes` fields cannot coexist on the same NetworkPolicy entry. If both are set, the profile takes precedence and the explicit fields are ignored.

**`podSelector` is independent of profile.**

The profile only sets the traffic rules. You still control which pods the policy applies to via `podSelector`. An empty map (`{}`) selects all pods in the namespace.

```yaml
networkPolicies:
  - name: deny-all
    podSelector: {}          # applies to all pods
    profile: deny-all

  - name: deny-worker-egress
    podSelector:
      role: worker           # applies only to pods with role=worker
    profile: deny-all-egress
```

**Unknown profiles log a warning and skip the resource.** A typo does not cause a reconcile failure.

---

## Choosing a profile

| Situation | Profile |
|-----------|---------|
| New namespace — start locked down | `deny-all` |
| Block inbound only, egress already managed | `deny-all-ingress` |
| Block outbound only | `deny-all-egress` |
| Services in the same namespace need to talk | `allow-same-namespace` |
| Workload needs DNS (after deny-all) | `allow-dns-egress` |
| Custom traffic rules | Omit profile, use `ingress`/`egress` directly |
