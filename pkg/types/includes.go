package types

import (
	"fmt"
	"os"
	"path/filepath"

	orkutils "github.com/orkspace/orkestra/pkg/utils"
)

// ExpandStatusInclude resolves the status.include field by reading the
// referenced file, unmarshaling its "fields:" list, and prepending it to
// the inline fields. Inline fields append after included ones.
// The include path is resolved relative to baseDir. Cleared after expansion.
func ExpandStatusInclude(s *StatusConfig, baseDir string) error {
	if s == nil || s.Include == "" {
		return nil
	}
	path := s.Include
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading status.include %q: %w", s.Include, err)
	}
	var f struct {
		Fields []StatusFieldSpec `yaml:"fields"`
	}
	if err := orkutils.StrictUnmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing status.include %q: %w", s.Include, err)
	}
	s.Fields = append(f.Fields, s.Fields...)
	s.Include = ""
	return nil
}

// ExpandValidationInclude resolves the validation.include field by reading the
// referenced file, unmarshaling its "rules:" list, and prepending it to the
// inline rules. Inline rules append after included ones.
// The include path is resolved relative to baseDir. Cleared after expansion.
func ExpandValidationInclude(v *ValidationConfig, baseDir string) error {
	if v == nil || v.Include == "" {
		return nil
	}
	path := v.Include
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading validation.include %q: %w", v.Include, err)
	}
	var f struct {
		Rules []ValidationRule `yaml:"rules"`
	}
	if err := orkutils.StrictUnmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing validation.include %q: %w", v.Include, err)
	}
	v.Rules = append(f.Rules, v.Rules...)
	v.Include = ""
	return nil
}

// ExpandMutationInclude resolves the mutation.include field by reading the
// referenced file, unmarshaling its "rules:" list, and prepending it to the
// inline rules. Inline rules append after included ones.
// The include path is resolved relative to baseDir. Cleared after expansion.
func ExpandMutationInclude(mu *MutationConfig, baseDir string) error {
	if mu == nil || mu.Include == "" {
		return nil
	}
	path := mu.Include
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading mutation.include %q: %w", mu.Include, err)
	}
	var f struct {
		Rules []MutationRule `yaml:"rules"`
	}
	if err := orkutils.StrictUnmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing mutation.include %q: %w", mu.Include, err)
	}
	mu.Rules = append(f.Rules, mu.Rules...)
	mu.Include = ""
	return nil
}

// ExpandConversionInclude resolves the conversion.include field by reading the
// referenced file, unmarshaling its "rules:" list, and prepending it to the
// inline rules. Inline rules append after included ones.
// The include path is resolved relative to baseDir. Cleared after expansion.
func ExpandConversionInclude(co *CRDConversion, baseDir string) error {
	if co == nil || co.Include == "" {
		return nil
	}
	path := co.Include
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading conversion.include %q: %w", co.Include, err)
	}
	var f struct {
		Paths []ConversionPath `yaml:"paths"`
	}
	if err := orkutils.StrictUnmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing conversion.include %q: %w", co.Include, err)
	}
	co.Paths = append(f.Paths, co.Paths...)
	co.Include = ""
	return nil
}

// ExpandNotesInclude resolves the notes.include field by reading the referenced
// file, unmarshaling its "functions:" list, and prepending it to the inline
// functions. Inline functions append after included ones.
// The include path is resolved relative to baseDir. Cleared after expansion.
func ExpandNotesInclude(nr *NoteRegistry, baseDir string) error {
	if nr.Include == "" {
		return nil
	}
	path := nr.Include
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading notes.include %q: %w", nr.Include, err)
	}
	var f struct {
		Functions []UserDefinedNote `yaml:"functions"`
	}
	if err := orkutils.StrictUnmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing notes.include %q: %w", nr.Include, err)
	}
	nr.Functions = append(f.Functions, nr.Functions...)
	nr.Include = ""
	return nil
}

