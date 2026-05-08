// pkg/motif/expander.go
//
// Expander instantiates a Motif by binding its inputs and expanding
// its resource blocks into a concrete HookTemplates value.
//
// Two modes:
//
// Static (ork doctor init): inputs resolved from explicit with: bindings
// at generation time. The expanded resources are inlined into the generated
// Katalog. No runtime dependency on the Motif.
//
// Dynamic (Katalog runtime): inputs resolved at Katalog startup, before
// any reconcile. The Motif is loaded once and its templates compiled with
// the input bindings.
package motif

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"gopkg.in/yaml.v3"
)

// ExpandedMotif holds the result of expanding a motif.
type ExpandedMotif struct {
	Resources *orktypes.HookTemplates
	Status    *orktypes.StatusConfig
	Admission *orktypes.Admission
}

// HasResources returns true when the motif produced resource templates.
func (e *ExpandedMotif) HasResources() bool {
	return e.Resources != nil
}

// HasStatus reports whether the motif defines status fields or conditions.
func (e *ExpandedMotif) HasStatus() bool {
	return e.Status != nil
}

// HasAdmission reports whether the motif includes admission rules.
func (e *ExpandedMotif) HasAdmission() bool {
	return e.Admission != nil
}

// Expand instantiates a Motif with the given input bindings and returns
// the expanded resources, status, and admission configuration.
//
// bindings maps input name → resolved value. Required inputs missing from
// bindings are a validation error. Unknown inputs in bindings are also an error.
// Optional inputs not in bindings use their Motif-declared defaults.
//
// Expand replaces all `{{ .inputs.Name }}` and `{{ inputs.Name }}` expressions
// in the YAML of resources, status, and admission with the resolved binding values.
// Other template expressions (e.g., `{{ .children.* }}`) are left untouched
// and will be evaluated at runtime by the reconciler.
func Expand(m *orktypes.Motif, bindings map[string]string) (*ExpandedMotif, error) {
	if err := validateBindings(m, bindings); err != nil {
		return nil, err
	}

	resolved := resolveDefaults(m, bindings)

	// ---- Expand resources ----
	var resources *orktypes.HookTemplates
	if m.Resources != nil {
		resourceYAML, err := yaml.Marshal(m.Resources)
		if err != nil {
			return nil, fmt.Errorf("marshaling motif resources: %w", err)
		}
		rendered, err := renderInputs(string(resourceYAML), resolved)
		if err != nil {
			return nil, fmt.Errorf("rendering motif %q resources: %w", m.Metadata.Name, err)
		}
		var hookTemplates orktypes.HookTemplates
		if err := yaml.Unmarshal([]byte(rendered), &hookTemplates); err != nil {
			return nil, fmt.Errorf("parsing expanded motif %q resources: %w", m.Metadata.Name, err)
		}
		resources = &hookTemplates
	}

	// ---- Expand status ----
	var status *orktypes.StatusConfig
	if m.Status != nil {
		statusYAML, err := yaml.Marshal(m.Status)
		if err != nil {
			return nil, fmt.Errorf("marshaling motif status: %w", err)
		}
		rendered, err := renderInputs(string(statusYAML), resolved)
		if err != nil {
			return nil, fmt.Errorf("rendering motif %q status: %w", m.Metadata.Name, err)
		}
		var statusConfig orktypes.StatusConfig
		if err := yaml.Unmarshal([]byte(rendered), &statusConfig); err != nil {
			return nil, fmt.Errorf("parsing expanded motif %q status: %w", m.Metadata.Name, err)
		}
		status = &statusConfig
	}

	// ---- Expand admission (validation + mutation) ----
	var admission *orktypes.Admission
	if m.Admission != nil {
		admissionYAML, err := yaml.Marshal(m.Admission)
		if err != nil {
			return nil, fmt.Errorf("marshaling motif admission: %w", err)
		}
		rendered, err := renderInputs(string(admissionYAML), resolved)
		if err != nil {
			return nil, fmt.Errorf("rendering motif %q admission: %w", m.Metadata.Name, err)
		}
		var adm orktypes.Admission
		if err := yaml.Unmarshal([]byte(rendered), &adm); err != nil {
			return nil, fmt.Errorf("parsing expanded motif %q admission: %w", m.Metadata.Name, err)
		}
		admission = &adm
	}

	return &ExpandedMotif{
		Resources: resources,
		Status:    status,
		Admission: admission,
	}, nil
}

// validateBindings checks that all required inputs are provided and no
// unknown inputs are supplied.
func validateBindings(m *orktypes.Motif, bindings map[string]string) error {
	declared := make(map[string]*orktypes.MotifInput, len(m.Inputs))
	for i := range m.Inputs {
		declared[m.Inputs[i].Name] = &m.Inputs[i]
	}

	for _, input := range m.Inputs {
		if input.Required {
			if _, ok := bindings[input.Name]; !ok {
				return fmt.Errorf(
					"motif %q: required input %q not provided in with:\n"+
						"  Motif requires: %s\n"+
						"  Missing: %s",
					m.Metadata.Name, input.Name,
					strings.Join(requiredInputNames(m.Inputs), ", "),
					input.Name,
				)
			}
		}
	}

	for name := range bindings {
		if _, ok := declared[name]; !ok {
			return fmt.Errorf(
				"motif %q: unknown input %q in with: — declared inputs: %s",
				m.Metadata.Name, name,
				strings.Join(inputNames(m.Inputs), ", "),
			)
		}
	}

	return nil
}

