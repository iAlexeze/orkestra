# 02 — Webhook Handlers

All handlers are methods on `*WebhookServer`. Each handler is registered on the HTTPS mux inside `Start()` only when the corresponding Katalog capability is declared.

## /validate — Validation handler

Registered when `kat.HasValidationRules()` is true.

```
POST /validate
  → decode AdmissionReview
  → gvrToKey(req.Resource)          look up GVR in the admission registry
  → GetValidationRules(gvrKey)      fetch declared rules
  → evaluateValidationRules(obj, rules, kind)
      for each rule: evaluateOneRule → denial or warning
  → build AdmissionResponse
      denials → allowed: false + Status.Message
      warnings → allowed: true + Warnings list
  → encode and return AdmissionReview
```

All rules are evaluated (not fail-fast) so the user sees all violations in one `kubectl apply` output.

Stats updated: `admissionStats.RecordValidation{Allowed,Denied,Warned}(duration)`.

## /mutate — Mutation handler

Registered when `kat.HasMutationRules()` is true.

```
POST /mutate
  → decode AdmissionReview
  → GetMutationRules(gvrKey)
  → deepCopyMap(original)            work on a copy to preserve original for diff
  → applyMutationRules(ctx, copy, rules, kind)
      for each rule: apply default or override using template resolver
      returns list of fieldChange{Field, OldValue, NewValue, ChangeType}
  → buildJSONPatch(changes)          RFC 6902 "add" or "replace" ops
  → return AdmissionResponse with base64-encoded patch
```

Mutation never blocks — errors return `allowed: true` without a patch.

Stats updated: `admissionStats.RecordMutation{Applied,Skipped}(duration)`.

## /convert — Conversion handler

Registered when `kat.HasConversionPaths()` is true.

```
POST /convert
  → decode ConversionReview
  → for each object:
      unmarshal → get kind → GetConversionRules(kind)
      applyConversion(obj, rules, desiredAPIVersion)
          FindPath(sourceVersion, targetVersion)
          resolveMap(resolver, path.Spec)   — template expressions resolved against source obj
          return converted object with updated apiVersion
  → return ConversionReview with all converted objects
```

Stats updated: `conversionStats.RecordSuccess/Failure(duration)`.

## /deletion-protection — Deletion protection handler

Registered when `kat.IsDeletionProtectionEnabled()` and deletion protection GVRs are present.

```
POST /deletion-protection
  → allow all non-DELETE operations immediately
  → self-protection: block deletion of the deletion-protection webhook itself
  → isCRD check: if customresourcedefinition
      isProtectedCRD(name) → block if managed
      else allow
  → non-CRD Orkestra resource (ObjectSelector already filtered to ours) → always block
```

Stats updated: `protectionStats.RecordBlocked/Allowed()`.

## /namespace-protection — Namespace protection handler

Registered when `kat.IsNamespaceProtectionEnabled()` and namespace protection GVRs are present.

```
POST /namespace-protection
  → allow all non-CREATE/UPDATE operations
  → namespaceRulesForCRD(group, plural) → get NamespaceRules
  → NamespaceRules.IsNamespaceAllowed(namespace)
      Allowed list takes precedence over Restricted list
      → block if namespace not allowed
      → allow otherwise
```

Stats updated: `namespaceStats.RecordBlocked/Allowed()`.

## /strict-mode-protection — Strict mode handler

Registered when `kat.IsStrictModeEnabled()` is true (requires `security.deletionProtection.strictMode: true` in the Katalog).

```
POST /strict-mode-protection
  → allow all non-UPDATE operations immediately
  → extractLabels(oldObject) + extractLabels(newObject)
  → hadLabel = old["orkestra.io/deletion-protection"] == "true"
  → hasLabel = new["orkestra.io/deletion-protection"] == "true"
  → hadLabel && !hasLabel → DENY (label removal blocked)
  → otherwise → ALLOW
```

Enforcement is stateless — each decision is made purely from the labels present in the admission request. No in-process register is maintained.

The `ObjectSelector` on the webhook configuration is set to `orkestra.io/deletion-protection=true`. Kubernetes evaluates this against both the old and new object on UPDATE, so the handler fires exactly when a protected resource is being updated — including the moment the label is removed.

Stats updated: `strictModeStats.RecordBlocked/Allowed()`.

→ Next: [03-registration.md](03-registration.md)
