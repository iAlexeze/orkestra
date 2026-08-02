package katalog

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/validate/content"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateIDPAdditionalFields checks idp.additionalFields.labels/annotations
// keys are syntactically valid Kubernetes label/annotation keys, that
// type: enum fields declare a non-empty enum, and that no key collides with
// idp.fields or between the labels/annotations buckets themselves.
func (k *Katalog) validateIDPAdditionalFields() error {
	for crdName, crd := range k.enabledCRDs {
		if !crd.HasAdditionalIDPFields() {
			continue
		}
		labels := crd.AdditionalLabelFields()
		annotations := crd.AdditionalAnnotationFields()

		seen := make(map[string]string, len(crd.IDP.Fields)+len(labels)+len(annotations))
		for name := range crd.IDP.Fields {
			seen[name] = "fields"
		}

		if err := validateIDPAdditionalBucket(crdName, "labels", labels, seen); err != nil {
			return err
		}
		if err := validateIDPAdditionalBucket(crdName, "annotations", annotations, seen); err != nil {
			return err
		}
	}
	return nil
}

// validateIDPFieldOrder rejects two idp.fields / idp.additionalFields
// entries on the same CRD sharing an explicit (non-zero) order: value.
// order: now decides synthesized validation-rule priority — see
// CRDEntry.DuplicateIDPFieldOrders — not just form layout, so a collision
// is no longer purely cosmetic.
func (k *Katalog) validateIDPFieldOrder() error {
	for crdName, crd := range k.enabledCRDs {
		dups := crd.DuplicateIDPFieldOrders()
		if len(dups) == 0 {
			continue
		}
		orders := make([]int, 0, len(dups))
		for order := range dups {
			orders = append(orders, order)
		}
		sort.Ints(orders)
		return errIDPOrderCollision(crdName, orders[0], dups[orders[0]])
	}
	return nil
}

func errIDPOrderCollision(crd string, order int, names []string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s idp field order collision: order: %s shared by %s
   CRD: %s

Each idp.fields / idp.additionalFields entry needs a distinct order: value
(0/unset doesn't count — any number of fields may leave it unset).
Order also decides which field's violation is reported when more than one fails at
once, not just where it renders on the form.
──────────────────────────────────────────────`, failureMark(), strconv.Itoa(order), strings.Join(names, ", "), crd)
}

// validateIDPNamespace enforces idp.namespace's three structural rules:
//
//   - A namespaced CRD (the default) with idp.enabled: true must declare it
//     — otherwise the Control Center form and any Apply API client have no
//     way to know which namespace a new CR belongs in, and there's no other
//     signal available (see IDPConfig.Namespace).
//   - A cluster-scoped CRD (namespaced: false) must NOT declare it — there
//     is no namespace to resolve into, so setting one is always a mistake.
//   - A templated (non-literal) idp.namespace is incompatible with a CRD
//     whose informer is pinned to watch one fixed namespace
//     (CRDEntry.PinnedToNamespace) — a CR resolved into any other namespace
//     would be created but never reconciled, silently, forever.
func (k *Katalog) validateIDPNamespace() error {
	for crdName, crd := range k.enabledCRDs {
		if !crd.IDPEnabled() {
			continue
		}
		hasNamespace := crd.HasIDPNamespace()

		if crd.IsNamespaced() && !hasNamespace {
			return errIDPNamespaceMissing(crdName)
		}
		if !crd.IsNamespaced() && hasNamespace {
			return errIDPNamespaceOnClusterScoped(crdName, crd.IDP.Namespace)
		}
		if hasNamespace && orktypes.IsTemplate(crd.IDP.Namespace) && crd.PinnedToNamespace() {
			pinned := crd.SingleNamespace()
			if pinned == "" {
				pinned = crd.Namespace
			}
			return errIDPNamespacePinnedConflict(crdName, crd.IDP.Namespace, pinned)
		}
	}
	return nil
}

func errIDPNamespaceMissing(crd string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s idp.enabled: true with no idp.namespace
   CRD: %s

This CRD is namespaced (the default — set namespaced: false to opt out).
idp.namespace tells the Apply API which namespace a new CR belongs in when
the caller (Control Center, curl, CI) doesn't say — without it, self-service
creation has no way to know where to place the CR.

Declare a literal namespace or a template expression, e.g.:
  idp:
    namespace: '{{ teamName }}'
──────────────────────────────────────────────`, failureMark(), crd)
}

