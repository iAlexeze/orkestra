# cert-manager

## SecurityConfig → Certificate

`01-cert-manager` wraps cert-manager. Your team creates a `SecurityConfig` CR. Orkestra creates a cert-manager `Certificate`. cert-manager handles the issuance. The TLS Secret appears in the same namespace. The developer never sees the Certificate spec.

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/01-cert-manager
```

---

## The mapping

```text
SecurityConfig CR (internal)   cert-manager Certificate (ecosystem)
───────────────────────────    ──────────────────────────────────────
spec.domain                →   spec.commonName
spec.dnsNames              →   spec.dnsNames
spec.issuer                →   spec.issuerRef.name
                           →   spec.issuerRef.kind: ClusterIssuer
                           →   spec.secretName: <name>-tls  (naming convention enforced)
                           →   spec.renewBefore: "720h"      (org default)
```

The `secretName` convention (`<name>-tls`) and the renewal window (`720h`) are enforced by the Katalog. Every certificate in the platform follows them — enforced, not documented.

---

## The naming convention

```yaml
onCreate:
  customResources:
    - apiVersion: cert-manager.io/v1
      kind: Certificate
      name: "{{ .metadata.name }}-cert"
      namespace: "{{ .metadata.namespace }}"
      spec:
        secretName: "{{ .metadata.name }}-tls"   # enforced convention
        issuerRef:
          name: "{{ .spec.issuer }}"
          kind: ClusterIssuer
        commonName: "{{ .spec.domain }}"
        dnsNames: "{{ .spec.dnsNames }}"
        renewBefore: "720h"                       # org default
```

The developer specifies `domain` and `issuer`. The secret name, the issuer kind, and the renewal window are fixed by the platform — they do not get to override them.

---

## Why this matters at scale

Without the mapping:
- Team A names their secret `webapp-tls-secret`
- Team B names theirs `cert-webapp`
- Team C names theirs whatever cert-manager would auto-generate

With the mapping:
- Every certificate's secret is `<cr-name>-tls`
- Every renewal window is 30 days
- Every certificate uses a `ClusterIssuer`, not a `Issuer`

The platform team makes the decisions once. The policy is in the Katalog, not in documentation that teams might not read.

---

## Try it

```bash
ork init --pack ecosystem-composition
cd ecosystem-composition/01-cert-manager
# Follow steps in README
```

→ [02 — Prometheus](./03-prometheus.md) — one CR creates two Prometheus Operator resources.
