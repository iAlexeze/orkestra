package katalog

import orktypes "github.com/orkspace/orkestra/pkg/types"

// -----------------------------------------------------------------------------
// Methods
// -----------------------------------------------------------------------------
//
// BuildIndexes builds lookup indexes after CRDs are loaded.
func (k *Katalog) BuildIndexes() {
	k.kindIndex = make(map[string]string)
	k.gvkIndex = make(map[string]string)
	k.gvrIndex = make(map[string]string)
	k.targetIndex = make(map[string]string)

	for name, crd := range k.enabledCRDs {
		k.kindIndex[crd.APITypes.Kind] = name
		k.gvkIndex[crd.GVK().String()] = name
		k.gvrIndex[crd.GVR().String()] = name
		if crd.HasIDPTarget() {
			k.targetIndex[crd.IDPTarget()] = name
		}
	}
}

// BuildAllIDEnabledCRDs builds a slice of all IDP enabled CRDs.
// Use this for iteration when the map key is not needed.
func (k *Katalog) BuildAllIDEnabledCRDs() {
	k.idpEnabledCRDs = make([]*orktypes.CRDEntry, 0, len(k.enabledCRDs))
	for _, crd := range k.enabledCRDs {
		if crd.IDPEnabled() {
			k.idpEnabledCRDs = append(k.idpEnabledCRDs, &crd)
		}
	}
}

// LookupByKind finds the CRD entry whose Kind matches the given kind string.
// O(1) lookup using the kind index.
func (k *Katalog) LookupByKind(kind string) *orktypes.CRDEntry {
	if name, ok := k.kindIndex[kind]; ok {
		if crd, ok := k.enabledCRDs[name]; ok {
			return &crd
		}
	}
	return nil
}

// LookupByGVKString finds the CRD entry whose GroupVersionKind matches the given GVK string.
// O(1) lookup using the GVK index.
func (k *Katalog) LookupByGVKString(gvkString string) *orktypes.CRDEntry {
	if name, ok := k.gvkIndex[gvkString]; ok {
		if crd, ok := k.enabledCRDs[name]; ok {
			return &crd
		}
	}
	return nil
}

// LookupByGVRString finds the CRD entry whose GroupVersionResource matches the given GVR string.
// O(1) lookup using the GVR index.
func (k *Katalog) LookupByGVRString(gvrString string) *orktypes.CRDEntry {
	if name, ok := k.gvrIndex[gvrString]; ok {
		if crd, ok := k.enabledCRDs[name]; ok {
			return &crd
		}
	}
	return nil
}

// LookupByTarget finds the CRD entry whose resolved target matches t.
// O(1) lookup using the target index.
func (k *Katalog) LookupByTarget(t string) *orktypes.CRDEntry {
	if name, ok := k.targetIndex[t]; ok {
		if crd, ok := k.enabledCRDs[name]; ok {
			return &crd
		}
	}
	return nil
}
