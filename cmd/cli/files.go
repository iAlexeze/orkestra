package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/orkspace/orkestra/pkg/registry"
)

const (
	fileKatalog    = "katalog.yaml"
	fileKomposer   = "komposer.yaml"
	fileE2e        = "e2e.yaml"
	fileSimulate   = "simulate.yaml"
	fileCrd        = "crd.yaml"
	fileCr         = "cr.yaml"
	fileReadMe     = "README.md"
	fileMakeFile   = "Makefile"
	fileDockerfile = "Dockerfile"
	fileValues     = "values.yaml"
)

// resolveKatalogPaths resolves the katalog file paths in the following order:
//
//  1. Explicit CLI paths (highest priority)
//  2. Default file paths in the working directory (katalog.yaml, komposer.yaml, etc.)
//  3. Paths defined in the Konfig (kfg.Katalog().Paths())
//
// Returns an error if no katalog file can be resolved.
func resolveKatalogPaths(cliPaths []string) ([]string, error) {
	cfgPaths := kfg.Katalog().Paths()

	// 1. CLI-provided paths — convert to absolute so all downstream relative
	// path resolutions (crdFile, crFiles, setup, Komposer imports) use the
	// file's directory as the base, not the current working directory.
	if len(cliPaths) > 0 {
		abs := make([]string, len(cliPaths))
		for i, p := range cliPaths {
			if a, err := filepath.Abs(p); err == nil {
				abs[i] = a
			} else {
				abs[i] = p
			}
		}
		return abs, nil
	}

	// 2. Default file paths
	if defaults := defaultFilePaths(); len(defaults) > 0 {
		return defaults, nil
	}

	// 3. Config-defined paths
	if len(cfgPaths) > 0 {
		return cfgPaths, nil
	}

	return nil, fmt.Errorf(errNoKatalog)
}

// defaultFilePaths returns the default katalog file if one exists in the
// current directory and no -f flag was provided. Tries katalog.yaml first,
// then komposer.yaml — the same precedence as Docker's Dockerfile / compose.yaml.
func defaultFilePaths() []string {
	for _, name := range []string{fileKatalog, fileKomposer} {
		if _, err := os.Stat(name); err == nil {
			return []string{name}
		}
	}
	return nil
}

const errNoKatalog = "no katalog.yaml or komposer.yaml found in current directory\n" +
	"pass -f <file> or create one with ork init"

// resolveOCIRunPath resolves an OCI pattern reference to a local katalog or
// komposer path, pulling to the cache if necessary. Only used by ork run in
// dev builds — not available in production runtime builds.
func resolveOCIRunPath(ctx context.Context, ref string, useKomposer, refresh bool) (string, error) {
	r, err := registry.ResolveForKind(ref, registry.KatalogKind)
	if err != nil {
		return "", fmt.Errorf("invalid reference: %w", err)
	}

	if !r.IsCached() || refresh {
		client, err := registry.NewClient()
		if err != nil {
			return "", fmt.Errorf("initializing registry client: %w", err)
		}
		fmt.Printf("Pulling %s\n  → %s\n", r.ShortName(), r.String())
		spin := StartSpinner("Downloading...")
		if _, err := client.Pull(ctx, r, refresh); err != nil {
			spin.Failure()
			return "", fmt.Errorf("pull failed: %w", err)
		}
		spin.Stop()
		fmt.Printf("  %s Cached\n", successMark())
	}

	cacheDir, err := r.CachePath()
	if err != nil {
		return "", err
	}

	target := fileKatalog
	if useKomposer {
		target = fileKomposer
	}
	p := filepath.Join(cacheDir, target)
	if !fileExists(p) {
		return "", fmt.Errorf("%s not found in cached pattern %s", target, r.ShortName())
	}
	return p, nil
}

// fileExists reports whether a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// resolveKatalogFile resolves a single katalog file path from a CLI flag value.
// If flagValue is empty it falls back to defaultFilePaths(). The resolved path
// is always returned as an absolute path.
func resolveKatalogFile(flagValue string) (string, error) {
	if flagValue == "" {
		if d := defaultFilePaths(); len(d) > 0 {
			flagValue = d[0]
		}
	}
	if flagValue == "" {
		return "", fmt.Errorf(errNoKatalog)
	}
	if abs, err := filepath.Abs(flagValue); err == nil {
		return abs, nil
	}
	return flagValue, nil
}
