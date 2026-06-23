package katalog

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// -----------------------------------------------------------------------------
// Validation: Reconciler Mode (entrypoint)
// -----------------------------------------------------------------------------

// validateReconcilerMode determines the reconciler mode (dynamic/typed) for each
// CRD and performs consistency checks for hooks, constructors, and managed
// resources. Delegates to helper functions for clarity.
func (k *Katalog) validateReconcilerMode() error {
	for name, crd := range k.enabledCRDs {

		// Mode defaulting + basic validation
		if err := k.validateMode(name, &crd); err != nil {
			return err
		}

		// Hooks validation
		if err := k.validateHooks(name, &crd); err != nil {
			return err
		}

		// Constructor validation
		if err := k.validateConstructor(name, &crd); err != nil {
			return err
		}

		// Managed resources validation
		if err := k.validateManagedResources(name, &crd); err != nil {
			return err
		}

		// Save updated CRD entry
		k.enabledCRDs[name] = crd
	}

	return nil
}

// -----------------------------------------------------------------------------
// validateMode — defaulting + mode-level checks
// -----------------------------------------------------------------------------

func (k *Katalog) validateMode(name string, crd *orktypes.CRDEntry) error {
	mode := crd.Mode

	switch mode {
	case "":
		// Default mode
		if crd.APITypes.Location != "" {
			crd.Mode = orktypes.CRDModeTyped
			logger.Debug().
				Str("crd", name).
				Msg("reconciler mode defaulted to 'typed' because apiTypes.location is set")
		} else {
			crd.Mode = orktypes.CRDModeDynamic
			logger.Debug().
				Str("crd", name).
				Msg("reconciler mode defaulted to 'dynamic'")
		}

	case orktypes.CRDModeDynamic, orktypes.CRDModeTyped:
		// Valid modes

	default:
		return fmt.Errorf(
			"CRD %q: reconciler mode %q is not supported — use %q or %q",
			name, mode,
			orktypes.CRDModeDynamic,
			orktypes.CRDModeTyped,
		)
	}

	// Typed mode requires apiTypes.location
	if crd.Mode == orktypes.CRDModeTyped && crd.APITypes.Location == "" {
		return fmt.Errorf(
			"CRD %q: mode is 'typed' but apiTypes.location is missing",
			name,
		)
	}

	return nil
}

// -----------------------------------------------------------------------------
// validateHooks — structural + typed-mode requirements
// -----------------------------------------------------------------------------

func (k *Katalog) validateHooks(name string, crd *orktypes.CRDEntry) error {
	if !crd.WithHooksDecl() {
		return nil
	}

	// Hooks and constructor cannot both be declared
	if crd.WithConstructorDecl() {
		return fmt.Errorf(
			"CRD %q: cannot declare both 'hooks' and 'constructor' – choose one",
			name,
		)
	}

	// Typed CRD required
	if crd.APITypes.Location == "" {
		return fmt.Errorf(
			"CRD %q: hooks declared but apiTypes.location is missing (typed hooks require typed CRD)",
			name,
		)
	}

	// Required fields
	if crd.OperatorBox.Reconciler.Hooks.Location == "" {
		return fmt.Errorf("CRD %q: reconciler.hooks.location is required", name)
	}
	if crd.OperatorBox.Reconciler.Hooks.Function == "" {
		return fmt.Errorf("CRD %q: reconciler.hooks.function is required", name)
	}

	return nil
}

// -----------------------------------------------------------------------------
// validateConstructor — structural + typed-mode requirements
// -----------------------------------------------------------------------------

func (k *Katalog) validateConstructor(name string, crd *orktypes.CRDEntry) error {
	if !crd.WithConstructorDecl() {
		return nil
	}

	// Constructor and hooks cannot both be declared
	if crd.WithHooksDecl() {
		return fmt.Errorf(
			"CRD %q: cannot declare both 'hooks' and 'constructor' – choose one",
			name,
		)
	}

	// Typed CRD required
	if crd.APITypes.Location == "" {
		return fmt.Errorf(
			"CRD %q: constructor declared but apiTypes.location is missing (typed constructor requires typed CRD)",
			name,
		)
	}

	// Required fields
	if crd.OperatorBox.Reconciler.ConstructorDecl.Location == "" {
		return fmt.Errorf("CRD %q: reconciler.constructor.location is required", name)
	}
	if crd.OperatorBox.Reconciler.ConstructorDecl.Function == "" {
		return fmt.Errorf("CRD %q: reconciler.constructor.function is required", name)
	}

	return nil
}

// -----------------------------------------------------------------------------
// validateManagedResources — RBAC requirements for typed mode
// -----------------------------------------------------------------------------

func (k *Katalog) validateManagedResources(name string, crd *orktypes.CRDEntry) error {
	// Hooks declared but no resources
	if crd.WithHooksDecl() && !crd.WithHookManagedResources() {
		return fmt.Errorf(
			"CRD %q: hooks declared but no managed resources defined.\n\n"+
				"Typed hooks require RBAC permissions, which are generated from the\n"+
				"'resources' list under the hooks block.\n\n"+
				"Example:\n"+
				"  hooks:\n"+
				"    location: %s\n"+
				"    function: %s\n"+
				"    resources:\n"+
				"      - kind: Pod\n"+
				"      - kind: Deployment\n",
			name,
			crd.OperatorBox.Reconciler.Hooks.Location,
			crd.OperatorBox.Reconciler.Hooks.Function,
		)
	}

	// Constructor declared but no resources
	if crd.WithConstructorDecl() && !crd.WithConstructorManagedResources() {
		return fmt.Errorf(
			"CRD %q: constructor declared but no managed resources defined.\n\n"+
				"Typed constructors take full ownership of reconciliation and must\n"+
				"declare the Kubernetes resources they manage so RBAC can be generated.\n\n"+
				"Example:\n"+
				"  constructor:\n"+
				"    location: %s\n"+
				"    function: %s\n"+
				"    resources:\n"+
				"      - kind: StatefulSet\n"+
				"      - kind: Service\n",
			name,
			crd.OperatorBox.Reconciler.ConstructorDecl.Location,
			crd.OperatorBox.Reconciler.ConstructorDecl.Function,
		)
	}

	return nil
}
