// pkg/katalog/motif_validate.go
package katalog

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/registry/motif"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// MotifValidationError represents a single validation failure for a Motif.
type MotifValidationError struct {
	Path    string
	Message string
}

func (e MotifValidationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

// ValidateMotif validates a Motif YAML file at the given path.
// Returns a slice of errors — empty means valid.
func ValidateMotif(path string) []MotifValidationError {
	var errs []MotifValidationError

	m, err := motif.Load(path)
	if err != nil {
		return []MotifValidationError{{Path: path, Message: err.Error()}}
	}

	if m.Metadata.Name == "" {
		errs = append(errs, MotifValidationError{Path: "metadata.name", Message: "name is required"})
	}

	seen := make(map[string]bool)
	for i, input := range m.Inputs {
		if input.Name == "" {
			errs = append(errs, MotifValidationError{
				Path:    fmt.Sprintf("inputs[%d].name", i),
				Message: "input name is required",
			})
			continue
		}
		if seen[input.Name] {
			errs = append(errs, MotifValidationError{
				Path:    fmt.Sprintf("inputs[%d].name", i),
				Message: fmt.Sprintf("duplicate input name: %s", input.Name),
			})
		}
		seen[input.Name] = true

		if input.Required && input.Default != "" {
			errs = append(errs, MotifValidationError{
				Path: fmt.Sprintf("inputs[%d]", i),
				Message: fmt.Sprintf(
					"input %q is required but also has a default — required inputs must not have defaults",
					input.Name,
				),
			})
		}
	}

	if m.Resources == nil {
		errs = append(errs, MotifValidationError{
			Path:    "resources",
			Message: "resources block is required in a Motif",
		})
	}

	if m.Resources != nil {
		for _, msg := range motif.ValidateMotifTemplates(m) {
			errs = append(errs, MotifValidationError{Path: "resources", Message: msg})
		}
	}

	return errs
}

// ValidateMotifImports validates that all imports in an operatorBox have
// required inputs provided in their with: block.
func ValidateMotifImports(crdName string, imports []orktypes.MotifImport) []MotifValidationError {
	var errs []MotifValidationError

	for i, imp := range imports {
		m, err := motif.LoadImport(&imp)
		if err != nil {
			errs = append(errs, MotifValidationError{
				Path:    fmt.Sprintf("spec.crds.%s.operatorBox.imports[%d]", crdName, i),
				Message: fmt.Sprintf("loading motif %q: %s", imp.Motif, err),
			})
			continue
		}

		for _, input := range m.Inputs {
			if input.Required {
				if _, ok := imp.With[input.Name]; !ok {
					errs = append(errs, MotifValidationError{
						Path: fmt.Sprintf("spec.crds.%s.operatorBox.imports[%d]", crdName, i),
						Message: fmt.Sprintf(
							"import %q is missing required input %q\n"+
								"  Motif requires: %s\n"+
								"  Provided: %s\n"+
								"  Missing: %s",
							imp.Motif, input.Name,
							motifRequiredInputList(m.Inputs),
							motifProvidedInputList(imp.With),
							input.Name,
						),
					})
				}
			}
		}

		declared := make(map[string]bool)
		for _, input := range m.Inputs {
			declared[input.Name] = true
		}
		for name := range imp.With {
			if !declared[name] {
				errs = append(errs, MotifValidationError{
					Path: fmt.Sprintf("spec.crds.%s.operatorBox.imports[%d]", crdName, i),
					Message: fmt.Sprintf(
						"unknown input %q supplied in with: — motif %q does not declare it",
						name, imp.Motif,
					),
				})
			}
		}
	}

	return errs
}

func motifRequiredInputList(inputs []orktypes.MotifInput) string {
	var names []string
	for _, input := range inputs {
		if input.Required {
			names = append(names, input.Name)
		}
	}
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}

func motifProvidedInputList(with map[string]string) string {
	if len(with) == 0 {
		return "(none)"
	}
	names := make([]string, 0, len(with))
	for k := range with {
		names = append(names, k)
	}
	return strings.Join(names, ", ")
}
