// pkg/orkestra-registry/template/resolver.go
package template

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/note"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// Resolver evaluates Go text/template expressions against a live CR object.
//
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
// all spec fields is available.
//
// TYPED:
// For typed objects only metadata fields were accessible — and used Go mode hooks for full spec access.
// Until 'objectToMap' breakthrough
type Resolver struct {
	data           map[string]interface{}
	ownerName      string
	ownerNamespace string
}

// NewResolver creates a Resolver from any domain.Object.
func NewResolver(ctx context.Context, obj domain.Object) (*Resolver, error) {
	data, err := objectToMap(obj) // converts any domain.Object to mapstringinterface{} for template execution.
	if err != nil {
		return nil, fmt.Errorf("template.NewResolver: %w", err)
	}

	// Basic defaults -> workaround for now until full implementation
	data["git"] = map[string]interface{}{
		"called":  "false",
		"commit":  "",
		"changed": "false",
		"path":    "",
		"error":   "",
	}

	data["docker"] = map[string]interface{}{
		"called":         "false",
		"image":          "",
		"buildSucceeded": "false",
		"error":          "",
	}

	// Inject empty inputs map so any unexpanded {{ .inputs.* }} motif expressions
	// that survive motif expansion return "" instead of nil-pointer-panicking.
	if _, ok := data["inputs"]; !ok {
		data["inputs"] = map[string]interface{}{}
	}

	return &Resolver{
		data:           data,
		ownerName:      obj.GetName(),
		ownerNamespace: obj.GetNamespace(),
	}, nil
}

// NewResolverFromMap creates a Resolver from a plain map[string]interface{}.
// Used by conversion webhooks where we work with unstructured JSON.
func NewResolverFromMap(data map[string]interface{}) *Resolver {
	return &Resolver{
		data: data,
		// ownerName/ownerNamespace are optional here; only needed for defaults.
	}
}

// ── Performance review from Engr. Kenny ───────────────────────────────────────────────────
//
// note.Map() is called on every Resolve() call that contains "{{".
// Each call allocates a new FuncMap. For high-throughput operators this
// can be optimised by making the FuncMap a package-level variable:
//
//	var orkNotes = note.Map()
//
// And referencing it in Resolve():
//
//	.Funcs(orkNotes)
//
// This is a safe optimisation — note.Map() is a pure function that always
// returns the same map. The template engine does not modify the FuncMap
// after registration.
var orkNotes = note.Map()

// Resolve evaluates a single field value against the CR.
//
// If value contains "{{" it is a template expression — evaluated
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

	tmpl, err := template.New("f").Option("missingkey=zero").
		Funcs(orkNotes). // ← notes registered here
		Parse(value)
	if err != nil {
		return "", fmt.Errorf("parsing %q: %w", value, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r.data); err != nil {
		return "", fmt.Errorf("executing %q: %w", value, err)
	}

	// missingkey=zero makes missing map keys produce nil (interface{} zero value).
	// Go's text/template renders nil interface{} as "<no value>", not "".
	// Replace all occurrences so callers get "" as documented.
	out := strings.TrimSpace(buf.String())
	out = strings.ReplaceAll(out, "<no value>", "")
	return out, nil
}

// resolveResourceRequirements resolves template expressions in resources.profile.
// All other ResourceRequirements fields (Requests, Limits) are static k8s quantity strings — not templates.
func (r *Resolver) resolveResourceRequirements(src *orktypes.ResourceRequirements) (*orktypes.ResourceRequirements, error) {
	if src == nil {
		return nil, nil
	}
	out := *src
	var err error
	if out.Profile, err = r.Resolve(src.Profile); err != nil {
		return nil, fmt.Errorf("resources.profile: %w", err)
	}
	return &out, nil
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
		Version: src.Version,
		Probes:  src.Probes, // static — passed through
	}

	var err error

	if resolved.Resources, err = r.resolveResourceRequirements(src.Resources); err != nil {
		return resolved, fmt.Errorf("pod.resources: %w", err)
	}
	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("pod.name: %w", err)
	}
	if resolved.Image, err = r.Resolve(src.Image); err != nil {
		return resolved, fmt.Errorf("pod.image: %w", err)
	}
	if resolved.ImagePullSecrets, err = r.ResolveStringSlice(src.ImagePullSecrets); err != nil {
		return resolved, fmt.Errorf("pod.imagePullSecrets: %w", err)
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
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("pod.sleep: %w", err)
	}

	resolved.SecurityContext = src.SecurityContext
	resolved.PodSecurity = src.PodSecurity

	return resolved, nil
}

