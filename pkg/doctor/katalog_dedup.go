package doctor

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DeduplicateKatalogGVKs reads the merged katalog at path, collapses all CRDs
// that share the same GVK into one, and writes the result back.
//
// When ork template merges multiple single-app katalogs, each produces a CRD
// with apiTypes.kind: ConfigMap. Orkestra forbids duplicate GVKs at runtime,
// so this must be called before ork generate bundle in the multi-app flow.
//
// Merge strategy for each duplicate-GVK group:
//   - operatorBox lifecycle hooks (onCreate, onReconcile, …): list fields concatenated
//   - labelSelector: union of all key-value pairs (first value wins on collision)
//   - allowedNamespaces: union of all namespace strings
//   - status: kept from the first (canonical) CRD — all duplicates are identical
func DeduplicateKatalogGVKs(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading katalog: %w", err)
	}

	var root map[string]interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parsing katalog: %w", err)
	}

	spec, _ := root["spec"].(map[string]interface{})
	if spec == nil {
		return nil
	}
	crds, _ := spec["crds"].(map[string]interface{})
	if crds == nil {
		return nil
	}

	// Group CRD names by GVK string.
	byGVK := map[string][]string{}
	for name, raw := range crds {
		if crd, ok := raw.(map[string]interface{}); ok {
			gvk := crdGVKString(crd)
			byGVK[gvk] = append(byGVK[gvk], name)
		}
	}

	// Merge each duplicate group into the first (canonical) CRD.
	for _, names := range byGVK {
		if len(names) <= 1 {
			continue
		}
		canonical, _ := crds[names[0]].(map[string]interface{})
		if canonical == nil {
			continue
		}
		for _, dup := range names[1:] {
			dupCRD, _ := crds[dup].(map[string]interface{})
			if dupCRD == nil {
				continue
			}
			katalogMergeHooks(canonical, dupCRD)
			katalogMergeLabelSelector(canonical, dupCRD)
			katalogMergeAllowedNamespaces(canonical, dupCRD)
			delete(crds, dup)
		}
	}

	out, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("marshaling katalog: %w", err)
	}
	return os.WriteFile(path, out, 0o644)
}

func crdGVKString(crd map[string]interface{}) string {
	apiTypes, _ := crd["apiTypes"].(map[string]interface{})
	kind, _ := apiTypes["kind"].(string)
	group, _ := apiTypes["group"].(string)
	version, _ := apiTypes["version"].(string)
	if version == "" {
		version = "v1"
	}
	return group + "/" + version + "/" + kind
}

// katalogMergeHooks appends list fields from each lifecycle hook in src into
// the corresponding hook in dst. The status section is intentionally skipped.
func katalogMergeHooks(dst, src map[string]interface{}) {
	srcBox, _ := src["operatorBox"].(map[string]interface{})
	if srcBox == nil {
		return
	}
	dstBox, _ := dst["operatorBox"].(map[string]interface{})
	if dstBox == nil {
		dst["operatorBox"] = srcBox
		return
	}
	for _, hook := range []string{"onCreate", "onReconcile", "onDelete"} {
		srcHook, _ := srcBox[hook].(map[string]interface{})
		if srcHook == nil {
			continue
		}
		dstHook, _ := dstBox[hook].(map[string]interface{})
		if dstHook == nil {
			dstBox[hook] = srcHook
			continue
		}
		for key, srcVal := range srcHook {
			srcList, ok := srcVal.([]interface{})
			if !ok {
				continue
			}
			if dstList, ok := dstHook[key].([]interface{}); ok {
				dstHook[key] = append(dstList, srcList...)
			} else {
				dstHook[key] = srcList
			}
		}
	}
}

func katalogMergeLabelSelector(dst, src map[string]interface{}) {
	srcSel, _ := src["labelSelector"].(map[string]interface{})
	if srcSel == nil {
		return
	}
	dstSel, _ := dst["labelSelector"].(map[string]interface{})
	if dstSel == nil {
		dst["labelSelector"] = srcSel
		return
	}
	for k, v := range srcSel {
		if _, exists := dstSel[k]; !exists {
			dstSel[k] = v
		}
	}
}

func katalogMergeAllowedNamespaces(dst, src map[string]interface{}) {
	srcNS, _ := src["allowedNamespaces"].([]interface{})
	if srcNS == nil {
		return
	}
	dstNS, _ := dst["allowedNamespaces"].([]interface{})
	if dstNS == nil {
		dst["allowedNamespaces"] = srcNS
		return
	}
	seen := make(map[interface{}]bool, len(dstNS))
	for _, v := range dstNS {
		seen[v] = true
	}
	for _, v := range srcNS {
		if !seen[v] {
			dstNS = append(dstNS, v)
			seen[v] = true
		}
	}
	dst["allowedNamespaces"] = dstNS
}
