// pkg/children/foreach.go
//
// forEach expansion — expands template sources with a forEach declaration
// into N resolved sources, one per element in the list field.
//
// forEach works on every resource type. The expansion happens before the
// resource-specific run_*.go function is called — each run_*.go function
// receives an already-expanded slice and is unaware of forEach.
//
// Expansion sequence in runTemplateReconcile:
//
//	deployments := ExpandForEachDeployments(resolver, t.Deployments)
//	runDeployments(ctx, kube, resolver, obj, deployments, update)
//
// YAML:
//
//	onReconcile:
//	  deployments:
//	    - name: "{{ .metadata.name }}-{{ .item }}"
//	      image: "{{ .spec.image }}"
//	      forEach:
//	        field: spec.regions
//	        as: item
//
// For CR with spec.regions: ["us-east-1", "eu-west-1"]:
// Produces two DeploymentTemplateSources:
//
//	{Name: "my-app-us-east-1", Image: "nginx:latest", ...}
//	{Name: "my-app-eu-west-1", Image: "nginx:latest", ...}
//
// The expansion resolves ALL template expressions immediately — the
// returned slice contains static (non-template) values ready for the
// registry functions.
//
// when: and anyOf: on forEach sources are evaluated per-item — each
// expanded source may pass or fail conditions independently.
package children

import (
	"sort"

	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ─────────────────────────────────────────────────────────────────────────────
// Namespace expansion
// ─────────────────────────────────────────────────────────────────────────────

func ExpandForEachNamespaces(
	resolver *orktmpl.Resolver,
	srcs []orktypes.NamespaceTemplateSource,
) []orktypes.NamespaceTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.NamespaceTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{
						Key:   l.Key,
						Value: resolvedVal,
					})
				}
			}

			if len(src.Finalizers) > 0 {
				expanded.Finalizers = make([]string, 0, len(src.Finalizers))
				for _, f := range src.Finalizers {
					resolvedVal, _ := ir.Resolve(f)
					expanded.Finalizers = append(expanded.Finalizers, resolvedVal)
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Deployment expansion
// ─────────────────────────────────────────────────────────────────────────────

// ExpandForEachDeployments expands deployments with forEach declarations.
// Sources without forEach are passed through unchanged.
func ExpandForEachDeployments(
	resolver *orktmpl.Resolver,
	srcs []orktypes.DeploymentTemplateSource,
) []orktypes.DeploymentTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs // fast path — no forEach in this list
	}

	var result []orktypes.DeploymentTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil // prevent re-expansion
			// Resolve all template expressions with item in context
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Image, _ = ir.Resolve(src.Image)
			expanded.Replicas, _ = ir.Resolve(src.Replicas)
			expanded.Port, _ = ir.Resolve(src.Port)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)

			// Resolve Env values
			if len(src.Env) > 0 {
				expanded.Env = make(orktypes.EnvVarList, 0, len(src.Env))
				for _, v := range src.Env {
					resolvedVal, _ := ir.Resolve(v.Value)
					expanded.Env = append(expanded.Env, orktypes.EnvVar{Name: v.Name, Value: resolvedVal})
				}
			}

			// Resolve Labels
			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{
						Key:   l.Key,
						Value: resolvedVal,
					})
				}
			}

			// Resolve Annotations
			if len(src.Annotations) > 0 {
				expanded.Annotations = make([]orktypes.ResourceLabel, 0, len(src.Annotations))
				for _, a := range src.Annotations {
					resolvedVal, _ := ir.Resolve(a.Value)
					expanded.Annotations = append(expanded.Annotations, orktypes.ResourceLabel{
						Key:   a.Key,
						Value: resolvedVal,
					})
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// ReplicaSet expansion
// ─────────────────────────────────────────────────────────────────────────────

// ExpandForEachReplicaSets expands replicasets with forEach declarations.
// Sources without forEach are passed through unchanged.
func ExpandForEachReplicaSets(
	resolver *orktmpl.Resolver,
	srcs []orktypes.ReplicaSetTemplateSource,
) []orktypes.ReplicaSetTemplateSource {

	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs // fast path — no forEach in this list
	}

	var result []orktypes.ReplicaSetTemplateSource

	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}

		// Expand forEach items
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)

			expanded := src
			expanded.ForEach = nil // prevent re-expansion

			// Resolve all template expressions with item in context
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Image, _ = ir.Resolve(src.Image)
			expanded.Replicas, _ = ir.Resolve(src.Replicas)
			expanded.Port, _ = ir.Resolve(src.Port)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)

			// Resolve Env values
			if len(src.Env) > 0 {
				expanded.Env = make(orktypes.EnvVarList, 0, len(src.Env))
				for _, v := range src.Env {
					resolvedVal, _ := ir.Resolve(v.Value)
					expanded.Env = append(expanded.Env, orktypes.EnvVar{Name: v.Name, Value: resolvedVal})
				}
			}

			// Resolve Labels
			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{
						Key:   l.Key,
						Value: resolvedVal,
					})
				}
			}

			// Resolve Annotations
			if len(src.Annotations) > 0 {
				expanded.Annotations = make([]orktypes.ResourceLabel, 0, len(src.Annotations))
				for _, a := range src.Annotations {
					resolvedVal, _ := ir.Resolve(a.Value)
					expanded.Annotations = append(expanded.Annotations, orktypes.ResourceLabel{
						Key:   a.Key,
						Value: resolvedVal,
					})
				}
			}

			result = append(result, expanded)
		}
	}

	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Service expansion
