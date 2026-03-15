package generate

import (
	"fmt"
	"path/filepath"
	"text/template"
	"time"

	"github.com/ialexeze/orkestra/pkg/utils"
)

var (
	dashboardTemplate = template.Must(
		template.ParseFiles("pkg/generate/templates/dashboard.tmpl"),
	)
)

func Dashboards(katalogPath string, dryRun bool) error {
	data, err := utils.LoadFile(katalogPath)
	if err != nil {
		return fmt.Errorf("loading katalog from %q: %w", katalogPath, err)
	}

	kat, err := parseKatalog(data)
	if err != nil {
		return err
	}

	var crds []CRDMeta
	for _, crd := range kat.Spec.CRDs {
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
