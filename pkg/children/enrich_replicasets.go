package children

import (
	"context"

	"github.com/orkspace/orkestra/pkg/kubeclient"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// enrichGroupWithOwner embeds the controller owner under "_owner" for each
// resource in the group. A no-op when owner enrichment is not enabled on the CRD.
// No API call — data comes from metadata.ownerReferences already in the object.
func enrichGroupWithOwner(_ context.Context, _ kubeclient.KubeClient, m map[string]interface{}, crd orktypes.CRDEntry) {
	if !crd.ShouldEnrich("owner") {
		return
	}

	for _, v := range m {
		obj, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		meta, _ := obj["metadata"].(map[string]interface{})
		if meta == nil {
			continue
		}
		ownerRefs, _ := meta["ownerReferences"].([]interface{})
		for _, ref := range ownerRefs {
			r, _ := ref.(map[string]interface{})
			if r == nil {
				continue
			}
			controller, _ := r["controller"].(bool)
			if !controller {
				continue
			}
			obj["_owner"] = map[string]interface{}{
				"name": r["name"],
				"kind": r["kind"],
				"uid":  r["uid"],
			}
			break
		}
	}
}

// enrichGroupWithReplicaSets lists ReplicaSets owned by each Deployment and
// embeds them as a slice under "_replicaSets". A no-op when replicasets
// enrichment is not enabled on the CRD.
func enrichGroupWithReplicaSets(ctx context.Context, kube kubeclient.KubeClient, m map[string]interface{}, crd orktypes.CRDEntry) {
	if !crd.ShouldEnrich("replicasets") {
		return
	}

	for _, v := range m {
		obj, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		meta, _ := obj["metadata"].(map[string]interface{})
		if meta == nil {
			continue
		}
		ns, _ := meta["namespace"].(string)
		uid, _ := meta["uid"].(string)
		if ns == "" || uid == "" {
			continue
		}
		list, err := kube.DynamicClient().
			Resource(ReplicaSetGVR).
			Namespace(ns).
			List(ctx, metav1.ListOptions{ResourceVersion: "0"})
		if err != nil || list == nil {
			continue
		}
		var rsList []interface{}
		for i := range list.Items {
			if objectOwnedByUID(list.Items[i].Object, uid) {
				rsList = append(rsList, list.Items[i].Object)
			}
		}
		if len(rsList) > 0 {
			obj["_replicaSets"] = rsList
		}
	}
}

// objectOwnedByUID returns true when the object has an ownerReference whose
// uid matches the given uid and controller is true.
func objectOwnedByUID(obj map[string]interface{}, uid string) bool {
	meta, _ := obj["metadata"].(map[string]interface{})
	if meta == nil {
		return false
	}
	ownerRefs, _ := meta["ownerReferences"].([]interface{})
	for _, ref := range ownerRefs {
		r, _ := ref.(map[string]interface{})
		if r == nil {
			continue
		}
		if u, _ := r["uid"].(string); u == uid {
			return true
		}
	}
	return false
}
