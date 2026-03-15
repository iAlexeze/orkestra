package generate

import (
	"fmt"
	"path/filepath"
	"text/template"
	"time"

	"github.com/ialexeze/orkestra/pkg/utils"
)

var exampleTemplate = template.Must(
	template.ParseFiles("pkg/generate/templates/example_manifest.tmpl"),
)

func Examples(katalogPath string, dryRun bool) error {
	data, err := utils.LoadFile(katalogPath)
	if err != nil {
		return fmt.Errorf("loading katalog: %w", err)
	}

	kat, err := parseKatalog(data)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	for _, crd := range kat.Spec.CRDs {
		if !crd.Enabled {
			continue
		}

		m := DocsTemplateData{
			Timestamp: now,
			CRD: CRDMeta{
				Name:       crd.Name,
				Group:      crd.APITypes.Group,
				Version:    crd.APITypes.Version,
				Kind:       crd.APITypes.Kind,
				Namespaced: crd.Namespaced,
				Namespace:  crd.Namespace,
			},
		}

		out := filepath.Join(ExamplesDir, crd.Name+".yaml")
		if err := renderTemplateToFile(exampleTemplate, m, out, false, dryRun); err != nil {
			return err
		}
	}

	return nil
}
