package katalog

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/profiles"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateUserProfiles checks the profiles: block declared on the katalog.
//
// Enforces:
//  1. No duplicate names within a class.
//  2. Warns (does not error) when a user profile shadows a built-in name.
//  3. Each profile entry must have a non-empty name.
func (k *Katalog) validateUserProfiles() error {
	reg := k.Profiles
	if reg.IsEmpty() {
		return nil
	}

	type check struct {
		class     string
		names     []string
		isBuiltin func(string) bool
	}

	checks := []check{
		{
			class:     "networkPolicies",
			names:     npDefNames(reg.NetworkPolicies),
			isBuiltin: profiles.IsValidNetworkPolicyProfile,
		},
		{
			class:     "resourceQuotas",
			names:     rqDefNames(reg.ResourceQuotas),
			isBuiltin: profiles.IsValidResourceQuotaProfile,
		},
		{
			class:     "limitRanges",
			names:     lrDefNames(reg.LimitRanges),
			isBuiltin: nil,
		},
		{
			class:     "hpa",
			names:     hpaDefNames(reg.HPA),
			isBuiltin: profiles.IsValidHPAProfile,
		},
		{
			class:     "pdb",
			names:     pdbDefNames(reg.PDB),
			isBuiltin: profiles.IsValidPDBProfile,
		},
		{
			class:     "rollingUpdate",
			names:     ruDefNames(reg.RollingUpdate),
			isBuiltin: profiles.IsValidRollingUpdateProfile,
		},
	}

	for _, c := range checks {
		seen := make(map[string]bool, len(c.names))
		for _, name := range c.names {
			if name == "" {
				return fmt.Errorf("profiles.%s: profile entry is missing a name", c.class)
			}
			if seen[name] {
				return fmt.Errorf("profiles.%s: duplicate profile name %q — names must be unique within a class", c.class, name)
			}
			seen[name] = true
			if c.isBuiltin != nil && c.isBuiltin(name) {
				logger.Warn().Msgf(
					"profiles.%s %q shadows a built-in Orkestra profile — the user-defined version will be used instead",
					c.class, name,
				)
			}
		}
	}
	return nil
}

// isUserNetworkPolicyProfile reports whether name is in the katalog's user registry.
func (k *Katalog) isUserNetworkPolicyProfile(name string) bool {
	_, found := k.Profiles.LookupNetworkPolicy(name)
	return found
}
func (k *Katalog) isUserResourceQuotaProfile(name string) bool {
	_, found := k.Profiles.LookupResourceQuota(name)
	return found
}
func (k *Katalog) isUserLimitRangeProfile(name string) bool {
	_, found := k.Profiles.LookupLimitRange(name)
	return found
}
func (k *Katalog) isUserHPAProfile(name string) bool {
	_, found := k.Profiles.LookupHPA(name)
	return found
}
func (k *Katalog) isUserPDBProfile(name string) bool {
	_, found := k.Profiles.LookupPDB(name)
	return found
}
func (k *Katalog) isUserRollingUpdateProfile(name string) bool {
	_, found := k.Profiles.LookupRollingUpdate(name)
	return found
}

func npDefNames(defs []orktypes.NetworkPolicyProfileDef) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.Name
	}
	return out
}
func rqDefNames(defs []orktypes.ResourceQuotaProfileDef) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.Name
	}
	return out
}
func lrDefNames(defs []orktypes.LimitRangeProfileDef) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.Name
	}
	return out
}
func hpaDefNames(defs []orktypes.HPAProfileDef) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.Name
	}
	return out
}
func pdbDefNames(defs []orktypes.PDBProfileDef) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.Name
	}
	return out
}
func ruDefNames(defs []orktypes.RollingUpdateProfileDef) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.Name
	}
	return out
}
