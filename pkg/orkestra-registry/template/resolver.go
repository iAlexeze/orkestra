// pkg/orkestra-registry/template/resolver.go
package template

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/ialexeze/orkestra/domain"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Resolver evaluates Go text/template expressions against a live CR object.
//
// Option B inference — no explicit fromCRD/fromKatalog split required.
// Any field value containing "{{" is treated as a template expression and
// evaluated against the CR. Any value without "{{" is returned as-is.
//
// Template context is the CR's full object as map[string]interface{}:
//
//	"{{ .metadata.name }}"        → CR name
//	"{{ .metadata.namespace }}"   → CR namespace
//	"{{ .spec.image }}"           → spec.image field value
//	"{{ .spec.replicas }}"        → spec.replicas field value
//	"{{ .metadata.name }}-app"    → CR name with a static suffix
//
// For *unstructured.Unstructured (unstructured mode) the full CR map including
// all spec fields is available. For typed objects only metadata fields are
// accessible — typed objects should use Go mode hooks for full spec access.
type Resolver struct {
	data           map[string]interface{}
	ownerName      string
	ownerNamespace string
}

// NewResolver creates a Resolver from any domain.Object.
// For unstructured CRDs the object is *unstructured.Unstructured — full spec accessible.
// For typed CRDs only metadata fields are available in template expressions.
func NewResolver(ctx context.Context, obj domain.Object) (*Resolver, error) {
	data, err := objectToMap(obj)
	if err != nil {
		return nil, fmt.Errorf("template.NewResolver: %w", err)
	}

	return &Resolver{
		data:           data,
		ownerName:      obj.GetName(),
		ownerNamespace: obj.GetNamespace(),
	}, nil
}

