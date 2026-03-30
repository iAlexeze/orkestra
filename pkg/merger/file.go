// pkg/merger/file.go
package merger

import (
	"fmt"

	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
)

// ── Merge rules ───────────────────────────────────────────────────────────────
//
// Katalog  (kind: Katalog)
//   Declares CRDs directly in spec.crds.
//   Must NOT declare sources — sources are a Komposer concern.
//   Error if sources block is present.
//
// Komposer (kind: Komposer)
//   Composes Katalogs from multiple sources (files, helm).
//   May declare inline spec.crds as overrides — merged last, win on conflict.
//   Sources are resolved recursively. Each source must be a Katalog.
//   A Komposer cannot reference another Komposer as a source.
//
// Within one file's source tree:
//   localSeen catches duplicates across sources and within inline block.
//
// Across entry point files:
//   seen in Merge() catches duplicates.
//
// Inline overrides source:
//   valid — removeCRD handles it silently.
//
// Inline duplicates inline:
//   always an error.

// loadKatalogFile parses one file and dispatches to the correct loader
// based on its kind. Returns the deduplicated CRD list for this file tree.
func (m *Merger) loadKatalogFile(path string) ([]orktypes.CRDEntry, error) {
	data, err := utils.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", path, err)
	}

	doc, err := parseKatalogDoc(data, path)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		logger.Debug().
			Str("path", path).
			Msg("merger: skipping — not a valid Katalog or Komposer document")
		return nil, nil
	}

	// Dispatch on Kind — Katalog and Komposer are handled differently
	switch doc.Kind {
	case konfig.KatalogKind():
		return m.loadKatalog(path, doc)
	case konfig.KomposerKind():
		return m.loadKomposer(path, doc)
	default:
		// Should not reach here — parseKatalogDoc already validates Kind
		return nil, fmt.Errorf("%q: unexpected kind %q", path, doc.Kind)
	}
}

// loadKatalog reads CRD definitions from a Katalog file.
//
// Katalogs declare CRDs in spec.crds only.
// A Katalog must NOT declare sources — sources are a Komposer concern.
// If a sources block is present, it is an error: the user should use
// kind: Komposer instead.
func (m *Merger) loadKatalog(path string, doc *orktypes.KatalogFile) ([]orktypes.CRDEntry, error) {
	// Guard — sources in a Katalog is a mistake
	if doc.Sources != nil && (len(doc.Sources.Files) > 0 || len(doc.Sources.Helm) > 0) {
		return nil, fmt.Errorf(
			"%q: kind Katalog cannot declare sources — "+
				"use kind: Komposer to compose multiple Katalogs",
			path,
		)
	}

	var allCRDs []orktypes.CRDEntry
	localSeen := map[string]string{}

	for _, crd := range doc.Spec.CRDs {
		if crd.Name == "" {
			return nil, fmt.Errorf("%q spec.crds: CRD with no name", path)
		}

		inlineKey := "inline:" + path

		if existing, ok := localSeen[crd.Name]; ok && existing == inlineKey {
			return nil, fmt.Errorf(
				"%q spec.crds: duplicate CRD %q — each CRD name must be unique",
				path, crd.Name,
			)
		}

		localSeen[crd.Name] = inlineKey
		allCRDs = append(allCRDs, crd)

		logger.Debug().
			Str("crd", crd.Name).
			Str("source", path).
			Msg("merger: CRD loaded from Katalog")
	}

	logger.Debug().
		Str("path", path).
		Int("crds", len(allCRDs)).
		Msg("merger: Katalog loaded")

	m.metadata = doc.Metadata // This is a katalog
	return allCRDs, nil
}

