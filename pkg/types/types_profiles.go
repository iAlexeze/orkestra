package types

import "fmt"

// ProfileRegistry holds all user-defined profiles declared in a Katalog or Motif.
// Profiles are resolved before built-ins at validate and reconcile time.
// Template expressions in profile field values are allowed and resolved at reconcile time.
type ProfileRegistry struct {
	NetworkPolicies []NetworkPolicyProfileDef  `yaml:"networkPolicies,omitempty" json:"networkPolicies,omitempty"`
	ResourceQuotas  []ResourceQuotaProfileDef  `yaml:"resourceQuotas,omitempty" json:"resourceQuotas,omitempty"`
	LimitRanges     []LimitRangeProfileDef     `yaml:"limitRanges,omitempty"    json:"limitRanges,omitempty"`
	HPA             []HPAProfileDef            `yaml:"hpa,omitempty"            json:"hpa,omitempty"`
	PDB             []PDBProfileDef            `yaml:"pdb,omitempty"            json:"pdb,omitempty"`
	RollingUpdate   []RollingUpdateProfileDef  `yaml:"rollingUpdate,omitempty"  json:"rollingUpdate,omitempty"`
}

func (r ProfileRegistry) IsEmpty() bool {
	return len(r.NetworkPolicies) == 0 &&
		len(r.ResourceQuotas) == 0 &&
		len(r.LimitRanges) == 0 &&
		len(r.HPA) == 0 &&
		len(r.PDB) == 0 &&
		len(r.RollingUpdate) == 0
}

func (r ProfileRegistry) LookupNetworkPolicy(name string) (NetworkPolicyProfileDef, bool) {
	for _, e := range r.NetworkPolicies {
		if e.Name == name {
			return e, true
		}
	}
	return NetworkPolicyProfileDef{}, false
}

func (r ProfileRegistry) LookupResourceQuota(name string) (ResourceQuotaProfileDef, bool) {
	for _, e := range r.ResourceQuotas {
		if e.Name == name {
			return e, true
		}
	}
	return ResourceQuotaProfileDef{}, false
}

func (r ProfileRegistry) LookupLimitRange(name string) (LimitRangeProfileDef, bool) {
	for _, e := range r.LimitRanges {
		if e.Name == name {
			return e, true
		}
	}
	return LimitRangeProfileDef{}, false
}

func (r ProfileRegistry) LookupHPA(name string) (HPAProfileDef, bool) {
	for _, e := range r.HPA {
		if e.Name == name {
			return e, true
		}
	}
	return HPAProfileDef{}, false
}

func (r ProfileRegistry) LookupPDB(name string) (PDBProfileDef, bool) {
	for _, e := range r.PDB {
		if e.Name == name {
			return e, true
		}
	}
	return PDBProfileDef{}, false
}

func (r ProfileRegistry) LookupRollingUpdate(name string) (RollingUpdateProfileDef, bool) {
	for _, e := range r.RollingUpdate {
		if e.Name == name {
			return e, true
		}
	}
	return RollingUpdateProfileDef{}, false
}

// Merge combines other into r, returning a conflict error if the same name
// appears in the same class in both registries.
func (r ProfileRegistry) Merge(other ProfileRegistry, otherSource string) (ProfileRegistry, error) {
	merged := r
	for _, e := range other.NetworkPolicies {
		if _, found := r.LookupNetworkPolicy(e.Name); found {
			return ProfileRegistry{}, profileConflictError("networkPolicies", e.Name, otherSource)
		}
		merged.NetworkPolicies = append(merged.NetworkPolicies, e)
	}
	for _, e := range other.ResourceQuotas {
		if _, found := r.LookupResourceQuota(e.Name); found {
			return ProfileRegistry{}, profileConflictError("resourceQuotas", e.Name, otherSource)
		}
		merged.ResourceQuotas = append(merged.ResourceQuotas, e)
	}
	for _, e := range other.LimitRanges {
		if _, found := r.LookupLimitRange(e.Name); found {
			return ProfileRegistry{}, profileConflictError("limitRanges", e.Name, otherSource)
		}
		merged.LimitRanges = append(merged.LimitRanges, e)
	}
	for _, e := range other.HPA {
		if _, found := r.LookupHPA(e.Name); found {
			return ProfileRegistry{}, profileConflictError("hpa", e.Name, otherSource)
		}
		merged.HPA = append(merged.HPA, e)
	}
	for _, e := range other.PDB {
		if _, found := r.LookupPDB(e.Name); found {
			return ProfileRegistry{}, profileConflictError("pdb", e.Name, otherSource)
		}
		merged.PDB = append(merged.PDB, e)
	}
	for _, e := range other.RollingUpdate {
		if _, found := r.LookupRollingUpdate(e.Name); found {
			return ProfileRegistry{}, profileConflictError("rollingUpdate", e.Name, otherSource)
		}
		merged.RollingUpdate = append(merged.RollingUpdate, e)
	}
	return merged, nil
}