// Resolve evaluates a single field value against the CR.
//
// Option B: if value contains "{{" it is a template expression — evaluated
// against the full CR map. Otherwise returned as-is (static value, no cost).
// Missing CR fields resolve to "" (missingkey=zero — no error on absent keys).
func (r *Resolver) Resolve(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	// Fast path — no template markers, static value
	if !strings.Contains(value, "{{") {
		return value, nil
	}

	tmpl, err := template.New("f").Option("missingkey=zero").Parse(value)
	if err != nil {
		return "", fmt.Errorf("parsing %q: %w", value, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r.data); err != nil {
		return "", fmt.Errorf("executing %q: %w", value, err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// ResolveLabels evaluates template expressions in label and annotation values.
// Keys are never template expressions — only values are resolved.
func (r *Resolver) ResolveLabels(labels []orktypes.ResourceLabel) ([]orktypes.ResourceLabel, error) {
	resolved := make([]orktypes.ResourceLabel, 0, len(labels))
	for _, l := range labels {
		v, err := r.Resolve(l.Value)
		if err != nil {
			return nil, fmt.Errorf("label %q: %w", l.Key, err)
		}
		resolved = append(resolved, orktypes.ResourceLabel{Key: l.Key, Value: v})
	}
	return resolved, nil
}

// ResolvePodTemplate resolves all template expressions in a PodTemplateSource.
// Returns a new PodTemplateSource with all expressions evaluated — safe to pass
// directly to pods.Resolve().
//
// Defaults applied when fields are empty after resolution:
//
//	Name      → ownerName + "-pod"      (applied later in pods.Resolve)
//	Namespace → ownerNamespace          (applied here so downstream has it)
func (r *Resolver) ResolvePodTemplate(src orktypes.PodTemplateSource) (orktypes.PodTemplateSource, error) {
	resolved := orktypes.PodTemplateSource{
		Version:   src.Version,
		Resources: src.Resources, // static — not resolved
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("pod.name: %w", err)
	}
	if resolved.Image, err = r.Resolve(src.Image); err != nil {
		return resolved, fmt.Errorf("pod.image: %w", err)
	}
	if resolved.Port, err = r.Resolve(src.Port); err != nil {
		return resolved, fmt.Errorf("pod.port: %w", err)
	}

	// Namespace — default to owner namespace when not declared
	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("pod.namespace: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("pod.labels: %w", err)
	}
	if resolved.Annotations, err = r.ResolveLabels(src.Annotations); err != nil {
		return resolved, fmt.Errorf("pod.annotations: %w", err)
	}

	return resolved, nil
}

// ResolveDeploymentTemplate resolves all template expressions in a DeploymentTemplateSource.
// Returns a new DeploymentTemplateSource with all expressions evaluated — safe to pass
// directly to deployments.Resolve().
func (r *Resolver) ResolveDeploymentTemplate(src orktypes.DeploymentTemplateSource) (orktypes.DeploymentTemplateSource, error) {
	resolved := orktypes.DeploymentTemplateSource{
		Version:   src.Version,
		Resources: src.Resources, // static — not resolved
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("deployment.name: %w", err)
	}
	if resolved.Image, err = r.Resolve(src.Image); err != nil {
		return resolved, fmt.Errorf("deployment.image: %w", err)
	}
	if resolved.Replicas, err = r.Resolve(src.Replicas); err != nil {
		return resolved, fmt.Errorf("deployment.replicas: %w", err)
	}
	if resolved.Port, err = r.Resolve(src.Port); err != nil {
		return resolved, fmt.Errorf("deployment.port: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("deployment.namespace: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("deployment.labels: %w", err)
	}
	if resolved.Annotations, err = r.ResolveLabels(src.Annotations); err != nil {
		return resolved, fmt.Errorf("deployment.annotations: %w", err)
	}

	return resolved, nil
}

// ResolveServiceTemplate resolves all template expressions in a ServiceTemplateSource.
func (r *Resolver) ResolveServiceTemplate(src orktypes.ServiceTemplateSource) (orktypes.ServiceTemplateSource, error) {
	resolved := orktypes.ServiceTemplateSource{
		Version: src.Version,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("service.name: %w", err)
	}
	if resolved.Type, err = r.Resolve(src.Type); err != nil {
		return resolved, fmt.Errorf("service.type: %w", err)
	}
	if resolved.Port, err = r.Resolve(src.Port); err != nil {
		return resolved, fmt.Errorf("service.port: %w", err)
	}
	if resolved.TargetPort, err = r.Resolve(src.TargetPort); err != nil {
		return resolved, fmt.Errorf("service.targetPort: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("service.namespace: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("service.labels: %w", err)
	}

	return resolved, nil
}

// ResolveJobTemplate resolves all template expressions in a JobTemplateSource.
func (r *Resolver) ResolveJobTemplate(src orktypes.JobTemplateSource) (orktypes.JobTemplateSource, error) {
	resolved := orktypes.JobTemplateSource{
		Version:      src.Version,
		BackoffLimit: src.BackoffLimit,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("job.name: %w", err)
	}
	if resolved.Image, err = r.Resolve(src.Image); err != nil {
		return resolved, fmt.Errorf("job.image: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("job.namespace: %w", err)
	}

	// Each command element resolved independently
	for i, c := range src.Command {
		rv, e := r.Resolve(c)
		if e != nil {
			return resolved, fmt.Errorf("job.command[%d]: %w", i, e)
		}
		resolved.Command = append(resolved.Command, rv)
	}
	for i, a := range src.Args {
		rv, e := r.Resolve(a)
		if e != nil {
			return resolved, fmt.Errorf("job.args[%d]: %w", i, e)
		}
		resolved.Args = append(resolved.Args, rv)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("job.labels: %w", err)
	}

	return resolved, nil
}

// OwnerName returns the CR name. Used by registry Resolve() for default naming.
func (r *Resolver) OwnerName() string { return r.ownerName }

// OwnerNamespace returns the CR namespace.
func (r *Resolver) OwnerNamespace() string { return r.ownerNamespace }

// ── Internal ──────────────────────────────────────────────────────────────────

// objectToMap converts a domain.Object to map[string]interface{} for template execution.
//
// Fast path: *unstructured.Unstructured already has the full object map including
// all spec fields — used directly with zero allocation overhead.
//
// Typed objects: only metadata fields are extracted. Spec fields are not accessible
// without reflection or JSON round-trip. Typed object users should use Go mode hooks
// for full spec access rather than YAML template expressions.
func objectToMap(obj domain.Object) (map[string]interface{}, error) {
	// Fast path — unstructured has full map natively
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u.Object, nil
	}

	// Typed fallback — metadata only
	// spec fields not available without reflection on typed objects
	return map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":        obj.GetName(),
			"namespace":   obj.GetNamespace(),
			"labels":      obj.GetLabels(),
			"annotations": obj.GetAnnotations(),
		},
	}, nil
}