// ─────────────────────────────────────────────────────────────────────────────

func ExpandForEachServices(
	resolver *orktmpl.Resolver,
	srcs []orktypes.ServiceTemplateSource,
) []orktypes.ServiceTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.ServiceTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)
			expanded.Port, _ = ir.Resolve(src.Port)
			expanded.TargetPort, _ = ir.Resolve(src.TargetPort)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{
						Key:   l.Key,
						Value: resolvedVal,
					})
				}
			}

			// Resolve selector values with per-item context so {{ .item }} works.
			if len(src.Selector) > 0 {
				expanded.Selector = make(orktypes.SelectorMap, len(src.Selector))
				for k, v := range src.Selector {
					resolvedVal, _ := ir.Resolve(v)
					expanded.Selector[k] = resolvedVal
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Secret expansion
// ─────────────────────────────────────────────────────────────────────────────

func ExpandForEachSecrets(
	resolver *orktmpl.Resolver,
	srcs []orktypes.SecretTemplateSource,
) []orktypes.SecretTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.SecretTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{
						Key:   l.Key,
						Value: resolvedVal,
					})
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// ConfigMap expansion
// ─────────────────────────────────────────────────────────────────────────────

func ExpandForEachConfigMaps(
	resolver *orktmpl.Resolver,
	srcs []orktypes.ConfigMapTemplateSource,
) []orktypes.ConfigMapTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.ConfigMapTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{
						Key:   l.Key,
						Value: resolvedVal,
					})
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Job expansion
// ─────────────────────────────────────────────────────────────────────────────

func ExpandForEachJobs(
	resolver *orktmpl.Resolver,
	srcs []orktypes.JobTemplateSource,
) []orktypes.JobTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.JobTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Image, _ = ir.Resolve(src.Image)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{
						Key:   l.Key,
						Value: resolvedVal,
					})
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// CronJob expansion
// ─────────────────────────────────────────────────────────────────────────────

func ExpandForEachCronJobs(
	resolver *orktmpl.Resolver,
	srcs []orktypes.CronJobTemplateSource,
) []orktypes.CronJobTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.CronJobTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Schedule, _ = ir.Resolve(src.Schedule)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{
						Key:   l.Key,
						Value: resolvedVal,
					})
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Ingress expansion
// ─────────────────────────────────────────────────────────────────────────────

func ExpandForEachIngresses(
	resolver *orktmpl.Resolver,
	srcs []orktypes.IngressTemplateSource,
) []orktypes.IngressTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.IngressTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)
			expanded.Host, _ = ir.Resolve(src.Host)
			expanded.ServiceName, _ = ir.Resolve(src.ServiceName)
			expanded.ServicePort, _ = ir.Resolve(src.ServicePort)
			expanded.Path, _ = ir.Resolve(src.Path)
			expanded.IngressClass, _ = ir.Resolve(src.IngressClass)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{Key: l.Key, Value: resolvedVal})
				}
			}
			if len(src.Annotations) > 0 {
				expanded.Annotations = make([]orktypes.ResourceLabel, 0, len(src.Annotations))
				for _, a := range src.Annotations {
					resolvedVal, _ := ir.Resolve(a.Value)
					expanded.Annotations = append(expanded.Annotations, orktypes.ResourceLabel{Key: a.Key, Value: resolvedVal})
				}
			}

			if src.TLS != nil {
				resolvedTLS := *src.TLS
				resolvedTLS.SecretName, _ = ir.Resolve(src.TLS.SecretName)
				if len(src.TLS.Hosts) > 0 {
					resolvedTLS.Hosts = make([]string, 0, len(src.TLS.Hosts))
					for _, h := range src.TLS.Hosts {
						rv, _ := ir.Resolve(h)
						resolvedTLS.Hosts = append(resolvedTLS.Hosts, rv)
					}
				}
				expanded.TLS = &resolvedTLS
			}

			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// HPA expansion
