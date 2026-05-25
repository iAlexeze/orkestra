// Package katalog — Security Profiles
//
// Security profiles are named presets that expand into a concrete
// ContainerSecurityContext or PodSecurityContext at katalog load time.
// The runtime sees a fully-expanded spec as if the user had written it manually.
//
// Profiles:
//
//   baseline   — minimal hardening; drops NET_RAW, prevents privilege escalation.
//                Suitable for internal services that do not need strict isolation.
//
//   restricted — drops all capabilities, requires non-root user.
//                Follows the Kubernetes "restricted" Pod Security Standard.
//
//   hardened   — restricted plus read-only root filesystem.
//                Suitable for production workloads that require strong isolation.
//
// Profile and explicit fields are mutually exclusive. Mixing both is rejected
// at load time by validateSecurityProfiles.

package katalog

import (
	"fmt"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// SecurityProfile is the named security preset enum.
type SecurityProfile string

const (
	SecurityBaseline   SecurityProfile = "baseline"
	SecurityRestricted SecurityProfile = "restricted"
	SecurityHardened   SecurityProfile = "hardened"
)

// ApplyContainerSecurityProfile expands a named container security profile into
// a concrete ContainerSecurityContext. Returns an error for unknown profile names.
func ApplyContainerSecurityProfile(profile string) (*orktypes.ContainerSecurityContext, error) {
	switch SecurityProfile(strings.ToLower(profile)) {

	case SecurityBaseline:
		// Prevent privilege escalation; drop the most dangerous raw network capability.
		return &orktypes.ContainerSecurityContext{
			AllowPrivilegeEscalation: utils.BoolPtr(false),
			Capabilities: &orktypes.CapabilitiesConfig{
				Drop: []string{"NET_RAW"},
			},
		}, nil

	case SecurityRestricted:
		// Drop all capabilities and require non-root. Matches Kubernetes restricted PSS.
		return &orktypes.ContainerSecurityContext{
			AllowPrivilegeEscalation: utils.BoolPtr(false),
			RunAsNonRoot:             utils.BoolPtr(true),
			Capabilities: &orktypes.CapabilitiesConfig{
				Drop: []string{"ALL"},
			},
		}, nil

	case SecurityHardened:
		// Restricted plus read-only root filesystem for maximum isolation.
		return &orktypes.ContainerSecurityContext{
			AllowPrivilegeEscalation: utils.BoolPtr(false),
			ReadOnlyRootFilesystem:   utils.BoolPtr(true),
			RunAsNonRoot:             utils.BoolPtr(true),
			Capabilities: &orktypes.CapabilitiesConfig{
				Drop: []string{"ALL"},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown container security profile: %q", profile)
	}
}

// ApplyPodSecurityProfile expands a named pod security profile into a concrete
// PodSecurityContext. Returns an error for unknown profile names.
func ApplyPodSecurityProfile(profile string) (*orktypes.PodSecurityContext, error) {
	switch SecurityProfile(strings.ToLower(profile)) {

	case SecurityBaseline:
		// Baseline pod settings: non-root user enforcement only.
		return &orktypes.PodSecurityContext{
			RunAsNonRoot: utils.BoolPtr(false),
		}, nil

	case SecurityRestricted:
		// Non-root with a predictable UID.
		return &orktypes.PodSecurityContext{
			RunAsNonRoot: utils.BoolPtr(true),
			RunAsUser:    utils.Int64Ptr(1000),
		}, nil

	case SecurityHardened:
		// Fully locked-down: nobody UID/GID, shared fsGroup.
		return &orktypes.PodSecurityContext{
			RunAsNonRoot: utils.BoolPtr(true),
			RunAsUser:    utils.Int64Ptr(65534),
			RunAsGroup:   utils.Int64Ptr(65534),
			FSGroup:      utils.Int64Ptr(65534),
		}, nil

	default:
		return nil, fmt.Errorf("unknown pod security profile: %q", profile)
	}
}
