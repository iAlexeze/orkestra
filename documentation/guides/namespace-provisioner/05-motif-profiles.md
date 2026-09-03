# Profiles via Motif

`04-motif-profiles` shows the same six profile classes as example 03, but the definitions move into the `tenant-policies` motif. The Katalog itself holds no `profiles:` block.

This is how teams share profile definitions across multiple Katalogs — the motif is the single source of truth. Any Katalog that imports it can reference `org-conservative` or `org-safe` without re-declaring them.

---

## The motif owns the registry

`tenant-policies/motif.yaml` declares all six classes:

```yaml
profiles:
  networkPolicies:
    - name: org-deny-all
    - name: org-allow-dns-egress
    - name: org-allow-monitoring

  resourceQuotas:
    - name: org-small
    - name: org-medium
    - name: org-large

  limitRanges:
    - name: org-container-defaults

  hpa:
    - name: org-conservative
    - name: org-burst

  pdb:
    - name: org-at-least-one
    - name: org-majority

  rollingUpdate:
    - name: org-safe
    - name: org-fast
```

The motif also creates the three NetworkPolicies directly (deny-all, allow-dns, allow-monitoring). The Katalog does not need to declare them.

---

## The Katalog imports and references

```yaml
imports:
  - motif: ../motifs/tenant-policies/motif.yaml
    with:
      namespace: "{{ .spec.targetNamespace }}"
      team: "{{ .spec.team }}"
```

After import, the Katalog's registry contains all profiles from the motif. References in `operatorBox` resolve against that merged registry:

```yaml
resourceQuotas:
  - name: "{{ .spec.team }}-quota"
    profile: "{{ printf \"org-%s\" .spec.tier }}"

limitRanges:
  - name: "{{ .spec.team }}-container-limits"
    profile: org-container-defaults

deployments:
  - name: "{{ .spec.team }}-ns-agent"
    rollingUpdate:
      profile: org-safe

hpa:
  - name: "{{ .spec.team }}-ns-agent-hpa"
    behavior:
      profile: org-conservative

pdb:
  - name: "{{ .spec.team }}-ns-agent-pdb"
    behavior:
      profile: org-at-least-one
```

`ork validate` resolves the import, merges the registries, and confirms every name exists — even though the definitions are in the motif, not the Katalog.

---

## Conflict detection

If this Katalog also declared `org-conservative` in its own `profiles:` block, `ork validate` would reject it:

```text
profile conflict: hpa "org-conservative" defined in both motif "tenant-policies" and the katalog
```

The same name in different classes is not a conflict — `resourceQuotas.org-medium` and `hpa.org-medium` are independent.

---

## Try it

```bash
cd 04-motif-profiles
ork validate
ork simulate
ork run
```

---

→ [Back to index](index.md)
