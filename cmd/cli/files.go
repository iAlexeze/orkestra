package cli

import (
	"fmt"
	"os"
)

const (
	fileKatalog  = "katalog.yaml"
	fileKomposer = "komposer.yaml"
	fileE2e      = "e2e.yaml"
	fileCrd      = "crd.yaml"
	fileCr       = "cr.yaml"
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

	// 1. CLI-provided paths
	if len(cliPaths) > 0 {
		return cliPaths, nil
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
