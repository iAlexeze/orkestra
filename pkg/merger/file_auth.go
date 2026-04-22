// pkg/merger/file_auth.go
package merger

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// loadSourceFileWithAuth loads a Katalog source file with optional authentication.
// The source must be a Katalog — Komposers cannot source other Komposers.
func (m *Merger) loadSourceFileWithAuth(komposerPath, sourcePath string, auth *utils.FileAuth) (map[string]orktypes.CRDEntry, error) {
	data, err := utils.LoadFileWithAuth(sourcePath, auth)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", sourcePath, err)
	}

	doc, err := parseKatalogDoc(data, sourcePath)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		logger.Debug().
			Str("path", sourcePath).
			Msg("merger: skipping source — not a valid Katalog document")
		return nil, nil
	}

	// Komposer sources must be Katalogs — not other Komposers
	if doc.Kind == konfig.KomposerKind() {
		return nil, fmt.Errorf(
			"%q sources.files[%q]: a Komposer cannot source another Komposer — "+
				"only Katalog files are valid sources",
			komposerPath, sourcePath,
		)
	}

	if doc.Kind != konfig.KatalogKind() {
		return nil, fmt.Errorf(
			"%q sources.files[%q]: expected kind %q, got %q",
			komposerPath, sourcePath, konfig.KatalogKind(), doc.Kind,
		)
	}

	return m.loadKatalog(sourcePath, doc)
}