// ─────────────────────────────────────────────────────────────────────────────

func ExpandForEachHPAs(
	resolver *orktmpl.Resolver,
	srcs []orktypes.HPATemplateSource,
) []orktypes.HPATemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.HPATemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)
			expanded.ScaleTargetRef.APIVersion, _ = ir.Resolve(src.ScaleTargetRef.APIVersion)
			expanded.ScaleTargetRef.Kind, _ = ir.Resolve(src.ScaleTargetRef.Kind)
			expanded.ScaleTargetRef.Name, _ = ir.Resolve(src.ScaleTargetRef.Name)
			expanded.MinReplicas, _ = ir.Resolve(src.MinReplicas)
			expanded.MaxReplicas, _ = ir.Resolve(src.MaxReplicas)
			expanded.TargetCPUUtilizationPercentage, _ = ir.Resolve(src.TargetCPUUtilizationPercentage)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{Key: l.Key, Value: resolvedVal})
				}
			}
			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// PDB expansion
// ─────────────────────────────────────────────────────────────────────────────

func ExpandForEachPDBs(
	resolver *orktmpl.Resolver,
	srcs []orktypes.PDBTemplateSource,
) []orktypes.PDBTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.PDBTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)
			expanded.MinAvailable, _ = ir.Resolve(src.MinAvailable)
			expanded.MaxUnavailable, _ = ir.Resolve(src.MaxUnavailable)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{Key: l.Key, Value: resolvedVal})
				}
			}

			if len(src.Selector) > 0 {
				expanded.Selector = make(orktypes.SelectorMap, len(src.Selector))
				for k, v := range src.Selector {
					resolvedVal, _ := ir.Resolve(v)
					expanded.Selector[k] = resolvedVal
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// ServiceAccount expansion
// ─────────────────────────────────────────────────────────────────────────────

func ExpandForEachServiceAccounts(
	resolver *orktmpl.Resolver,
	srcs []orktypes.ServiceAccountTemplateSource,
) []orktypes.ServiceAccountTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.ServiceAccountTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{
						Key:   l.Key,
						Value: resolvedVal,
					})
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

func ExpandForEachStatefulSets(
	resolver *orktmpl.Resolver,
	srcs []orktypes.StatefulSetTemplateSource,
) []orktypes.StatefulSetTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.StatefulSetTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)
			expanded.Image, _ = ir.Resolve(src.Image)
			expanded.Tag, _ = ir.Resolve(src.Tag)
			expanded.Replicas, _ = ir.Resolve(src.Replicas)
			expanded.Port, _ = ir.Resolve(src.Port)
			expanded.ServiceName, _ = ir.Resolve(src.ServiceName)
			for i, vct := range src.VolumeClaimTemplates {
				rv := src.VolumeClaimTemplates[i]
				rv.StorageClass, _ = ir.Resolve(vct.StorageClass)
				rv.StorageSize, _ = ir.Resolve(vct.StorageSize)
				rv.MountPath, _ = ir.Resolve(vct.MountPath)
				rv.Name, _ = ir.Resolve(vct.Name)
				expanded.VolumeClaimTemplates[i] = rv
			}

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{Key: l.Key, Value: resolvedVal})
				}
			}
			if len(src.Annotations) > 0 {
				expanded.Annotations = make([]orktypes.ResourceLabel, 0, len(src.Annotations))
				for _, a := range src.Annotations {
					resolvedVal, _ := ir.Resolve(a.Value)
					expanded.Annotations = append(expanded.Annotations, orktypes.ResourceLabel{Key: a.Key, Value: resolvedVal})
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

func ExpandForEachPVCs(
	resolver *orktmpl.Resolver,
	srcs []orktypes.PVCTemplateSource,
) []orktypes.PVCTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.PVCTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)
			expanded.StorageClassName, _ = ir.Resolve(src.StorageClassName)
			expanded.Storage, _ = ir.Resolve(src.Storage)
			expanded.VolumeName, _ = ir.Resolve(src.VolumeName)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{Key: l.Key, Value: resolvedVal})
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

