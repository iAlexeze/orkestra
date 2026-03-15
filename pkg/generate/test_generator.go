package generate

import (
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/ialexeze/orkestra/pkg/utils"
)

var testTemplate = template.Must(
	template.ParseFiles("pkg/generate/templates/test_scaffold.tmpl"),
)

func Tests(katalogPath string, dryRun bool) error {
	data, err := utils.LoadFile(katalogPath)
	if err != nil {
		return fmt.Errorf("loading katalog: %w", err)
	}

	kat, err := parseKatalog(data)
	if err != nil {
		return err
	}

	for _, crd := range kat.Spec.CRDs {
		if !crd.Enabled {
			continue
		}

		m := CRDMeta{
			Name: crd.Name,
			Kind: crd.APITypes.Kind,
		}
		m.API.Object = crd.APITypes.Object
		m.API.Alias = crd.APITypes.Alias
		m.API.Import = crd.APITypes.Location

		out := filepath.Join(TestDir, crd.Name+"_test.go")
		if err := renderTemplateToFile(testTemplate, m, out, true, dryRun); err != nil {
			return err
		}
	}

	return nil
}
