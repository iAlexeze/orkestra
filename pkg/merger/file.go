// pkg/merger/file.go
package merger

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
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
//   valid — map key collision triggers mergeCRDEntry.
//
// Inline duplicates inline:
//   always an error.

// loadKatalogFile parses one file and dispatches to the correct loader
// based on its kind. Returns the deduplicated CRD map for this file tree.
func (m *Merger) loadKatalogFile(path string) (map[string]orktypes.CRDEntry, error) {
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
// Map keys are the CRD names; Name is injected from the key.
func (m *Merger) loadKatalog(path string, doc *orktypes.KatalogFile) (map[string]orktypes.CRDEntry, error) {
	// Guard — sources in a Katalog is a mistake
	if doc.Sources != nil && (len(doc.Sources.Files) > 0 || len(doc.Sources.Helm) > 0) {
		return nil, fmt.Errorf(
			"%q: kind Katalog cannot declare sources — "+
				"use kind: Komposer to compose multiple Katalogs",
			path,
		)
	}

	result := make(map[string]orktypes.CRDEntry, len(doc.Spec.CRDs))

	for name, crd := range doc.Spec.CRDs {
		if name == "" {
			return nil, fmt.Errorf("%q spec.crds: CRD with empty key", path)
		}
		// Duplicate within the same file is impossible — map keys are unique.
		crd.Name = name

		// Merge spec-level restrictions into each CRD (additive).
		crd.RestrictedNamespaces = doc.Security.NamespaceProtection.RestrictedNamespaces.Merge(crd.RestrictedNamespaces)
		crd.AllowedNamespaces = doc.Security.NamespaceProtection.AllowedNamespaces.Merge(crd.AllowedNamespaces)

		result[name] = crd

		logger.Debug().
			Str("crd", name).
			Str("source", path).
			Msg("merger: CRD loaded from Katalog")
	}

	logger.Debug().
		Str("path", path).
		Int("crds", len(result)).
		Msg("merger: Katalog loaded")

	// This is a katalog
	apiMetadata := apiMetadata{
		APIVersion: doc.APIVersion,
		Kind:       doc.Kind,
		Metadata:   doc.Metadata,
	}
	m.apiMetadata = apiMetadata
	m.security = doc.Security
	m.providers = doc.Providers

	return result, nil
}

// loadKomposer resolves sources from a Komposer file and merges all CRDs.
func (m *Merger) loadKomposer(path string, doc *orktypes.KatalogFile) (map[string]orktypes.CRDEntry, error) {
	if doc.Sources == nil && len(doc.Spec.CRDs) == 0 {
		logger.Warn().
			Str("path", path).
			Msg("merger: Komposer has no sources and no inline CRDs — nothing to load")
		return nil, nil
	}

	localSeen := map[string]string{}
	allCRDs := make(map[string]orktypes.CRDEntry)

	// ── Step 1: registry sources ─────────────────────────────────────────────
	if doc.Sources != nil {
		for i, regSrc := range doc.Sources.Registry {
			crds, err := m.loadRegistrySource(regSrc)
			if err != nil {
				return nil, fmt.Errorf("%q sources.registry[%d]: %w", path, i, err)
			}

			for name, crd := range crds {
				srcName := fmt.Sprintf("registry:%d", i)
				if regSrc.URL != "" {
					srcName = "registry:" + regSrc.URL
				}
				if err := checkDuplicate(localSeen, name, srcName); err != nil {
					return nil, fmt.Errorf("%q: %w", path, err)
				}
				localSeen[name] = srcName
				allCRDs[name] = crd
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

			for name, crd := range crds {
				if err := checkDuplicate(localSeen, name, "file:"+resolved); err != nil {
					return nil, fmt.Errorf("%q: %w", path, err)
				}
				localSeen[name] = "file:" + resolved
				allCRDs[name] = crd
			}
		}
		// ── Step 3: helm sources ──────────────────────────────────────────────
		for i, helmSrc := range doc.Sources.Helm {
			crds, err := m.loadHelmSource(helmSrc)
			if err != nil {
				return nil, fmt.Errorf("%q sources.helm[%d]: %w", path, i, err)
			}

			srcName := fmt.Sprintf("helm:%s/%s@%s", helmSrc.Repo, helmSrc.Chart, helmSrc.Version)
			for name, crd := range crds {
				if err := checkDuplicate(localSeen, name, srcName); err != nil {
					return nil, fmt.Errorf("%q: %w", path, err)
				}
				localSeen[name] = srcName
				allCRDs[name] = crd
			}
		}
	}

	// ── Step 4: inline spec.crds — override any source CRD with same name ────
	inlineKey := "inline:" + path
	for name, crd := range doc.Spec.CRDs {
		if name == "" {
			return nil, fmt.Errorf("%q spec.crds: CRD with empty key", path)
		}

		// Duplicate within the same inline block is impossible (map keys).
		// But if same inline key was already recorded, it's a bug.
		if existing, ok := localSeen[name]; ok && existing == inlineKey {
			return nil, fmt.Errorf(
				"%q spec.crds: duplicate CRD %q — each CRD name must be unique",
				path, name,
			)
		}

		crd.Name = name

		// Merge onto source, don't replace
		if base, found := allCRDs[name]; found {
			allCRDs[name] = mergeCRDEntry(base, crd)
			logger.Debug().
				Str("crd", name).
				Str("source", inlineKey).
				Msg("merger: inline override merged onto source entry")
		} else {
			allCRDs[name] = crd
			logger.Debug().
				Str("crd", name).
				Str("source", inlineKey).
				Msg("merger: new CRD from inline spec.crds")
		}

		localSeen[name] = inlineKey
	}
	// Merge Komposer-level restrictions into every CRD (additive).
	if len(doc.Security.NamespaceProtection.RestrictedNamespaces) > 0 || len(doc.Security.NamespaceProtection.AllowedNamespaces) > 0 {
		for name, crd := range allCRDs {
		crd.RestrictedNamespaces = doc.Security.NamespaceProtection.RestrictedNamespaces.Merge(crd.RestrictedNamespaces)
		crd.AllowedNamespaces = doc.Security.NamespaceProtection.AllowedNamespaces.Merge(crd.AllowedNamespaces)
			allCRDs[name] = crd
		}
	}

	logger.Debug().
		Str("path", path).
		Int("crds", len(allCRDs)).
		Msg("merger: Komposer loaded")

	// This is a komposer
	apiMetadata := apiMetadata{
		APIVersion: doc.APIVersion,
		Kind:       doc.Kind,
		Metadata:   doc.Metadata,
	}

	m.apiMetadata = apiMetadata
	m.security = doc.Security
	m.providers = doc.Providers
	return allCRDs, nil
}
