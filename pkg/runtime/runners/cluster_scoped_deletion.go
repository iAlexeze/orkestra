// pkg/runners/cluster_scoped_deletion.go
package runners

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orkcrb "github.com/orkspace/orkestra/pkg/resources/clusterrolebindings"
	orkcr "github.com/orkspace/orkestra/pkg/resources/clusterroles"
	orkcust "github.com/orkspace/orkestra/pkg/resources/customresources"
	orkns "github.com/orkspace/orkestra/pkg/resources/namespaces"
	orkpv "github.com/orkspace/orkestra/pkg/resources/pvs"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// DeleteOwnedClusterScopedResources explicitly deletes all cluster-scoped resources declared across
// onCreate, onReconcile, and onDelete that are owned by this CR.
//
// Kubernetes GC does not handle this automatically: owner references from
// namespace-scoped resources (CRs) to cluster-scoped resources (Namespaces, ClusterRoles,
// ClusterRoleBindings, PersistentVolumes) are not honoured by the garbage collector.
// Explicit deletion is required.
//
// This should be called during the deletion path of the reconciler.
func DeleteOwnedClusterScopedResources(
	ctx context.Context,
	kube kubeclient.Interface,
	resolver *orktmpl.Resolver,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
) error {
	// Delete Namespaces
	if err := deleteOwnedNamespaces(ctx, kube, resolver, obj, box); err != nil {
		return err
	}

	// Delete ClusterRoles
	if err := deleteOwnedClusterRoles(ctx, kube, resolver, obj, box); err != nil {
		return err
	}

	// Delete ClusterRoleBindings
	if err := deleteOwnedClusterRoleBindings(ctx, kube, resolver, obj, box); err != nil {
		return err
	}

	// Delete PersistentVolumes
	if err := deleteOwnedPersistentVolumes(ctx, kube, resolver, obj, box); err != nil {
		return err
	}

	// Delete Cluster-scoped Custom Resources
	if err := deleteOwnedCustomResources(ctx, kube, resolver, obj, box); err != nil {
		return err
	}

	return nil
}

// deleteOwnedNamespaces explicitly deletes all Namespaces owned by this CR.
func deleteOwnedNamespaces(
	ctx context.Context,
	kube kubeclient.Interface,
	resolver *orktmpl.Resolver,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
) error {
	var srcs []orktypes.NamespaceTemplateSource
	if box.OnCreate != nil {
		srcs = append(srcs, box.OnCreate.Namespaces...)
	}
	if box.OnReconcile != nil {
		srcs = append(srcs, box.OnReconcile.Namespaces...)
	}
	if box.OnDelete != nil {
		srcs = append(srcs, box.OnDelete.Namespaces...)
	}

	for i, src := range srcs {
		name, err := resolver.Resolve(src.Name)
		if err != nil || name == "" {
			continue
		}
		if err := orkns.DeleteIfOwned(ctx, kube, obj, name); err != nil {
			return fmt.Errorf("namespace[%d] %q: %w", i, name, err)
		}
	}
	return nil
}

// deleteOwnedClusterRoles explicitly deletes all ClusterRoles owned by this CR.
func deleteOwnedClusterRoles(
	ctx context.Context,
	kube kubeclient.Interface,
	resolver *orktmpl.Resolver,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
) error {
	var srcs []orktypes.ClusterRoleTemplateSource
	if box.OnCreate != nil {
		srcs = append(srcs, box.OnCreate.ClusterRoles...)
	}
	if box.OnReconcile != nil {
		srcs = append(srcs, box.OnReconcile.ClusterRoles...)
	}
	if box.OnDelete != nil {
		srcs = append(srcs, box.OnDelete.ClusterRoles...)
	}

	for i, src := range srcs {
		name, err := resolver.Resolve(src.Name)
		if err != nil || name == "" {
			continue
		}
		if err := orkcr.DeleteIfOwned(ctx, kube, obj, name); err != nil {
			return fmt.Errorf("clusterrole[%d] %q: %w", i, name, err)
		}
	}
	return nil
}

// deleteOwnedClusterRoleBindings explicitly deletes all ClusterRoleBindings owned by this CR.
func deleteOwnedClusterRoleBindings(
	ctx context.Context,
	kube kubeclient.Interface,
	resolver *orktmpl.Resolver,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
) error {
	var srcs []orktypes.ClusterRoleBindingTemplateSource
	if box.OnCreate != nil {
		srcs = append(srcs, box.OnCreate.ClusterRoleBindings...)
	}
	if box.OnReconcile != nil {
		srcs = append(srcs, box.OnReconcile.ClusterRoleBindings...)
	}
	if box.OnDelete != nil {
		srcs = append(srcs, box.OnDelete.ClusterRoleBindings...)
	}

	for i, src := range srcs {
		name, err := resolver.Resolve(src.Name)
		if err != nil || name == "" {
			continue
		}
		if err := orkcrb.DeleteIfOwned(ctx, kube, obj, name); err != nil {
			return fmt.Errorf("clusterrolebinding[%d] %q: %w", i, name, err)
		}
	}
	return nil
}

// deleteOwnedPersistentVolumes explicitly deletes all PersistentVolumes owned by this CR.
func deleteOwnedPersistentVolumes(
	ctx context.Context,
	kube kubeclient.Interface,
	resolver *orktmpl.Resolver,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
) error {
	var srcs []orktypes.PVTemplateSource
	if box.OnCreate != nil {
		srcs = append(srcs, box.OnCreate.PersistentVolumes...)
	}
	if box.OnReconcile != nil {
		srcs = append(srcs, box.OnReconcile.PersistentVolumes...)
	}
	if box.OnDelete != nil {
		srcs = append(srcs, box.OnDelete.PersistentVolumes...)
	}

	for i, src := range srcs {
		name, err := resolver.Resolve(src.Name)
		if err != nil || name == "" {
			continue
		}
		if err := orkpv.DeleteIfOwned(ctx, kube, obj, name); err != nil {
			return fmt.Errorf("persistentvolume[%d] %q: %w", i, name, err)
		}
	}
	return nil
}

// deleteOwnedCustomResources explicitly deletes all Cluster-scoped custom resources owned by this CR.
func deleteOwnedCustomResources(
	ctx context.Context,
	kube kubeclient.Interface,
	resolver *orktmpl.Resolver,
	obj domain.Object,
	box orktypes.OperatorBoxConfig,
) error {
	var srcs []orktypes.CustomResourceTemplateSource
	if box.OnCreate != nil {
		srcs = append(srcs, box.OnCreate.CustomResource...)
	}
	if box.OnReconcile != nil {
		srcs = append(srcs, box.OnReconcile.CustomResource...)
	}
	if box.OnDelete != nil {
		srcs = append(srcs, box.OnDelete.CustomResource...)
	}

	for i, src := range srcs {
		// Only process cluster-scoped CRs
		if src.IsNamespaced() {
			continue // GC handles these
		}

		name, err := resolver.Resolve(src.Metadata.Name)
		if err != nil || name == "" {
			continue
		}

		// Cluster-scoped custom resources need explicit cleanup
		if err := orkcust.DeleteIfOwned(ctx, kube, obj, name, "", src.APIVersion, src.Kind); err != nil {
			return fmt.Errorf("customresource[%d] %q: %w", i, name, err)
		}
	}
	return nil
}
