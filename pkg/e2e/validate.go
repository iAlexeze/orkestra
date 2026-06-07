package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"gopkg.in/yaml.v3"
)

// ValidateImports checks that every file listed in imports exists and declares
// kind: E2E. baseDir is the directory that relative paths are resolved against.
// Returns one error per invalid import — callers may print all of them.
func ValidateImports(baseDir string, imports []orktypes.E2EImport) []error {
	var errs []error
	for _, imp := range imports {
		path := imp.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", imp.Path, err))
			continue
		}
		var head struct {
			Kind string `yaml:"kind"`
		}
		if err := yaml.Unmarshal(data, &head); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", imp.Path, err))
			continue
		}
		if head.Kind != "E2E" {
			errs = append(errs, fmt.Errorf("%s: expected kind E2E, got %q", imp.Path, head.Kind))
		}
		if imp.Wait != "" {
			if _, err := time.ParseDuration(imp.Wait); err != nil {
				errs = append(errs, fmt.Errorf("%s: wait %q is not a valid duration: %w", imp.Path, imp.Wait, err))
			}
		}
	}
	return errs
}