func errIDPNamespaceOnClusterScoped(crd, ns string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s idp.namespace set on a cluster-scoped CRD: %q
   CRD: %s

namespaced: false means this CRD has no namespace concept — idp.namespace
has nothing to resolve into. Remove it, or drop namespaced: false if this
CRD should actually be namespaced.
──────────────────────────────────────────────`, failureMark(), ns, crd)
}

func errIDPNamespacePinnedConflict(crd, tmpl, pinned string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s idp.namespace is templated but the informer is pinned to one namespace
   CRD: %s
   idp.namespace: %s
   Pinned to: %q

A templated idp.namespace can resolve to a different namespace per
submission, but this CRD's informer only watches %q — anything created
outside it would exist in the cluster but never be reconciled, silently.
Either make idp.namespace a literal matching the pinned namespace, or widen
the CRD's namespace scope (allowedNamespaces with more than one entry, or
drop the legacy namespace: field) so the informer can see everywhere
idp.namespace might resolve to.
──────────────────────────────────────────────`, failureMark(), crd, tmpl, pinned, pinned)
}

func validateIDPAdditionalBucket(crdName, bucket string, fields map[string]orktypes.IDPFieldConfig, seen map[string]string) error {
	bucketPath := "additionalFields." + bucket
	for key, cfg := range fields {
		// Validate that the key is a valid Kubernetes label/annotation key
		if errs := content.IsLabelKey(key); len(errs) > 0 {
			return errInvalidIDPKey(crdName, bucket, key, errs[0])
		}

		// Check for key collisions across idp.fields and other buckets
		if owner, ok := seen[key]; ok {
			return errIDPKeyCollision(crdName, key, owner, bucketPath)
		}
		seen[key] = bucketPath

		// Validate that the type is one of the allowed types.
		if !orktypes.IsValidIDPFieldType(cfg.Type) {
			return errInvalidIDPType(crdName, bucket, key, cfg.Type)
		}

		// Special validation for enum type: must have non-empty enum values
		if cfg.Type == "enum" && len(cfg.Enum) == 0 {
			return errIDPEnumEmpty(crdName, bucket, key)
		}
	}
	return nil
}

func errInvalidIDPKey(crd, bucket, key, reason string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s Invalid idp.additionalFields.%s key: %q
   CRD: %s

%s

Kubernetes label/annotation keys are [prefix/]name — name must be an
alphanumeric string (max 63 chars, may contain '-', '_', '.') and, if
present, prefix must be a valid DNS subdomain (max 253 chars).
──────────────────────────────────────────────`, failureMark(), bucket, key, crd, reason)
}

func errIDPKeyCollision(crd, key, firstBucket, secondBucket string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s idp field name collision: %q
   CRD: %s
   Declared in both: idp.%s and idp.%s

A field name must be unique across idp.fields and every
idp.additionalFields bucket.
──────────────────────────────────────────────`, failureMark(), key, crd, firstBucket, secondBucket)
}

func errIDPEnumEmpty(crd, bucket, key string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s idp.additionalFields.%s %q has type: enum but no enum: values
   CRD: %s

type: enum requires a non-empty enum: list.
──────────────────────────────────────────────`, failureMark(), bucket, key, crd)
}

// errInvalidIDPType returns an error for invalid field types.
// This ensures that users only use the allowed FieldType values.
func errInvalidIDPType(crd, bucket, key, invalidType string) error {
	return fmt.Errorf(`
──────────────────────────────────────────────
%s Invalid idp.additionalFields.%s %q type: %q
   CRD: %s

Valid types are: string, integer, number, boolean, enum
──────────────────────────────────────────────`, failureMark(), bucket, key, invalidType, crd)
}