func profileConflictError(class, name, source string) error {
	return fmt.Errorf("profile conflict: %s %q defined in both %s and the katalog", class, name, source)
}

// NetworkPolicyProfileDef defines a named NetworkPolicy profile.
// Fields mirror NetworkPolicyTemplateSource minus declaration-level concerns.
// Template expressions are allowed and resolved at reconcile time.
type NetworkPolicyProfileDef struct {
	Name        string                     `yaml:"name" json:"name"`
	Description string                     `yaml:"description,omitempty" json:"description,omitempty"`
	PodSelector map[string]interface{}     `yaml:"podSelector,omitempty" json:"podSelector,omitempty"`
	Ingress     []NetworkPolicyIngressRule `yaml:"ingress,omitempty" json:"ingress,omitempty"`
	Egress      []NetworkPolicyEgressRule  `yaml:"egress,omitempty" json:"egress,omitempty"`
	PolicyTypes []string                   `yaml:"policyTypes,omitempty" json:"policyTypes,omitempty"`
}

// ResourceQuotaProfileDef defines a named ResourceQuota profile.
type ResourceQuotaProfileDef struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Hard        map[string]string `yaml:"hard" json:"hard"`
}

// LimitRangeProfileDef defines a named LimitRange profile.
type LimitRangeProfileDef struct {
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description,omitempty" json:"description,omitempty"`
	Limits      []LimitRangeItem `yaml:"limits" json:"limits"`
}

// HPAProfileDef defines a named HPA profile.
// Template expressions in MinReplicas, MaxReplicas, and TargetCPUUtilizationPercentage
// are resolved at reconcile time.
type HPAProfileDef struct {
	Name                           string       `yaml:"name" json:"name"`
	Description                    string       `yaml:"description,omitempty" json:"description,omitempty"`
	MinReplicas                    string       `yaml:"minReplicas,omitempty" json:"minReplicas,omitempty"`
	MaxReplicas                    string       `yaml:"maxReplicas,omitempty" json:"maxReplicas,omitempty"`
	TargetCPUUtilizationPercentage string       `yaml:"targetCPUUtilizationPercentage,omitempty" json:"targetCPUUtilizationPercentage,omitempty"`
	Behavior                       *HPABehavior `yaml:"behavior,omitempty" json:"behavior,omitempty"`
}

// PDBProfileDef defines a named PodDisruptionBudget profile.
type PDBProfileDef struct {
	Name           string `yaml:"name" json:"name"`
	Description    string `yaml:"description,omitempty" json:"description,omitempty"`
	MinAvailable   string `yaml:"minAvailable,omitempty" json:"minAvailable,omitempty"`
	MaxUnavailable string `yaml:"maxUnavailable,omitempty" json:"maxUnavailable,omitempty"`
}

// RollingUpdateProfileDef defines a named rolling update profile.
type RollingUpdateProfileDef struct {
	Name           string `yaml:"name" json:"name"`
	Description    string `yaml:"description,omitempty" json:"description,omitempty"`
	MaxSurge       string `yaml:"maxSurge,omitempty" json:"maxSurge,omitempty"`
	MaxUnavailable string `yaml:"maxUnavailable,omitempty" json:"maxUnavailable,omitempty"`
}