// resolveDefaults returns the full input map with defaults filled in for
// optional inputs not present in bindings.
func resolveDefaults(m *orktypes.Motif, bindings map[string]string) map[string]string {
	resolved := make(map[string]string, len(m.Inputs))
	for _, input := range m.Inputs {
		if val, ok := bindings[input.Name]; ok {
			resolved[input.Name] = val
		} else if input.Default != "" {
			resolved[input.Name] = input.Default
		}
	}
	return resolved
}

// renderInputs replaces `{{ .inputs.KEY }}` and `{{ inputs.KEY }}` with
// the corresponding resolved value from the map. It does NOT evaluate any
// other template expressions, leaving them for runtime evaluation.
func renderInputs(resourceYAML string, resolved map[string]string) (string, error) {
	result := resourceYAML
	for key, val := range resolved {
		patterns := []string{
			fmt.Sprintf("{{ .inputs.%s }}", key),
			fmt.Sprintf("{{ inputs.%s }}", key),
		}
		for _, pat := range patterns {
			result = strings.ReplaceAll(result, pat, val)
		}
	}
	return result, nil
}

// renderInputs replaces {{ inputs.Name }} expressions in the YAML string
// with resolved values using Go's text/template.
func oldRenderInputs(resourceYAML string, resolved map[string]string) (string, error) {
	data := map[string]interface{}{
		"inputs": resolved,
	}
	tmpl, err := template.New("motif").Option("missingkey=error").Parse(resourceYAML)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ValidateMotifTemplates checks that all inputs.X references in the resource
// YAML correspond to declared input names. Returns a list of error strings.
func ValidateMotifTemplates(m *orktypes.Motif) []string {
	var errs []string
	declared := make(map[string]bool)
	for _, input := range m.Inputs {
		declared[input.Name] = true
	}

	resourceYAML, err := yaml.Marshal(m.Resources)
	if err != nil {
		return []string{"could not marshal resources for template validation"}
	}

	re := regexp.MustCompile(`\{\{\s*(?:index\s+)?\.?inputs\.?(\w+)`)
	matches := re.FindAllStringSubmatch(string(resourceYAML), -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		inputName := match[1]
		if !declared[inputName] {
			errs = append(errs, fmt.Sprintf(
				"template references inputs.%s but no input named %q is declared",
				inputName, inputName,
			))
		}
	}
	return errs
}

// MergeHookTemplates appends resources from src into dst.
func MergeHookTemplates(dst, src *orktypes.HookTemplates) {
	if dst == nil || src == nil {
		return
	}
	dst.Deployments = append(dst.Deployments, src.Deployments...)
	dst.ReplicaSets = append(dst.ReplicaSets, src.ReplicaSets...)
	dst.StatefulSets = append(dst.StatefulSets, src.StatefulSets...)
	dst.Services = append(dst.Services, src.Services...)
	dst.Pods = append(dst.Pods, src.Pods...)
	dst.ConfigMaps = append(dst.ConfigMaps, src.ConfigMaps...)
	dst.Secrets = append(dst.Secrets, src.Secrets...)
	dst.Jobs = append(dst.Jobs, src.Jobs...)
	dst.CronJobs = append(dst.CronJobs, src.CronJobs...)
	dst.ServiceAccounts = append(dst.ServiceAccounts, src.ServiceAccounts...)
	dst.HorizontalPodAutoscalers = append(dst.HorizontalPodAutoscalers, src.HorizontalPodAutoscalers...)
	dst.PodDisruptionBudgets = append(dst.PodDisruptionBudgets, src.PodDisruptionBudgets...)
	dst.Ingresses = append(dst.Ingresses, src.Ingresses...)
	dst.PersistentVolumes = append(dst.PersistentVolumes, src.PersistentVolumes...)
	dst.PersistentVolumeClaims = append(dst.PersistentVolumeClaims, src.PersistentVolumeClaims...)
	dst.Namespaces = append(dst.Namespaces, src.Namespaces...)
	dst.Roles = append(dst.Roles, src.Roles...)
	dst.RoleBindings = append(dst.RoleBindings, src.RoleBindings...)
	dst.ClusterRoles = append(dst.ClusterRoles, src.ClusterRoles...)
	dst.ClusterRoleBindings = append(dst.ClusterRoleBindings, src.ClusterRoleBindings...)
}

func inputNames(inputs []orktypes.MotifInput) []string {
	names := make([]string, len(inputs))
	for i, input := range inputs {
		names[i] = input.Name
	}
	return names
}

func requiredInputNames(inputs []orktypes.MotifInput) []string {
	var names []string
	for _, input := range inputs {
		if input.Required {
			names = append(names, input.Name)
		}
	}
	return names
}
