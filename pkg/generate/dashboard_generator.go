package generate

import (
	"fmt"
	"path/filepath"
	"text/template"
	"time"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

var (
	dashboardTemplate = template.Must(
		template.ParseFiles("pkg/generate/templates/dashboard.tmpl"),
	)
)

func Dashboards(c []orktypes.CRDEntry, dryRun bool) error {
	var crds []CRDMeta
	for _, crd := range c {
		if !crd.Enabled {
			continue
		}

		crds = append(crds, CRDMeta{
			Name:        crd.Name,
			Description: crd.Description,
			Group:       crd.APITypes.Group,
			Version:     crd.APITypes.Version,
			Kind:        crd.APITypes.Kind,
		})
	}

	if len(crds) == 0 {
		return fmt.Errorf("no enabled CRDs found — cannot generate dashboards")
	}

	now := time.Now().UTC()

	for _, crd := range crds {
		out := filepath.Join(DashDir, crd.Name+".json")
		if err := renderTemplateToFile(
			dashboardTemplate,
			DashboardTemplateData{Timestamp: now, CRD: crd},
			out,
			false,
			dryRun,
		); err != nil {
			return err
		}
	}

	return nil
}
