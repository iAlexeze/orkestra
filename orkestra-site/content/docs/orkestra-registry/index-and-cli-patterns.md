---
title: "Index And Cli Patterns"
weight: 49
---

# Orkestra Registry Index Format

This is the canonical `index.yaml` for an Orkestra Pattern Registry.

It supports:

- multiple patterns  
- multiple versions per pattern  
- OCI + Git sources  
- metadata for discovery  
- digest pinning  
- semver ordering  
- future extensibility  

```yaml
apiVersion: orkestra.io/v1alpha1
kind: PatternIndex
generated: "2026-04-01T12:00:00Z"

patterns:
  postgres:
    name: postgres
    description: Declarative PostgreSQL operator pattern
    latest: "1.0.0"
    versions:
      "1.0.0":
        version: "1.0.0"
        artifactType: application/vnd.orkestra.katalog.v1.tar+gzip
        digest: "sha256:2258726047733aa22ca7fe64e1a6e15c7d00f1c28b62d6a3fe91611b8d340510"
        urls:
          - oci://docker.io/ialexeze/postgres:v1
        created: "2026-04-01T11:58:00Z"
        maintainers:
          - name: Alex Eze
            email: alex@example.com
        keywords:
          - database
          - postgres
          - stateful
        annotations:
          orkestra.io/category: database
          orkestra.io/stability: stable

  redis:
    name: redis
    description: In-memory key-value store pattern
    latest: "0.2.0"
    versions:
      "0.2.0":
        version: "0.2.0"
        artifactType: application/vnd.orkestra.katalog.v1.tar+gzip
        digest: "sha256:..."
        urls:
          - oci://ghcr.io/org/redis-pattern:0.2.0
        created: "2026-03-20T09:00:00Z"
        keywords:
          - cache
          - redis
```

## Design Principles

- **Deterministic**: every version is pinned by digest  
- **OCI‑first**: URLs are OCI references  
- **Git‑compatible**: URLs may also be `https://...`  
- **Extensible**: annotations, keywords, maintainers  
- **Minimal**: no unnecessary fields  
- **Human‑readable**: easy to inspect and diff  

This is the exact level of structure a registry needs — no more, no less.

---

# 2. Pattern Discovery CLI Workflow

This is how `ork` should interact with the registry index.

The workflow is intentionally simple and mirrors Helm’s UX, but tailored to Orkestra’s declarative model.

---

## 2.1. Add a Registry

```
ork registry add myrepo oci://docker.io/ialexeze
```

Stores:

```
~/.orkestra/registries/myrepo.yaml
```

---

## 2.2. List Available Patterns

```
ork registry list myrepo
```

Output:

```
PATTERN     LATEST     DESCRIPTION
postgres    1.0.0      Declarative PostgreSQL operator pattern
redis       0.2.0      In-memory key-value store pattern
```

---

## 2.3. Show Pattern Versions

```
ork registry versions myrepo/postgres
```

Output:

```
VERSION     DIGEST      CREATED
1.0.0       sha256:22…  2026-04-01
```

---

## 2.4. Inspect a Pattern

```
ork registry info myrepo/postgres:1.0.0
```

Output:

```
Name: postgres
Version: 1.0.0
Artifact Type: application/vnd.orkestra.katalog.v1.tar+gzip
Digest: sha256:225872...
Description: Declarative PostgreSQL operator pattern
Maintainers:
  - Alex Eze <alex@example.com>
```

---

## 2.5. Pull a Pattern

```
ork registry pull myrepo/postgres:1.0.0
```

Downloads the bundle into:

```
~/.orkestra/patterns/postgres/1.0.0/
```

---

## 2.6. Apply a Pattern

```
ork apply myrepo/postgres:1.0.0
```

This performs:

1. Pull bundle  
2. Validate artifact type  
3. Validate CRD + Katalog  
4. Install CRD  
5. Install Katalog  
6. Ready for CR creation  

---

## 2.7. Search Patterns

```
ork search postgres
```

Output:

```
postgres (1.0.0) — Declarative PostgreSQL operator pattern
```

Search uses:

- name  
- keywords  
- annotations  

---

# 3. Why This Design Works

- Mirrors Helm’s proven registry model  
- Fully OCI‑compatible  
- Simple enough for Git‑based registries  
- Extensible for future Orkestra features  
- Easy to validate and lint  
- Friendly for both humans and automation  

This is the kind of registry format that can scale to:

- 10 patterns  
- 100 patterns  
- 1,000 patterns  

without becoming a burden.
