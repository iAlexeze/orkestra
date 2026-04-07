---
title: "The Orkestra Provider Model"
weight: 50
description: "**Extending declarative operators to external infrastructure.**"
---

**Extending declarative operators to external infrastructure.**

---

## The problem with external resources

Kubernetes operators manage Kubernetes resources. The reconcile loop reads a
CR, creates Deployments and Services, writes status. This is the model
controller-runtime was designed for and the model Orkestra's declarative layer
covers completely.

Real operators manage more than Kubernetes resources. A `DatabaseCluster` CR
creates an RDS instance and a Route 53 record. A `MessageQueue` CR provisions
an SQS queue and attaches an IAM policy. A `MongoUser` CR creates a database
user in Atlas.

The provider model is the third option: external resources declared in the
Katalog, executed by a registered provider library, composable with Kubernetes
resource declarations, publishable to the OrkestraRegistry.

---

## What a provider is

A provider is a Go library that handles a named block in the Katalog's
`onCreate`, `onReconcile`, or `onDelete` sections. AWS ships a provider that
handles `aws:` blocks. MongoDB ships a provider that handles `database:` blocks
with `driver: mongo`.

The provider receives:

- The resolved CR object — all spec fields, metadata, status accessible as
  `map[string]interface{}`
- The declarations from its YAML block — already template-resolved by Orkestra's
  resolver before the provider is called
- A context with cancellation, logger, and request ID
- The Kubernetes client — for reading Secrets, ConfigMaps, and other cluster
  resources the provider may need

The provider returns an error or nil. Orkestra handles retry, backoff, status
patching, and event emission.

---

## The Katalog syntax

```yaml
spec:
  crds:
    - name: application-stack
      reconciler:
        default: true
        onCreate:
          # ── Kubernetes resources (Orkestra core) ────────────────────────
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"

          # ── External resources (provider libraries) ─────────────────────
          aws:
            - s3:
                bucket: "{{ .metadata.name }}-{{ .metadata.namespace }}-assets"
                region: "{{ .spec.region }}"
```

The Katalog author writes YAML. The provider author writes Go. The operator
user writes neither — they apply a CR.

---

## What providers must guarantee

**Idempotency.** Reconcile is called on every cycle — every resync, every watch
event. A provider that creates an RDS instance must check whether it exists
before calling `CreateDBInstance`.

**Declarative intent, not imperative sequence.** The provider receives
declarations of desired state. The correct question is always
"does the current external state match the declared desired state?"

**Safe deletion.** Delete is called once during finalizer execution. If the
external resource does not exist, Delete should return nil — not an error.

**No Kubernetes writes.** Providers read Secrets for credentials and ConfigMaps
for configuration. They do not create, update, or delete Kubernetes resources.

---

## Conclusion

The provider model is the natural endpoint of Orkestra's architecture. The
declarative layer handles Kubernetes resources. Providers handle external
resources. Notes handle data transformation. The boundary between them is clean
and permanent.

When AWS ships `github.com/aws/orkestra-provider-aws`, every Orkestra operator
that registers it gains declarative RDS, S3, Route 53, and IAM — without
writing a line of AWS SDK code.

This is the vision: one runtime, many providers, operators as pure
declarations of intent.