// ResolveDeploymentTemplate resolves all template expressions in a DeploymentTemplateSource.
// Returns a new DeploymentTemplateSource with all expressions evaluated — safe to pass
// directly to deployments.Resolve().
func (r *Resolver) ResolveDeploymentTemplate(src orktypes.DeploymentTemplateSource) (orktypes.DeploymentTemplateSource, error) {
	resolved := orktypes.DeploymentTemplateSource{
		Version: src.Version,
		Probes:  src.Probes, // static — passed through
	}

	var err error

	if resolved.Resources, err = r.resolveResourceRequirements(src.Resources); err != nil {
		return resolved, fmt.Errorf("deployment.resources: %w", err)
	}
	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("deployment.name: %w", err)
	}
	if resolved.Image, err = r.Resolve(src.Image); err != nil {
		return resolved, fmt.Errorf("deployment.image: %w", err)
	}
	if resolved.ImagePullSecrets, err = r.ResolveStringSlice(src.ImagePullSecrets); err != nil {
		return resolved, fmt.Errorf("deployment.imagePullSecrets: %w", err)
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
	// service account name
	if resolved.ServiceAccountName, err = r.Resolve(src.ServiceAccountName); err != nil {
		return resolved, fmt.Errorf("deployment.serviceAccountName: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("deployment.sleep: %w", err)
	}

	// Env resolution
	if len(src.Env) > 0 {
		resolved.Env = make([]orktypes.EnvVar, 0, len(src.Env))
		for _, v := range src.Env {
			ev := orktypes.EnvVar{Name: v.Name}
			if v.Value != "" {
				if ev.Value, err = r.Resolve(v.Value); err != nil {
					return resolved, fmt.Errorf("env[%s].value: %w", v.Name, err)
				}
			}
			if v.ValueFrom != nil {
				ev.ValueFrom = &orktypes.ValueFrom{}
				if v.ValueFrom.SecretKeyRef != nil {
					name, nerr := r.Resolve(v.ValueFrom.SecretKeyRef.Name)
					if nerr != nil {
						return resolved, fmt.Errorf("env[%s].valueFrom.secretKeyRef.name: %w", v.Name, nerr)
					}
					ev.ValueFrom.SecretKeyRef = &orktypes.SecretKeyRef{Name: name, Key: v.ValueFrom.SecretKeyRef.Key}
				}
				if v.ValueFrom.ConfigMapKeyRef != nil {
					name, nerr := r.Resolve(v.ValueFrom.ConfigMapKeyRef.Name)
					if nerr != nil {
						return resolved, fmt.Errorf("env[%s].valueFrom.configMapKeyRef.name: %w", v.Name, nerr)
					}
					ev.ValueFrom.ConfigMapKeyRef = &orktypes.ConfigMapKeyRef{Name: name, Key: v.ValueFrom.ConfigMapKeyRef.Key}
				}
			}
			resolved.Env = append(resolved.Env, ev)
		}
	}

	// EnvFrom resolution
	if src.EnvFrom != nil {
		resolved.EnvFrom = &orktypes.EnvFrom{}
		for _, name := range src.EnvFrom.SecretRef {
			rn, rerr := r.Resolve(name)
			if rerr != nil {
				return resolved, fmt.Errorf("envFrom.secretRef: %w", rerr)
			}
			resolved.EnvFrom.SecretRef = append(resolved.EnvFrom.SecretRef, rn)
		}
		for _, name := range src.EnvFrom.ConfigMapRef {
			rn, rerr := r.Resolve(name)
			if rerr != nil {
				return resolved, fmt.Errorf("envFrom.configMapRef: %w", rerr)
			}
			resolved.EnvFrom.ConfigMapRef = append(resolved.EnvFrom.ConfigMapRef, rn)
		}
	}

	if src.RollingUpdate != nil {
		ru := *src.RollingUpdate
		resolved.RollingUpdate = &ru
	}

	resolved.SecurityContext = src.SecurityContext
	resolved.PodSecurity = src.PodSecurity

	return resolved, nil
}

// ResolveReplicaSetTemplate resolves all template expressions in a ReplicaSetTemplateSource.
// Returns a new ReplicaSetTemplateSource with all expressions evaluated — safe to pass
// directly to replicasets.Resolve().
func (r *Resolver) ResolveReplicaSetTemplate(src orktypes.ReplicaSetTemplateSource) (orktypes.ReplicaSetTemplateSource, error) {
	resolved := orktypes.ReplicaSetTemplateSource{
		Version: src.Version,
		Probes:  src.Probes, // static — passed through
	}

	var err error

	if resolved.Resources, err = r.resolveResourceRequirements(src.Resources); err != nil {
		return resolved, fmt.Errorf("replicaset.resources: %w", err)
	}

	// Name
	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("replicaset.name: %w", err)
	}

	// Image
	if resolved.Image, err = r.Resolve(src.Image); err != nil {
		return resolved, fmt.Errorf("replicaset.image: %w", err)
	}

	// ImagePullSecrets
	if resolved.ImagePullSecrets, err = r.ResolveStringSlice(src.ImagePullSecrets); err != nil {
		return resolved, fmt.Errorf("replicaset.imagePullSecrets: %w", err)
	}

	// Replicas
	if resolved.Replicas, err = r.Resolve(src.Replicas); err != nil {
		return resolved, fmt.Errorf("replicaset.replicas: %w", err)
	}

	// Port
	if resolved.Port, err = r.Resolve(src.Port); err != nil {
		return resolved, fmt.Errorf("replicaset.port: %w", err)
	}

	// Namespace (default to CR namespace)
	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("replicaset.namespace: %w", err)
	}

	// service account name
	if resolved.ServiceAccountName, err = r.Resolve(src.ServiceAccountName); err != nil {
		return resolved, fmt.Errorf("replicaet.serviceAccountName: %w", err)
	}

	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("replicaset.sleep: %w", err)
	}

	// Labels
	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("replicaset.labels: %w", err)
	}

	// Annotations
	if resolved.Annotations, err = r.ResolveLabels(src.Annotations); err != nil {
		return resolved, fmt.Errorf("replicaset.annotations: %w", err)
	}

	// Env
	if len(src.Env) > 0 {
		resolved.Env = make([]orktypes.EnvVar, 0, len(src.Env))
		for _, v := range src.Env {
			ev := orktypes.EnvVar{Name: v.Name}
			if v.Value != "" {
				if ev.Value, err = r.Resolve(v.Value); err != nil {
					return resolved, fmt.Errorf("env[%s].value: %w", v.Name, err)
				}
			}
			if v.ValueFrom != nil {
				ev.ValueFrom = &orktypes.ValueFrom{}
				if v.ValueFrom.SecretKeyRef != nil {
					name, nerr := r.Resolve(v.ValueFrom.SecretKeyRef.Name)
					if nerr != nil {
						return resolved, fmt.Errorf("env[%s].valueFrom.secretKeyRef.name: %w", v.Name, nerr)
					}
					ev.ValueFrom.SecretKeyRef = &orktypes.SecretKeyRef{Name: name, Key: v.ValueFrom.SecretKeyRef.Key}
				}
				if v.ValueFrom.ConfigMapKeyRef != nil {
					name, nerr := r.Resolve(v.ValueFrom.ConfigMapKeyRef.Name)
					if nerr != nil {
						return resolved, fmt.Errorf("env[%s].valueFrom.configMapKeyRef.name: %w", v.Name, nerr)
					}
					ev.ValueFrom.ConfigMapKeyRef = &orktypes.ConfigMapKeyRef{Name: name, Key: v.ValueFrom.ConfigMapKeyRef.Key}
				}
			}
			resolved.Env = append(resolved.Env, ev)
		}
	}

	// EnvFrom
	if src.EnvFrom != nil {
		resolved.EnvFrom = &orktypes.EnvFrom{}
		for _, name := range src.EnvFrom.SecretRef {
			rn, rerr := r.Resolve(name)
			if rerr != nil {
				return resolved, fmt.Errorf("envFrom.secretRef: %w", rerr)
			}
			resolved.EnvFrom.SecretRef = append(resolved.EnvFrom.SecretRef, rn)
		}
		for _, name := range src.EnvFrom.ConfigMapRef {
			rn, rerr := r.Resolve(name)
			if rerr != nil {
				return resolved, fmt.Errorf("envFrom.configMapRef: %w", rerr)
			}
			resolved.EnvFrom.ConfigMapRef = append(resolved.EnvFrom.ConfigMapRef, rn)
		}
	}

	if src.RollingUpdate != nil {
		ru := *src.RollingUpdate
		resolved.RollingUpdate = &ru
	}

	resolved.SecurityContext = src.SecurityContext
	resolved.PodSecurity = src.PodSecurity

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
	if resolved.Protocol, err = r.Resolve(src.Protocol); err != nil {
		return resolved, fmt.Errorf("service.protocol: %w", err)
	}
	if resolved.Port, err = r.Resolve(src.Port); err != nil {
		return resolved, fmt.Errorf("service.port: %w", err)
	}
	if resolved.TargetPort, err = r.Resolve(src.TargetPort); err != nil {
		return resolved, fmt.Errorf("service.targetPort: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("service.sleep: %w", err)
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

	if resolved.Selector, err = r.ResolveSelectors(src.Selector); err != nil {
		return resolved, fmt.Errorf("service.selector: %w", err)
	}

	return resolved, nil
}

// ResolveNamespaceTemplate resolves all template expressions in a NamespaceTemplateSource.
func (r *Resolver) ResolveNamespaceTemplate(src orktypes.NamespaceTemplateSource) (orktypes.NamespaceTemplateSource, error) {
	resolved := orktypes.NamespaceTemplateSource{
		Version: src.Version,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("namespace.name: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("namespace.labels: %w", err)
	}

	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("namespace.sleep: %w", err)
	}

	for i, a := range src.Finalizers {
		rv, e := r.Resolve(a)
		if e != nil {
			return resolved, fmt.Errorf("namespace.finalizers[%d]: %w", i, e)
		}
		resolved.Finalizers = append(resolved.Finalizers, rv)
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

	if resolved.Resources, err = r.resolveResourceRequirements(src.Resources); err != nil {
		return resolved, fmt.Errorf("job.resources: %w", err)
	}
	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("job.name: %w", err)
	}
	if resolved.Image, err = r.Resolve(src.Image); err != nil {
		return resolved, fmt.Errorf("job.image: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("job.sleep: %w", err)
	}
	if resolved.ImagePullSecrets, err = r.ResolveStringSlice(src.ImagePullSecrets); err != nil {
		return resolved, fmt.Errorf("job.imagePullSecrets: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("job.namespace: %w", err)
	}
	// service account name
	if resolved.ServiceAccountName, err = r.Resolve(src.ServiceAccountName); err != nil {
		return resolved, fmt.Errorf("job.serviceAccountName: %w", err)
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

	resolved.SecurityContext = src.SecurityContext
	resolved.PodSecurity = src.PodSecurity

	return resolved, nil
}

// ResolveSecretTemplate resolves all template expressions in a SecretTemplateSource.
// Returns a new SecretTemplateSource with all expressions evaluated — safe to pass
// directly to secrets.Resolve().
func (r *Resolver) ResolveSecretTemplate(src orktypes.SecretTemplateSource) (orktypes.SecretTemplateSource, error) {
	resolved := orktypes.SecretTemplateSource{
		Version: src.Version,
	}

	var err error

	// Resolve the template expressions
	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("secret.name: %w", err)
	}
	if resolved.FromSecret, err = r.Resolve(src.FromSecret); err != nil {
		return resolved, fmt.Errorf("secret.fromSecret: %w", err)
	}
	if resolved.FromNamespace, err = r.Resolve(src.FromNamespace); err != nil {
		return resolved, fmt.Errorf("secret.fromNamespace: %w", err)
	}
	if resolved.Type, err = r.Resolve(src.Type); err != nil {
		return resolved, fmt.Errorf("secret.type: %w", err)
	}
	if resolved.Namespace, err = r.Resolve(src.Namespace); err != nil {
		return resolved, fmt.Errorf("secret.namespace: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("secret.sleep: %w", err)
	}
	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("secret.data: %w", err)
	}

	if len(src.Data) > 0 {
		resolved.Data = make(map[string]string, len(src.Data))
		for k, v := range src.Data {
			rv, e := r.Resolve(v)
			if e != nil {
				return resolved, fmt.Errorf("secret.data[%q]: %w", k, e)
			}
			resolved.Data[k] = rv
		}
	}

	// toNamespaces needs special handling.
	// Each element is either:
	//   a) A literal namespace name → resolve as string
	//   b) A template expression that resolves to a string → resolve as string
	//   c) A template expression that resolves to a []interface{} (list field) →
	//      extract each element individually
	//
	// Case (c) is what happens when toNamespaces: ["{{ .spec.targetNamespaces }}"]
	// where .spec.targetNamespaces is a YAML list in the CR.

	for i, v := range src.ToNamespaces {
		if !strings.Contains(v, "{{") {
			// Static string — no resolution needed
			if v != "" {
				resolved.ToNamespaces = append(resolved.ToNamespaces, v)
			}
			continue
		}

		// Template expression — check if it resolves to a list field
		raw := resolveRawValue(r.data, v)
		switch typed := raw.(type) {
		case []interface{}:
			// List field — extract each string element
			for _, item := range typed {
				if s, ok := item.(string); ok && s != "" {
					resolved.ToNamespaces = append(resolved.ToNamespaces, s)
				}
			}
		case string:
			if typed != "" {
				resolved.ToNamespaces = append(resolved.ToNamespaces, typed)
			}
		default:
			// Fall back to string resolution
			rv, e := r.Resolve(v)
			if e != nil {
				return resolved, fmt.Errorf("secret.toNamespaces[%d]: %w", i, e)
			}
			if rv != "" {
				resolved.ToNamespaces = append(resolved.ToNamespaces, rv)
			}
		}
	}
	return resolved, nil
}

// ResolveConfigMapTemplate resolves all template expressions in a ConfigMapsTemplateSource.
// Returns a new ConfigMapTemplateSource with all expressions evaluated — safe to pass
// directly to configmaps.Resolve().
func (r *Resolver) ResolveConfigMapTemplate(src orktypes.ConfigMapTemplateSource) (orktypes.ConfigMapTemplateSource, error) {
	resolved := orktypes.ConfigMapTemplateSource{
		Version: src.Version,
	}
	var err error

	// Resolve the template expressions
	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("configmap.name: %w", err)
	}
	if resolved.Namespace, err = r.Resolve(src.Namespace); err != nil {
		return resolved, fmt.Errorf("configmap.namespace: %w", err)
	}
	if resolved.FromNamespace, err = r.Resolve(src.FromNamespace); err != nil {
		return resolved, fmt.Errorf("configmap.fromNamespace: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("configmap.sleep: %w", err)
	}
	if resolved.FromConfigMap, err = r.Resolve(src.FromConfigMap); err != nil {
		return resolved, fmt.Errorf("configmap.fromConfigMap: %w", err)
	}
	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("configmap.labels: %w", err)
	}

	if len(src.Data) > 0 {
		resolved.Data = make(map[string]string, len(src.Data))
		for k, v := range src.Data {
			rv, e := r.Resolve(v)
			if e != nil {
				return resolved, fmt.Errorf("configmap.data[%q]: %w", k, e)
			}
			resolved.Data[k] = rv
		}
	}

	// toNamespaces needs special handling.
	// Each element is either:
	//   a) A literal namespace name → resolve as string
	//   b) A template expression that resolves to a string → resolve as string
	//   c) A template expression that resolves to a []interface{} (list field) →
	//      extract each element individually
	//
	// Case (c) is what happens when toNamespaces: ["{{ .spec.targetNamespaces }}"]
	// where .spec.targetNamespaces is a YAML list in the CR.

	for i, v := range src.ToNamespaces {
		if !strings.Contains(v, "{{") {
			// Static string — no resolution needed
			if v != "" {
				resolved.ToNamespaces = append(resolved.ToNamespaces, v)
			}
			continue
		}

		// Template expression — check if it resolves to a list field
		raw := resolveRawValue(r.data, v)
		switch typed := raw.(type) {
		case []interface{}:
			// List field — extract each string element
			for _, item := range typed {
				if s, ok := item.(string); ok && s != "" {
					resolved.ToNamespaces = append(resolved.ToNamespaces, s)
				}
			}
		case string:
			if typed != "" {
				resolved.ToNamespaces = append(resolved.ToNamespaces, typed)
			}
		default:
			// Fall back to string resolution
			rv, e := r.Resolve(v)
			if e != nil {
				return resolved, fmt.Errorf("secret.toNamespaces[%d]: %w", i, e)
			}
			if rv != "" {
				resolved.ToNamespaces = append(resolved.ToNamespaces, rv)
			}
		}
	}

	return resolved, nil
}

// ResolveCronJobTemplate resolves all template expressions in a CronJobTemplateSource.
func (r *Resolver) ResolveCronJobTemplate(src orktypes.CronJobTemplateSource) (orktypes.CronJobTemplateSource, error) {
	resolved := orktypes.CronJobTemplateSource{
		Version: src.Version,
	}

	var err error

	if resolved.Resources, err = r.resolveResourceRequirements(src.Resources); err != nil {
		return resolved, fmt.Errorf("cronjob.resources: %w", err)
	}
	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("cronjob.name: %w", err)
	}
	if resolved.Image, err = r.Resolve(src.Image); err != nil {
		return resolved, fmt.Errorf("cronjob.image: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("cronjob.sleep: %w", err)
	}
	if resolved.ImagePullSecrets, err = r.ResolveStringSlice(src.ImagePullSecrets); err != nil {
		return resolved, fmt.Errorf("cronjob.imagePullSecrets: %w", err)
	}
	if resolved.Schedule, err = r.Resolve(src.Schedule); err != nil {
		return resolved, fmt.Errorf("cronjob.schedule: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("cronjob.namespace: %w", err)
	}
	// service account name
	if resolved.ServiceAccountName, err = r.Resolve(src.ServiceAccountName); err != nil {
		return resolved, fmt.Errorf("cronjob.serviceAccountName: %w", err)
	}

	for i, c := range src.Command {
		rv, e := r.Resolve(c)
		if e != nil {
			return resolved, fmt.Errorf("cronjob.command[%d]: %w", i, e)
		}
		resolved.Command = append(resolved.Command, rv)
	}
	for i, a := range src.Args {
		rv, e := r.Resolve(a)
		if e != nil {
			return resolved, fmt.Errorf("cronjob.args[%d]: %w", i, e)
		}
		resolved.Args = append(resolved.Args, rv)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("cronjob.labels: %w", err)
	}

	resolved.SecurityContext = src.SecurityContext
	resolved.PodSecurity = src.PodSecurity

	return resolved, nil
}

// ResolveServiceAccountTemplate resolves all template expressions in a ServiceAccountTemplateSource.
func (r *Resolver) ResolveServiceAccountTemplate(src orktypes.ServiceAccountTemplateSource) (orktypes.ServiceAccountTemplateSource, error) {
	resolved := orktypes.ServiceAccountTemplateSource{
		Version: src.Version,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("serviceaccount.name: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("serviceaccount.namespace: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("serviceAccount.sleep: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("serviceaccount.labels: %w", err)
	}

	return resolved, nil
}

// ResolveRoleTemplate resolves all template expressions in a RoleTemplateSource.
func (r *Resolver) ResolveRoleTemplate(src orktypes.RoleTemplateSource) (orktypes.RoleTemplateSource, error) {
	resolved := orktypes.RoleTemplateSource{
		Version:   src.Version,
		Reconcile: src.Reconcile,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("role.name: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("role.namespace: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("role.sleep: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("role.labels: %w", err)
	}

	// Resolve resourceNames in each rule (apiGroups/resources/verbs are typically static)
	for _, rule := range src.Rules {
		resolvedRule := orktypes.PolicyRuleSpec{
			APIGroups: rule.APIGroups,
			Resources: rule.Resources,
			Verbs:     rule.Verbs,
		}
		for _, rn := range rule.ResourceNames {
			rv, resolveErr := r.Resolve(rn)
			if resolveErr != nil {
				return resolved, fmt.Errorf("role.rules.resourceNames: %w", resolveErr)
			}
			resolvedRule.ResourceNames = append(resolvedRule.ResourceNames, rv)
		}
		resolved.Rules = append(resolved.Rules, resolvedRule)
	}

	return resolved, nil
}

// ResolveRoleBindingTemplate resolves all template expressions in a RoleBindingTemplateSource.
func (r *Resolver) ResolveRoleBindingTemplate(src orktypes.RoleBindingTemplateSource) (orktypes.RoleBindingTemplateSource, error) {
	resolved := orktypes.RoleBindingTemplateSource{
		Version:   src.Version,
		Reconcile: src.Reconcile,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("rolebinding.name: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("rolebinding.namespace: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("rolebinding.sleep: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("rolebinding.labels: %w", err)
	}

	resolved.RoleRef.Kind = src.RoleRef.Kind
	if resolved.RoleRef.Name, err = r.Resolve(src.RoleRef.Name); err != nil {
		return resolved, fmt.Errorf("rolebinding.roleRef.name: %w", err)
	}

	for i, s := range src.Subjects {
		rs := orktypes.SubjectSpec{Kind: s.Kind}
		if rs.Name, err = r.Resolve(s.Name); err != nil {
			return resolved, fmt.Errorf("rolebinding.subjects[%d].name: %w", i, err)
		}
		if rs.Namespace, err = r.Resolve(s.Namespace); err != nil {
			return resolved, fmt.Errorf("rolebinding.subjects[%d].namespace: %w", i, err)
		}
		resolved.Subjects = append(resolved.Subjects, rs)
	}

	return resolved, nil
}

// ResolveIngressTemplate resolves all template expressions in an IngressTemplateSource.
func (r *Resolver) ResolveIngressTemplate(src orktypes.IngressTemplateSource) (orktypes.IngressTemplateSource, error) {
	resolved := orktypes.IngressTemplateSource{
		Version: src.Version,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("ingress.name: %w", err)
	}
	if resolved.Host, err = r.Resolve(src.Host); err != nil {
		return resolved, fmt.Errorf("ingress.host: %w", err)
	}
	if resolved.ServiceName, err = r.Resolve(src.ServiceName); err != nil {
		return resolved, fmt.Errorf("ingress.serviceName: %w", err)
	}
	if resolved.ServicePort, err = r.Resolve(src.ServicePort); err != nil {
		return resolved, fmt.Errorf("ingress.servicePort: %w", err)
	}
	if resolved.Path, err = r.Resolve(src.Path); err != nil {
		return resolved, fmt.Errorf("ingress.path: %w", err)
	}
	if resolved.PathType, err = r.Resolve(src.PathType); err != nil {
		return resolved, fmt.Errorf("ingress.pathType: %w", err)
	}
	if resolved.IngressClass, err = r.Resolve(src.IngressClass); err != nil {
		return resolved, fmt.Errorf("ingress.ingressClass: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("ingress.sleep: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("ingress.namespace: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("ingress.labels: %w", err)
	}
	if resolved.Annotations, err = r.ResolveLabels(src.Annotations); err != nil {
		return resolved, fmt.Errorf("ingress.annotations: %w", err)
	}

	if src.TLS != nil {
		resolvedTLS := &orktypes.IngressTLSSpec{
			Create:   src.TLS.Create,
			ValidFor: src.TLS.ValidFor,
		}
		if resolvedTLS.SecretName, err = r.Resolve(src.TLS.SecretName); err != nil {
			return resolved, fmt.Errorf("ingress.tls.secretName: %w", err)
		}
		for i, h := range src.TLS.Hosts {
			rv, e := r.Resolve(h)
			if e != nil {
				return resolved, fmt.Errorf("ingress.tls.hosts[%d]: %w", i, e)
			}
			resolvedTLS.Hosts = append(resolvedTLS.Hosts, rv)
		}
		resolved.TLS = resolvedTLS
	}

	return resolved, nil
}

// ResolveHPATemplate resolves all template expressions in an HPATemplateSource.
func (r *Resolver) ResolveHPATemplate(src orktypes.HPATemplateSource) (orktypes.HPATemplateSource, error) {
	resolved := orktypes.HPATemplateSource{
		Version: src.Version,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("hpa.name: %w", err)
	}
	if resolved.ScaleTargetRef.APIVersion, err = r.Resolve(src.ScaleTargetRef.APIVersion); err != nil {
		return resolved, fmt.Errorf("hpa.scaledRef.apiVersion: %w", err)
	}
	if resolved.ScaleTargetRef.Kind, err = r.Resolve(src.ScaleTargetRef.Kind); err != nil {
		return resolved, fmt.Errorf("hpa.scaledRef.kind: %w", err)
	}
	if resolved.ScaleTargetRef.Name, err = r.Resolve(src.ScaleTargetRef.Name); err != nil {
		return resolved, fmt.Errorf("hpa.scaledRef.name: %w", err)
	}
	if resolved.MinReplicas, err = r.Resolve(src.MinReplicas); err != nil {
		return resolved, fmt.Errorf("hpa.minReplicas: %w", err)
	}
	if resolved.MaxReplicas, err = r.Resolve(src.MaxReplicas); err != nil {
		return resolved, fmt.Errorf("hpa.maxReplicas: %w", err)
	}
	if resolved.TargetCPUUtilizationPercentage, err = r.Resolve(src.TargetCPUUtilizationPercentage); err != nil {
		return resolved, fmt.Errorf("hpa.targetCPUUtilizationPercentage: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("hpa.sleep: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("hpa.namespace: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("hpa.labels: %w", err)
	}

	if src.Behavior != nil {
		b := *src.Behavior
		resolved.Behavior = &b
	}

	return resolved, nil
}

// ResolvePDBTemplate resolves all template expressions in a PDBTemplateSource.
func (r *Resolver) ResolvePDBTemplate(src orktypes.PDBTemplateSource) (orktypes.PDBTemplateSource, error) {
	resolved := orktypes.PDBTemplateSource{
		Version: src.Version,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("pdb.name: %w", err)
	}
	if resolved.MinAvailable, err = r.Resolve(src.MinAvailable); err != nil {
		return resolved, fmt.Errorf("pdb.minAvailable: %w", err)
	}
	if resolved.MaxUnavailable, err = r.Resolve(src.MaxUnavailable); err != nil {
		return resolved, fmt.Errorf("pdb.maxUnavailable: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("pdb.namespace: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("pdb.labels: %w", err)
	}

	if resolved.Selector, err = r.ResolveSelectors(src.Selector); err != nil {
		return resolved, fmt.Errorf("pdb.selector: %w", err)
	}

	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("pdb.sleep: %w", err)
	}

	if src.Behavior != nil {
		b := *src.Behavior
		resolved.Behavior = &b
	}

	return resolved, nil
}

// ResolveStatefulSetTemplate resolves all template expressions in a StatefulSetTemplateSource.
func (r *Resolver) ResolveStatefulSetTemplate(src orktypes.StatefulSetTemplateSource) (orktypes.StatefulSetTemplateSource, error) {
	resolved := orktypes.StatefulSetTemplateSource{
		Version: src.Version,
		Probes:  src.Probes, // static — passed through
	}

	var err error

	if resolved.Resources, err = r.resolveResourceRequirements(src.Resources); err != nil {
		return resolved, fmt.Errorf("statefulset.resources: %w", err)
	}
	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("statefulset.name: %w", err)
	}
	if resolved.Image, err = r.Resolve(src.Image); err != nil {
		return resolved, fmt.Errorf("statefulset.image: %w", err)
	}
	if resolved.ImagePullSecrets, err = r.ResolveStringSlice(src.ImagePullSecrets); err != nil {
		return resolved, fmt.Errorf("statefulset.imagePullSecrets: %w", err)
	}
	if resolved.Tag, err = r.Resolve(src.Tag); err != nil {
		return resolved, fmt.Errorf("statefulset.tag: %w", err)
	}
	if resolved.Replicas, err = r.Resolve(src.Replicas); err != nil {
		return resolved, fmt.Errorf("statefulset.replicas: %w", err)
	}
	if resolved.Port, err = r.Resolve(src.Port); err != nil {
		return resolved, fmt.Errorf("statefulset.port: %w", err)
	}
	if resolved.ServiceName, err = r.Resolve(src.ServiceName); err != nil {
		return resolved, fmt.Errorf("statefulset.serviceName: %w", err)
	}
	for i, vct := range src.VolumeClaimTemplates {
		rv := orktypes.VolumeClaimTemplateSource{AccessModes: vct.AccessModes}
		if rv.Name, err = r.Resolve(vct.Name); err != nil {
			return resolved, fmt.Errorf("statefulset.volumeClaimTemplates[%d].name: %w", i, err)
		}
		if rv.StorageClass, err = r.Resolve(vct.StorageClass); err != nil {
			return resolved, fmt.Errorf("statefulset.volumeClaimTemplates[%d].storageClass: %w", i, err)
		}
		if rv.StorageSize, err = r.Resolve(vct.StorageSize); err != nil {
			return resolved, fmt.Errorf("statefulset.volumeClaimTemplates[%d].storageSize: %w", i, err)
		}
		if rv.MountPath, err = r.Resolve(vct.MountPath); err != nil {
			return resolved, fmt.Errorf("statefulset.volumeClaimTemplates[%d].mountPath: %w", i, err)
		}
		resolved.VolumeClaimTemplates = append(resolved.VolumeClaimTemplates, rv)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("statefulset.sleep: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("statefulset.namespace: %w", err)
	}
	// service account name
	if resolved.ServiceAccountName, err = r.Resolve(src.ServiceAccountName); err != nil {
		return resolved, fmt.Errorf("statefulset.serviceAccountName: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("statefulset.labels: %w", err)
	}
	if resolved.Annotations, err = r.ResolveLabels(src.Annotations); err != nil {
		return resolved, fmt.Errorf("statefulset.annotations: %w", err)
	}

	// Env resolution
	if len(src.Env) > 0 {
		resolved.Env = make([]orktypes.EnvVar, 0, len(src.Env))
		for _, v := range src.Env {
			ev := orktypes.EnvVar{Name: v.Name}
			if v.Value != "" {
				if ev.Value, err = r.Resolve(v.Value); err != nil {
					return resolved, fmt.Errorf("env[%s].value: %w", v.Name, err)
				}
			}
			if v.ValueFrom != nil {
				ev.ValueFrom = &orktypes.ValueFrom{}
				if v.ValueFrom.SecretKeyRef != nil {
					name, nerr := r.Resolve(v.ValueFrom.SecretKeyRef.Name)
					if nerr != nil {
						return resolved, fmt.Errorf("env[%s].valueFrom.secretKeyRef.name: %w", v.Name, nerr)
					}
					ev.ValueFrom.SecretKeyRef = &orktypes.SecretKeyRef{Name: name, Key: v.ValueFrom.SecretKeyRef.Key}
				}
				if v.ValueFrom.ConfigMapKeyRef != nil {
					name, nerr := r.Resolve(v.ValueFrom.ConfigMapKeyRef.Name)
					if nerr != nil {
						return resolved, fmt.Errorf("env[%s].valueFrom.configMapKeyRef.name: %w", v.Name, nerr)
					}
					ev.ValueFrom.ConfigMapKeyRef = &orktypes.ConfigMapKeyRef{Name: name, Key: v.ValueFrom.ConfigMapKeyRef.Key}
				}
			}
			resolved.Env = append(resolved.Env, ev)
		}
	}

	// EnvFrom resolution
	if src.EnvFrom != nil {
		resolved.EnvFrom = &orktypes.EnvFrom{}
		for _, name := range src.EnvFrom.SecretRef {
			rn, rerr := r.Resolve(name)
			if rerr != nil {
				return resolved, fmt.Errorf("envFrom.secretRef: %w", rerr)
			}
			resolved.EnvFrom.SecretRef = append(resolved.EnvFrom.SecretRef, rn)
		}
		for _, name := range src.EnvFrom.ConfigMapRef {
			rn, rerr := r.Resolve(name)
			if rerr != nil {
				return resolved, fmt.Errorf("envFrom.configMapRef: %w", rerr)
			}
			resolved.EnvFrom.ConfigMapRef = append(resolved.EnvFrom.ConfigMapRef, rn)
		}
	}

	if src.RollingUpdate != nil {
		ru := *src.RollingUpdate
		resolved.RollingUpdate = &ru
	}

	resolved.SecurityContext = src.SecurityContext
	resolved.PodSecurity = src.PodSecurity

	return resolved, nil
}

// ResolvePVCTemplate resolves all template expressions in a PVCTemplateSource.
func (r *Resolver) ResolvePVCTemplate(src orktypes.PVCTemplateSource) (orktypes.PVCTemplateSource, error) {
	resolved := orktypes.PVCTemplateSource{
		Version:     src.Version,
		AccessModes: src.AccessModes,
		VolumeMode:  src.VolumeMode,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("pvc.name: %w", err)
	}
	if resolved.StorageClassName, err = r.Resolve(src.StorageClassName); err != nil {
		return resolved, fmt.Errorf("pvc.storageClassName: %w", err)
	}
	if resolved.Storage, err = r.Resolve(src.Storage); err != nil {
		return resolved, fmt.Errorf("pvc.storage: %w", err)
	}
	if resolved.VolumeName, err = r.Resolve(src.VolumeName); err != nil {
		return resolved, fmt.Errorf("pvc.volumeName: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("pvc.sleep: %w", err)
	}

	ns := src.Namespace
	if ns == "" {
		ns = "{{ .metadata.namespace }}"
	}
	if resolved.Namespace, err = r.Resolve(ns); err != nil {
		return resolved, fmt.Errorf("pvc.namespace: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("pvc.labels: %w", err)
	}

	return resolved, nil
}

// ResolvePVTemplate resolves all template expressions in a PVTemplateSource.
func (r *Resolver) ResolvePVTemplate(src orktypes.PVTemplateSource) (orktypes.PVTemplateSource, error) {
	resolved := orktypes.PVTemplateSource{
		Version:     src.Version,
		AccessModes: src.AccessModes,
	}

	var err error

	if resolved.Name, err = r.Resolve(src.Name); err != nil {
		return resolved, fmt.Errorf("pv.name: %w", err)
	}
	if resolved.StorageClassName, err = r.Resolve(src.StorageClassName); err != nil {
		return resolved, fmt.Errorf("pv.storageClassName: %w", err)
	}
	if resolved.Capacity, err = r.Resolve(src.Capacity); err != nil {
		return resolved, fmt.Errorf("pv.capacity: %w", err)
	}
	if resolved.ReclaimPolicy, err = r.Resolve(src.ReclaimPolicy); err != nil {
		return resolved, fmt.Errorf("pv.reclaimPolicy: %w", err)
	}
	if resolved.HostPath, err = r.Resolve(src.HostPath); err != nil {
		return resolved, fmt.Errorf("pv.hostPath: %w", err)
	}
	if resolved.CSIDriver, err = r.Resolve(src.CSIDriver); err != nil {
		return resolved, fmt.Errorf("pv.csiDriver: %w", err)
	}
	if resolved.CSIVolumeHandle, err = r.Resolve(src.CSIVolumeHandle); err != nil {
		return resolved, fmt.Errorf("pv.csiVolumeHandle: %w", err)
	}
	if resolved.Sleep, err = r.Resolve(src.Sleep); err != nil {
		return resolved, fmt.Errorf("pv.sleep: %w", err)
	}

	if resolved.Labels, err = r.ResolveLabels(src.Labels); err != nil {
		return resolved, fmt.Errorf("pv.labels: %w", err)
	}

	return resolved, nil
}