// loadKomposer resolves sources from a Komposer file and merges all CRDs.
//
// Komposers compose Katalogs — they do not declare CRDs directly except
// as inline overrides in spec.crds (which win over source definitions).
//
// Source resolution order:
//  1. File sources — each must be a Katalog (not another Komposer)
//  2. Helm sources — rendered chart must contain a Katalog template
//  3. Inline spec.crds — merged last, override any source CRD with same name
func (m *Merger) loadKomposer(path string, doc *orktypes.KatalogFile) ([]orktypes.CRDEntry, error) {
	if doc.Sources == nil && len(doc.Spec.CRDs) == 0 {
		logger.Warn().
			Str("path", path).
			Msg("merger: Komposer has no sources and no inline CRDs — nothing to load")
		return nil, nil
	}

	localSeen := map[string]string{}
	var allCRDs []orktypes.CRDEntry

	// ── Step 1: registry sources ─────────────────────────────────────────────
	if doc.Sources != nil {
		for i, regSrc := range doc.Sources.Registry {
			crds, err := m.loadRegistrySource(regSrc)
			if err != nil {
				return nil, fmt.Errorf("%q sources.registry[%d]: %w", path, i, err)
			}

			for _, crd := range crds {
				srcName := fmt.Sprintf("registry:%d", i)
				if regSrc.URL != "" {
					srcName = "registry:" + regSrc.URL
				}
				if err := checkDuplicate(localSeen, crd.Name, srcName); err != nil {
					return nil, fmt.Errorf("%q: %w", path, err)
				}
				localSeen[crd.Name] = srcName
				allCRDs = append(allCRDs, crd)
			}
		}
	}

	// ── Step 2: file sources ──────────────────────────────────────────────────
	if doc.Sources != nil {
		for _, fileSrc := range doc.Sources.Files {

			// Resolve environment variable in the URL if needed
			resolved, err := resolveEnvVar(fileSrc.URL)
			if err != nil {
				return nil, fmt.Errorf("%q sources.files: %w", path, err)
			}

			// Resolve authentication credentials from environment variables
			auth, err := fileSrc.Auth.Resolve()
			if err != nil {
				return nil, fmt.Errorf("%q sources.files[%q]: auth: %w", path, resolved, err)
			}

			// Load the file — must be a Katalog, not another Komposer
			crds, err := m.loadSourceFileWithAuth(path, resolved, auth)
			if err != nil {
				return nil, fmt.Errorf("%q sources.files[%q]: %w", path, resolved, err)
			}

			for _, crd := range crds {
				if err := checkDuplicate(localSeen, crd.Name, "file:"+resolved); err != nil {
					return nil, fmt.Errorf("%q: %w", path, err)
				}
				localSeen[crd.Name] = "file:" + resolved
				allCRDs = append(allCRDs, crd)
			}
		}
		// ── Step 3: helm sources ──────────────────────────────────────────────
		for i, helmSrc := range doc.Sources.Helm {
			crds, err := m.loadHelmSource(helmSrc)
			if err != nil {
				return nil, fmt.Errorf("%q sources.helm[%d]: %w", path, i, err)
			}

			srcName := fmt.Sprintf("helm:%s/%s@%s", helmSrc.Repo, helmSrc.Chart, helmSrc.Version)
			for _, crd := range crds {
				if err := checkDuplicate(localSeen, crd.Name, srcName); err != nil {
					return nil, fmt.Errorf("%q: %w", path, err)
				}
				localSeen[crd.Name] = srcName
				allCRDs = append(allCRDs, crd)
			}
		}
	}

	// ── Step 4: inline spec.crds — override any source CRD with same name ────
	for _, crd := range doc.Spec.CRDs {
		if crd.Name == "" {
			return nil, fmt.Errorf("%q spec.crds: CRD with no name", path)
		}

		inlineKey := "inline:" + path

		// Duplicate within the same inline block — always an error
		if existing, ok := localSeen[crd.Name]; ok && existing == inlineKey {
			return nil, fmt.Errorf(
				"%q spec.crds: duplicate CRD %q — each CRD name must be unique",
				path, crd.Name,
			)
		}

		// Override a source CRD with the same name — valid
		allCRDs = removeCRD(allCRDs, crd.Name)
		localSeen[crd.Name] = inlineKey
		allCRDs = append(allCRDs, crd)

		logger.Debug().
			Str("crd", crd.Name).
			Str("source", "inline:"+path).
			Msg("merger: inline CRD overrides source")
	}

	logger.Debug().
		Str("path", path).
		Int("crds", len(allCRDs)).
		Msg("merger: Komposer loaded")

	m.metadata = doc.Metadata // This is a komposer
	return allCRDs, nil
}

// loadSourceFile loads a file that is referenced from a Komposer sources block.
// The file MUST be a Katalog — a Komposer cannot source another Komposer.
// This prevents deep composition chains that are hard to reason about.
func (m *Merger) loadSourceFile(komposerPath, sourcePath string) ([]orktypes.CRDEntry, error) {
	data, err := utils.LoadFile(sourcePath)
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
