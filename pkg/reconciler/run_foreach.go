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
		items := resolveListField(resolver.Data(), src.ForEach.Field)
		for i, item := range items {
			itemResolver := resolver.WithItem(item, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil // prevent re-expansion
			// Resolve all template expressions with item in context
			expanded.Name, _ = itemResolver.Resolve(src.Name)
			expanded.Image, _ = itemResolver.Resolve(src.Image)
			expanded.Replicas, _ = itemResolver.Resolve(src.Replicas)
			expanded.Namespace, _ = itemResolver.Resolve(src.Namespace)

			// Resolve Env map values
			if len(src.Env) > 0 {
				expanded.Env = make(map[string]orktypes.EnvVarSource, len(src.Env))
				for k, v := range src.Env {
					resolvedVal, _ := itemResolver.Resolve(v.Value)
					expanded.Env[k] = orktypes.EnvVarSource(orktypes.EnvVarSource{Value: resolvedVal})
				}
			}

			// Resolve Labels
			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := itemResolver.Resolve(l.Value)
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
					resolvedVal, _ := itemResolver.Resolve(a.Value)
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
		for i, item := range resolveListField(resolver.Data(), src.ForEach.Field) {
			itemResolver := resolver.WithItem(item, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = itemResolver.Resolve(src.Name)
			expanded.Namespace, _ = itemResolver.Resolve(src.Namespace)
			expanded.Port, _ = itemResolver.Resolve(src.Port)

			// Resolve labels
			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := itemResolver.Resolve(l.Value)
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
		for i, item := range resolveListField(resolver.Data(), src.ForEach.Field) {
			itemResolver := resolver.WithItem(item, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = itemResolver.Resolve(src.Name)
			expanded.Namespace, _ = itemResolver.Resolve(src.Namespace)

			// Resolve labels
			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := itemResolver.Resolve(l.Value)
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
		for i, item := range resolveListField(resolver.Data(), src.ForEach.Field) {
			itemResolver := resolver.WithItem(item, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = itemResolver.Resolve(src.Name)
			expanded.Namespace, _ = itemResolver.Resolve(src.Namespace)

			// Resolve labels
			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := itemResolver.Resolve(l.Value)
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
		for i, item := range resolveListField(resolver.Data(), src.ForEach.Field) {
			itemResolver := resolver.WithItem(item, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = itemResolver.Resolve(src.Name)
			expanded.Image, _ = itemResolver.Resolve(src.Image)
			expanded.Namespace, _ = itemResolver.Resolve(src.Namespace)

			// Resolve labels
			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := itemResolver.Resolve(l.Value)
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
		for i, item := range resolveListField(resolver.Data(), src.ForEach.Field) {
			itemResolver := resolver.WithItem(item, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = itemResolver.Resolve(src.Name)
			expanded.Schedule, _ = itemResolver.Resolve(src.Schedule)
			expanded.Namespace, _ = itemResolver.Resolve(src.Namespace)

			// Resolve labels
			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := itemResolver.Resolve(l.Value)
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
		for i, item := range resolveListField(resolver.Data(), src.ForEach.Field) {
			itemResolver := resolver.WithItem(item, src.ForEach.As, i)
			expanded := src
			expanded.ForEach = nil
			expanded.Name, _ = itemResolver.Resolve(src.Name)
			expanded.Namespace, _ = itemResolver.Resolve(src.Namespace)

			// Resolve labels
			if len(src.Labels) > 0 {
				expanded.Labels = make([]orktypes.ResourceLabel, 0, len(src.Labels))
				for _, l := range src.Labels {
					resolvedVal, _ := itemResolver.Resolve(l.Value)
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

// resolveListField navigates a dot-notation path and returns the list value.
// Returns nil when the field is absent or not a list.
func resolveListField(data map[string]interface{}, path string) []interface{} {
	val := orktypes.NavigateDotPath(data, path)
	_ = val // NavigateDotPath returns string — need raw value for lists

	// Navigate the path to get the raw interface{} value
	var current interface{} = data
	for _, part := range splitFieldPath(path) {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}

	if list, ok := current.([]interface{}); ok {
		return list
	}
	return nil
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
