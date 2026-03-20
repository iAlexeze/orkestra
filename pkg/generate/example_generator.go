package generate

import (
	"path/filepath"
	"text/template"
	"time"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

var exampleTemplate = template.Must(
	template.ParseFiles("pkg/generate/templates/example_manifest.tmpl"),
)

func Examples(crds []orktypes.CRDEntry, dryRun bool) error {
	now := time.Now().UTC()

	for _, crd := range crds {
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
