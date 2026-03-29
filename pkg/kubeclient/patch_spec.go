package kubeclient

import (
	"context"
	"encoding/json"

	"github.com/ialexeze/orkestra/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// pkg/kubeclient/patch_spec.go
func (k *Kubeclient) PatchSpec(
    ctx context.Context,
    obj domain.Object,
    gvr schema.GroupVersionResource,
    specFields map[string]interface{},
) error {
    patch := map[string]interface{}{"spec": specFields}
    patchBytes, _ := json.Marshal(patch)

    _, err := k.DynamicClient().
        Resource(gvr).
        Namespace(obj.GetNamespace()).
        Patch(ctx, obj.GetName(), types.MergePatchType, patchBytes, metav1.PatchOptions{})
    return err
}