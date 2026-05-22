//go:build !runtime && !gateway

package cli

import "fmt"

// resolveKatalogPaths resolves the katalog file paths in the following order:
//
//  1. Explicit CLI paths (highest priority)
//  2. Default file paths in the working directory (katalog.yaml, komposer.yaml, etc.)
//  3. Paths defined in the Konfig (kfg.Katalog().Paths())
//
// Returns an error if no katalog file can be resolved.
func resolveKatalogPaths(cliPaths, cfgPaths []string) ([]string, error) {
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