func ExpandForEachPVs(
	resolver *orktmpl.Resolver,
	srcs []orktypes.PVTemplateSource,
) []orktypes.PVTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.PVTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.StorageClassName, _ = ir.Resolve(src.StorageClassName)
			expanded.Capacity, _ = ir.Resolve(src.Capacity)
			expanded.ReclaimPolicy, _ = ir.Resolve(src.ReclaimPolicy)
			expanded.HostPath, _ = ir.Resolve(src.HostPath)
			expanded.CSIDriver, _ = ir.Resolve(src.CSIDriver)
			expanded.CSIVolumeHandle, _ = ir.Resolve(src.CSIVolumeHandle)

			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := ir.Resolve(l.Value)
					expanded.Labels = append(expanded.Labels, orktypes.ResourceLabel{Key: l.Key, Value: resolvedVal})
				}
			}

			result = append(result, expanded)
		}
	}
	return result
}

func ExpandForEachRoles(
	resolver *orktmpl.Resolver,
	srcs []orktypes.RoleTemplateSource,
) []orktypes.RoleTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.RoleTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)
			result = append(result, expanded)
		}
	}
	return result
}

func ExpandForEachRoleBindings(
	resolver *orktmpl.Resolver,
	srcs []orktypes.RoleBindingTemplateSource,
) []orktypes.RoleBindingTemplateSource {
	if !anyHasForEach(len(srcs), func(i int) *orktypes.ForEachSpec { return srcs[i].ForEach }) {
		return srcs
	}
	var result []orktypes.RoleBindingTemplateSource
	for _, src := range srcs {
		if src.ForEach == nil {
			result = append(result, src)
			continue
		}
		for i, fi := range resolveForEachItems(resolver.Data(), src.ForEach.Field) {
			ir := itemResolver(resolver, fi, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = ir.Resolve(src.Name)
			expanded.Namespace, _ = ir.Resolve(src.Namespace)
			result = append(result, expanded)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// forEachItem is one iteration step produced by resolveForEachItems.
// For list fields: key = element value, value = nil.
// For map fields:  key = map key (string), value = map value (object or string).
type forEachItem struct {
	key   interface{}
	value interface{}
}

// resolveForEachItems navigates a dot-notation path and returns iteration items.
//
// List field → one item per element; item.value is nil.
//
//	spec.regions: [us-east-1, eu-west-1]
//	→ [{key:"us-east-1"}, {key:"eu-west-1"}]
//
// Map field → one item per key (sorted); item.value is the map value.
//
//	spec.regions: {us-east-1: {replicas: 3}, eu-west-1: {replicas: 1}}
//	→ [{key:"eu-west-1", value:{replicas:1}}, {key:"us-east-1", value:{replicas:3}}]
//
// Template access for map items:
//
//	{{ .item }}            → map key  ("us-east-1")
//	{{ .<as> }}            → same as .item
//	{{ .value.replicas }}  → nested field in the map value
func resolveForEachItems(data map[string]interface{}, path string) []forEachItem {
	var current interface{} = data
	for _, part := range splitFieldPath(path) {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
	}

	if list, ok := current.([]interface{}); ok {
		items := make([]forEachItem, len(list))
		for i, v := range list {
			items[i] = forEachItem{key: v}
		}
		return items
	}

	if m, ok := current.(map[string]interface{}); ok {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		items := make([]forEachItem, len(keys))
		for i, k := range keys {
			items[i] = forEachItem{key: k, value: m[k]}
		}
		return items
	}

	return nil
}

// itemResolver returns an item-scoped resolver for one forEach iteration step.
// For list items (fi.value == nil) only .item is injected.
// For map items (fi.value != nil) both .item and .value are injected.
func itemResolver(base *orktmpl.Resolver, fi forEachItem, as string, index int) *orktmpl.Resolver {
	if fi.value != nil {
		return base.WithItemAndValue(fi.key, fi.value, as, index)
	}
	return base.WithItem(fi.key, as, index)
}

// splitFieldPath splits a dot-notation path into segments.
func splitFieldPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			parts = append(parts, path[start:i])
			start = i + 1
		}
	}
	return append(parts, path[start:])
}

// anyHasForEach returns true if any element has a non-nil ForEach.
// Used as a fast-path check to avoid allocation when no forEach is declared.
func anyHasForEach(n int, getForEach func(int) *orktypes.ForEachSpec) bool {
	for i := 0; i < n; i++ {
		if getForEach(i) != nil {
			return true
		}
	}
	return false
}
