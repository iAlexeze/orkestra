package katalog

import (
	"fmt"
	"text/template"

	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/note"
)

// validateUserNotes checks the notes: block declared on the Katalog.
//
// Enforces:
//  1. Each note must have a non-empty name.
//  2. Each note must have a non-empty expression.
//  3. Expression must be a valid Go template.
//  4. No duplicate names.
//  5. Warns when a note name shadows a built-in unless shadow: true is set.
func (k *Katalog) validateUserNotes() error {
	reg := k.Notes
	if reg.IsEmpty() {
		return nil
	}

	builtins := note.Map()

	// Pre-build stubs for every user-defined note so cross-references resolve
	// during template parsing. Matches runtime behaviour where the full funcmap
	// (built-ins + all user notes) is assembled before any expression runs.
	funcMap := make(template.FuncMap, len(builtins)+len(reg.Functions))
	for k, v := range builtins {
		funcMap[k] = v
	}
	for _, n := range reg.Functions {
		funcMap[n.Name] = func() string { return "" }
	}

	seen := make(map[string]bool, len(reg.Functions))

	for i, n := range reg.Functions {
		if n.Name == "" {
			return fmt.Errorf("notes[%d]: name must not be empty", i)
		}
		if n.Expression == "" {
			return fmt.Errorf("notes[%d] %q: expression must not be empty", i, n.Name)
		}
		if seen[n.Name] {
			return fmt.Errorf("notes: duplicate note name %q — names must be unique", n.Name)
		}
		seen[n.Name] = true

		if _, err := template.New("").Funcs(funcMap).Parse(n.Expression); err != nil {
			return fmt.Errorf("notes %q: invalid expression: %w", n.Name, err)
		}

		if _, isBuiltin := builtins[n.Name]; isBuiltin && !n.Shadow {
			logger.Warn().Msgf(
				"notes %q shadows a built-in Orkestra note — add `shadow: true` to acknowledge",
				n.Name,
			)
		}
	}
	return nil
}
