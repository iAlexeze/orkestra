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
// file, unmarshaling its "fields:" map, and merging it under the inline fields.
// Inline fields take precedence — included keys present in both are overridden.
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
		Fields map[string]IDPFieldConfig `yaml:"fields"`
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
	idp.Include = ""
	return nil
}
