package generate

import (
	"path/filepath"
	"text/template"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

var testTemplate = template.Must(
	template.ParseFS(templateFS, "templates/test_scaffold.tmpl"),
)

func Tests(crds []orktypes.CRDEntry, dryRun bool) error {
	for _, crd := range crds {
		if !crd.IsEnabled() {
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