// ExpandProfileInclude resolves the profiles.include field by reading the
// referenced file, unmarshaling its "profiles:" block, and prepending each
// sub-list before the inline entries. Inline entries append after included ones.
// The include path is resolved relative to baseDir. Cleared after expansion.
func ExpandProfileInclude(r *ProfileRegistry, baseDir string) error {
	if r.Include == "" {
		return nil
	}
	path := r.Include
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading profiles.include %q: %w", r.Include, err)
	}
	var f struct {
		Profiles ProfileRegistry `yaml:"profiles"`
	}
	if err := orkutils.StrictUnmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing profiles.include %q: %w", r.Include, err)
	}
	p := f.Profiles
	r.NetworkPolicies = append(p.NetworkPolicies, r.NetworkPolicies...)
	r.ResourceQuotas = append(p.ResourceQuotas, r.ResourceQuotas...)
	r.LimitRanges = append(p.LimitRanges, r.LimitRanges...)
	r.HPA = append(p.HPA, r.HPA...)
	r.PDB = append(p.PDB, r.PDB...)
	r.RollingUpdate = append(p.RollingUpdate, r.RollingUpdate...)
	r.Reconciler = append(p.Reconciler, r.Reconciler...)
	r.Resources = append(p.Resources, r.Resources...)
	r.Probes = append(p.Probes, r.Probes...)
	r.ContainerSecurity = append(p.ContainerSecurity, r.ContainerSecurity...)
	r.PodSecurity = append(p.PodSecurity, r.PodSecurity...)
	r.Include = ""
	return nil
}

// ExpandExternalCalls resolves include entries in a []ExternalCallSpec list.
// An entry with include: set is replaced in-place by the "calls:" list from the
// referenced file. Entries without include: are kept as-is.
// The include path is resolved relative to baseDir.
func ExpandExternalCalls(calls []ExternalCallSpec, baseDir string) ([]ExternalCallSpec, error) {
	var expanded []ExternalCallSpec
	for _, call := range calls {
		if call.Include == "" {
			expanded = append(expanded, call)
			continue
		}
		path := call.Include
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading external include %q: %w", call.Include, err)
		}
		var f struct {
			Calls []ExternalCallSpec `yaml:"calls"`
		}
		if err := orkutils.StrictUnmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("parsing external include %q: %w", call.Include, err)
		}
		expanded = append(expanded, f.Calls...)
	}
	return expanded, nil
}

// ExpandSimulateOpsIncludes resolves include entries in expect.Ops, expect.Absent,
// and each per-CRD sub-expect. An entry {include: ./path.yaml} is replaced
// in-place by the ops: list from the referenced file.
// The include path is resolved relative to baseDir.
func ExpandSimulateOpsIncludes(expect *SimulateExpect, baseDir string) error {
	if expect == nil {
		return nil
	}
	var err error
	if expect.Ops, err = expandOpRules(expect.Ops, baseDir); err != nil {
		return err
	}
	if expect.Absent, err = expandOpRules(expect.Absent, baseDir); err != nil {
		return fmt.Errorf("absent: %w", err)
	}
	for name, sub := range expect.CRDs {
		if sub == nil {
			continue
		}
		if sub.Ops, err = expandOpRules(sub.Ops, baseDir); err != nil {
			return fmt.Errorf("crds[%s]: %w", name, err)
		}
		if sub.Absent, err = expandOpRules(sub.Absent, baseDir); err != nil {
			return fmt.Errorf("crds[%s].absent: %w", name, err)
		}
	}
	return nil
}

func expandOpRules(rules []SimulateOpRule, baseDir string) ([]SimulateOpRule, error) {
	var expanded []SimulateOpRule
	for _, rule := range rules {
		if rule.Include == "" {
			expanded = append(expanded, rule)
			continue
		}
		path := rule.Include
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading ops include %q: %w", rule.Include, err)
		}
		var f struct {
			Ops []SimulateOpRule `yaml:"ops"`
		}
		if err := orkutils.StrictUnmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("parsing ops include %q: %w", rule.Include, err)
		}
		expanded = append(expanded, f.Ops...)
	}
	return expanded, nil
}

