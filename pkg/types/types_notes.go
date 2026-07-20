package types

import "fmt"

// UserDefinedNote is a named template expression declared in a Katalog or Motif.
// Once registered, the note is callable by name in any template expression
// across the Katalog — status fields, when: conditions, normalize, e2e assertions.
type UserDefinedNote struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	// Expression is a Go template string evaluated by the Resolver.
	// Built-in notes and other user-defined notes (same Katalog or imported) are
	// available inside the expression. The result is always a string.
	Expression string `yaml:"expression"`
	// Shadow: true acknowledges that this note intentionally overrides a built-in.
	Shadow bool `yaml:"shadow,omitempty"`
}

// NoteRegistry holds user-defined notes merged from a Katalog and its spec.imports Motifs.
type NoteRegistry struct {
	Include   string            `yaml:"include,omitempty"`
	Functions []UserDefinedNote `yaml:"functions,omitempty"`
}

// Merge merges src into nr. Local (nr) entries override src entries with the same name.
// Two src entries (from different Motifs) with the same name are a hard error.
func (nr NoteRegistry) Merge(src NoteRegistry, srcLabel string) (NoteRegistry, error) {
	if len(src.Functions) == 0 {
		return nr, nil
	}
	existing := make(map[string]bool, len(nr.Functions))
	for _, n := range nr.Functions {
		existing[n.Name] = true
	}
	result := NoteRegistry{Functions: make([]UserDefinedNote, len(nr.Functions))}
	copy(result.Functions, nr.Functions)
	for _, n := range src.Functions {
		if existing[n.Name] {
			// Local note with same name — local wins silently (intentional override).
			continue
		}
		existing[n.Name] = true
		result.Functions = append(result.Functions, n)
	}
	return result, nil
}

// MergeImport merges src into nr detecting conflicts between two non-local sources.
// Used when accumulating notes from multiple spec.imports Motifs.
func (nr NoteRegistry) MergeImport(src NoteRegistry, srcLabel string, seen map[string]string) (NoteRegistry, error) {
	result := NoteRegistry{Functions: make([]UserDefinedNote, len(nr.Functions))}
	copy(result.Functions, nr.Functions)
	for _, n := range src.Functions {
		if prev, conflict := seen[n.Name]; conflict {
			return NoteRegistry{}, fmt.Errorf(
				"note conflict: %q declared in both %s and %s — rename one or declare a local override in the Katalog's notes: block",
				n.Name, prev, srcLabel,
			)
		}
		seen[n.Name] = srcLabel
		result.Functions = append(result.Functions, n)
	}
	return result, nil
}

// IsEmpty reports whether the registry contains no notes.
func (nr NoteRegistry) IsEmpty() bool {
	return len(nr.Functions) == 0
}
