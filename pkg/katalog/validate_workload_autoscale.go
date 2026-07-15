package katalog

import (
	"fmt"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// validateWorkloadAutoscale checks autoscale: blocks on all Deployment declarations.
//
// Rules:
//  1. max is required.
//  2. min < max when both are declared.
//  3. target and increment/decrement are mutually exclusive per direction.
//  4. scaleUp without scaleDown is allowed but emits a warning.
func (k *Katalog) validateWorkloadAutoscale() error {
	for crdName, crd := range k.enabledCRDs {
		box := crd.OperatorBox
		var allDeps []orktypes.DeploymentTemplateSource
		if box.OnCreate != nil {
			allDeps = append(allDeps, box.OnCreate.Deployments...)
		}
		if box.OnReconcile != nil {
			allDeps = append(allDeps, box.OnReconcile.Deployments...)
		}
		for _, dep := range allDeps {
			if dep.Autoscale == nil {
				continue
			}
			if err := validateWorkloadAutoscaleSpec(crdName, dep.Name, dep.Autoscale); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateWorkloadAutoscaleSpec(crdName, depName string, cfg *orktypes.WorkloadAutoscale) error {
	loc := fmt.Sprintf("crds.%s deployment %q autoscale", crdName, depName)

	if cfg.Max == 0 {
		return fmt.Errorf("%s: max is required and must be > 0", loc)
	}

	if cfg.Min != nil && *cfg.Min >= cfg.Max {
		return fmt.Errorf("%s: min (%d) must be less than max (%d)", loc, *cfg.Min, cfg.Max)
	}

	if err := validateScaleDirection(loc+".scaleUp", cfg.ScaleUp, cfg.Max); err != nil {
		return err
	}
	if err := validateScaleDirection(loc+".scaleDown", cfg.ScaleDown, cfg.Max); err != nil {
		return err
	}

	return nil
}

func validateScaleDirection(loc string, dir *orktypes.WorkloadScaleDirection, max ...int32) error {
	if dir == nil {
		return nil
	}
	hasTarget := dir.Target != nil
	hasStep := dir.Increment != nil || dir.Decrement != nil
	if hasTarget && hasStep {
		return fmt.Errorf("%s: target and increment/decrement are mutually exclusive", loc)
	}
	if hasTarget && len(max) > 0 {
		t := *dir.Target
		if t > max[0] {
			return fmt.Errorf("%s: target (%d) exceeds max (%d)", loc, t, max[0])
		}
		if t < 0 {
			return fmt.Errorf("%s: target (%d) must be >= 0", loc, t)
		}
	}
	return nil
}
