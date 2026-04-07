// pkg/merger/parse.go
package merger

import (
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/pkg/konfig"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
)

// parseKatalogDoc parses a single YAML document and returns a KatalogFile
// if it is a valid Katalog or Komposer. Returns (nil, nil) for any other
// document — not an error, just not a document we care about.
//
// Hard errors only when the document looks like a Katalog/Komposer
// (passes the fast string check) but fails to parse or has an unsupported
// apiVersion. Silently skipping malformed documents would hide user mistakes.
func parseKatalogDoc(doc []byte, source string) (*orktypes.KatalogFile, error) {
	// Fast path — skip documents that contain neither Katalog nor Komposer kind.
	// This avoids full YAML parsing for every non-relevant template in a Helm manifest.
	if !containsValidKind(doc) {
		return nil, nil
	}

	var katalog orktypes.KatalogFile
	if err := utils.StrictUnmarshal(doc, &katalog); err != nil {
		return nil, fmt.Errorf("parsing %q: %w", source, err)
	}

	// Kind must be Katalog or Komposer — anything else is silently skipped.
	// This handles YAML files that happen to contain the word "Katalog" in a comment.
	if !konfig.IsValidDocumentKind(katalog.Kind) {
		return nil, nil
	}

	// apiVersion must match a supported version — hard error with guidance.
	if !konfig.IsValidApiVersion(katalog.APIVersion) {
		return nil, fmt.Errorf(
			"%q: unsupported apiVersion %q — supported: %v",
			source, katalog.APIVersion, konfig.ApiVersions(),
		)
	}

	// Name is required irrespective of kind
	if katalog.Metadata.Name == "" {
		return nil, fmt.Errorf("%q: missing metadata.name", source)
	}

	return &katalog, nil
}

// containsValidKind is a fast string check before committing to a full YAML parse.
// Returns true if the document contains either "kind: Katalog" or "kind: Komposer".
func containsValidKind(doc []byte) bool {
	s := string(doc)
	return strings.Contains(s, fmt.Sprintf("kind: %s", konfig.KatalogKind())) ||
		strings.Contains(s, fmt.Sprintf("kind: %s", konfig.KomposerKind()))
}

// Export
var ParseKatalogDoc = parseKatalogDoc
