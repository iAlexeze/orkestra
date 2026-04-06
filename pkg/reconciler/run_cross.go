// pkg/reconciler/run_cross.go
//
// Cross-CRD HTTP fallback — fetches a CR detail from another Orkestra instance.
//
// The primary path (informer cache, zero API calls) lives in run_template_reconcile.go
// in ReadCrossFromDeclarations. That function calls fetchCrossViaHTTP when the
// informer is unavailable (cross-binary, cross-cluster).
//
// The full informer-cache path requires threading the kontroller registry into
// GenericReconciler — currently pending. Once wired, ReadCrossFromInformer is
// called instead of fetchCrossViaHTTP for same-binary CRDs.
package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// fetchCrossViaHTTP fetches a CR's detail from an Orkestra CR endpoint.
// The endpoint should be the /katalog/{crd}/cr/{namespace}/{name} URL
// which is already built and running on every Orkestra instance.
//
// Returns nil on any error — callers treat nil as "not found".
func fetchCrossViaHTTP(ctx context.Context, endpoint, token string) map[string]interface{} {
	if endpoint == "" {
		return nil
	}

	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp == nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return map[string]interface{}{"found": "false"}
	}
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	// Ensure "found" is set
	if _, ok := result["found"]; !ok {
		if result["name"] != nil {
			result["found"] = "true"
		} else {
			result["found"] = "false"
		}
	}

	return result
}

// ReadCrossFromInformer reads cross-CRD data from an informer cache.
// Zero API server calls — pure in-memory map lookup.
//
// indexer is the GetIndexer() of a SharedIndexInformer for the target CRD.
// key is "namespace/name" for namespaced CRDs or "name" for cluster-scoped.
//
// Returns a map with the same shape as fetchCrossViaHTTP — callers are
// agnostic to which path was used.
func ReadCrossFromInformer(
	indexer interface {
		GetByKey(key string) (interface{}, bool, error)
	},
	key string,
) map[string]interface{} {
	raw, exists, err := indexer.GetByKey(key)
	if err != nil || !exists {
		return map[string]interface{}{
			"found":  "false",
			"status": map[string]interface{}{},
			"spec":   map[string]interface{}{},
		}
	}

	u, ok := raw.(*unstructured.Unstructured)
	if !ok {
		return map[string]interface{}{"found": "false"}
	}

	result := make(map[string]interface{}, 5)
	result["found"] = "true"
	result["name"] = u.GetName()
	result["namespace"] = u.GetNamespace()

	if spec, ok := u.Object["spec"].(map[string]interface{}); ok && spec != nil {
		result["spec"] = spec
	} else {
		result["spec"] = map[string]interface{}{}
	}

	if status, ok := u.Object["status"].(map[string]interface{}); ok && status != nil {
		result["status"] = status
	} else {
		result["status"] = map[string]interface{}{}
	}

	if labels := u.GetLabels(); len(labels) > 0 {
		labelsMap := make(map[string]interface{}, len(labels))
		for k, v := range labels {
			labelsMap[k] = v
		}
		result["labels"] = labelsMap
	}

	return result
}

// crossKey builds the informer cache key for a CR.
// Namespaced: "namespace/name"
// Cluster-scoped: "name"
func crossKey(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return fmt.Sprintf("%s/%s", namespace, name)
}
