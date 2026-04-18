// pkg/reconciler/run_foreach.go
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
//	deployments := expandForEachDeployments(resolver, t.Deployments)
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
package reconciler

import (
	"sort"

	orktmpl "github.com/orkspace/orkestra/pkg/orkestra-registry/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ─────────────────────────────────────────────────────────────────────────────
// Deployment expansion
// ─────────────────────────────────────────────────────────────────────────────

// expandForEachDeployments expands deployments with forEach declarations.
// Sources without forEach are passed through unchanged.
func expandForEachDeployments(
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

			// Resolve Env map values
			if len(src.Env) > 0 {
				expanded.Env = make(map[string]orktypes.EnvVarSource, len(src.Env))
				for k, v := range src.Env {
					resolvedVal, _ := ir.Resolve(v.Value)
					expanded.Env[k] = orktypes.EnvVarSource{Value: resolvedVal}
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

func expandForEachServices(
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

func expandForEachSecrets(
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

func expandForEachConfigMaps(
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

func expandForEachJobs(
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

func expandForEachCronJobs(
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
// ServiceAccount expansion
// ─────────────────────────────────────────────────────────────────────────────

func expandForEachServiceAccounts(
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

// resolveListField is kept for backward compatibility with any callers outside this file.
func resolveListField(data map[string]interface{}, path string) []interface{} {
	items := resolveForEachItems(data, path)
	result := make([]interface{}, len(items))
	for i, fi := range items {
		result[i] = fi.key
	}
	return result
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