// ExpandIDPInclude resolves the idp.include field by reading the referenced
// file, unmarshaling its "fields:" and "additionalFields:" maps, and merging
// them under the inline equivalents. Inline entries take precedence — included
// keys present in both are overridden, per bucket (fields, additionalFields.labels,
// additionalFields.annotations independently).
// The include path is resolved relative to baseDir. Cleared after expansion.
func ExpandIDPInclude(idp *IDPConfig, baseDir string) error {
	if idp == nil || idp.Include == "" {
		return nil
	}
	path := idp.Include
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading idp.include %q: %w", idp.Include, err)
	}
	var f struct {
		Fields           map[string]IDPFieldConfig `yaml:"fields"`
		AdditionalFields *AdditionalIDPFields      `yaml:"additionalFields"`
	}
	if err := orkutils.StrictUnmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing idp.include %q: %w", idp.Include, err)
	}
	merged := make(map[string]IDPFieldConfig, len(f.Fields)+len(idp.Fields))
	for k, v := range f.Fields {
		merged[k] = v
	}
	for k, v := range idp.Fields {
		merged[k] = v
	}
	idp.Fields = merged

	if f.AdditionalFields != nil {
		if idp.AdditionalFields == nil {
			idp.AdditionalFields = &AdditionalIDPFields{}
		}
		idp.AdditionalFields.Labels = mergeIDPFieldConfigs(f.AdditionalFields.Labels, idp.AdditionalFields.Labels)
		idp.AdditionalFields.Annotations = mergeIDPFieldConfigs(f.AdditionalFields.Annotations, idp.AdditionalFields.Annotations)
	}

	idp.Include = ""
	return nil
}

// ExpandApplyAPIAuth resolves include entries in GatewayConfig.ApplyAPI.Auth.
// If auth.Include is set, it reads the referenced file, unmarshals its "tokens:" list,
// and merges it with the inline tokens. Inline tokens override included tokens
// with the same name.
// The include path is resolved relative to baseDir. Cleared after expansion.
func ExpandApplyAPIAuth(gw *GatewayConfig, baseDir string) error {
	if gw == nil || gw.ApplyAPI == nil {
		return nil
	}
	auth := &gw.ApplyAPI.Auth
	if auth.Include == "" {
		return nil
	}

	path := auth.Include
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading applyAPI.auth.include %q: %w", auth.Include, err)
	}

	var f struct {
		Tokens []ApplyAPIToken `yaml:"tokens"`
	}
	if err := orkutils.StrictUnmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing applyAPI.auth.include %q: %w", auth.Include, err)
	}

	// Merge: included tokens first, then inline overrides by name
	merged := make(map[string]ApplyAPIToken)
	for _, t := range f.Tokens {
		merged[t.Name] = t
	}
	for _, t := range auth.Tokens {
		merged[t.Name] = t
	}

	// Convert map back to slice
	auth.Tokens = make([]ApplyAPIToken, 0, len(merged))
	for _, t := range merged {
		auth.Tokens = append(auth.Tokens, t)
	}
	auth.Include = ""

	return nil
}

// ExpandIDPAllowedTokensInclude resolves the idp.allowedTokens include.
// If at.Include is set, it reads the referenced file, unmarshals its "allowedTokens:" map,
// and merges it with the inline tokens. Inline tokens override included tokens
// with the same name.
// The include path is resolved relative to baseDir. Cleared after expansion.
func ExpandIDPAllowedTokensInclude(idp *IDPConfig, baseDir string) error {
	if idp == nil {
		return nil
	}

	at := idp.AllowedTokens
	if at.Include == "" {
		return nil
	}

	path := at.Include
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading idp.allowedTokens.include %q: %w", at.Include, err)
	}

	var f struct {
		AllowedTokens map[string]IDPTokenPermissions `yaml:"allowedTokens"`
	}
	if err := orkutils.StrictUnmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing idp.allowedTokens.include %q: %w", at.Include, err)
	}

	// Merge: included tokens first, then inline overrides
	merged := make(map[string]IDPTokenPermissions)
	for k, v := range f.AllowedTokens {
		merged[k] = v
	}
	for k, v := range at.Tokens {
		merged[k] = v
	}

	// Assign merged result back
	idp.AllowedTokens.Tokens = merged
	idp.AllowedTokens.Include = ""

	return nil
}

// mergeIDPFieldConfigs merges included and inline IDPFieldConfig maps, inline
// taking precedence. Returns nil when both inputs are empty, matching the
// omitempty shape AdditionalIDPFields expects.
func mergeIDPFieldConfigs(included, inline map[string]IDPFieldConfig) map[string]IDPFieldConfig {
	if len(included) == 0 && len(inline) == 0 {
		return nil
	}
	merged := make(map[string]IDPFieldConfig, len(included)+len(inline))
	for k, v := range included {
		merged[k] = v
	}
	for k, v := range inline {
		merged[k] = v
	}
	return merged
}
