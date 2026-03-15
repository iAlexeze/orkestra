// pkg/generate/helper.go
package generate

import (
	"fmt"

	"bytes"
	"go/format"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"text/template"
)

func parseKatalog(data []byte) (*generateKatalog, error) {
	var kat generateKatalog
	if err := yaml.Unmarshal(data, &kat); err != nil {
		return nil, fmt.Errorf("parsing katalog: %w", err)
	}
	if len(kat.Spec.CRDs) == 0 {
		return nil, fmt.Errorf("katalog has no CRDs defined")
	}
	return &kat, nil
}

func renderTemplateToFile(tmpl *template.Template, data any, outPath string, gofmt bool, dryRun bool) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("rendering template for %q: %w", outPath, err)
	}

	content := buf.Bytes()
	formatted, err := format.Source(content)
	if gofmt {
		if err != nil {
			return fmt.Errorf("gofmt failed for %q:\n%s\n\nerror: %w", outPath, buf.String(), err)
		}
		content = formatted
	}

	if dryRun {
		fmt.Println("--- dry run: would write to", filepath.Join(RuntimePackage, RegistryFile), "---")
		fmt.Println(string(formatted))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", filepath.Dir(outPath), err)
	}

	if err := os.WriteFile(outPath, content, 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", outPath, err)
	}

	return nil
}
