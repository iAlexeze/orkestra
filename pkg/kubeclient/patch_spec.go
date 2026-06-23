package kubeclient

import (
	"context"
	"encoding/json"

	"github.com/orkspace/orkestra/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// pkg/kubeclient/patch_spec.go
func (k *Kubeclient) PatchSpec(
	ctx context.Context,
	obj domain.Object,
	specFields map[string]interface{},
) error {
	mapping, err := k.gvrFor(obj)
	if err != nil {
		return err
	}

	patch := map[string]interface{}{"spec": specFields}
	patchBytes, _ := json.Marshal(patch)

	_, err = k.DynamicClient().
		Resource(mapping.Resource).
		Namespace(obj.GetNamespace()).
		Patch(ctx, obj.GetName(), types.MergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}
