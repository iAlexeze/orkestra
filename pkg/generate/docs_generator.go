package generate

import (
	"fmt"
	"path/filepath"
	"text/template"
	"time"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

var (
	crdDocTemplate = template.Must(
		template.ParseFS(templateFS, "templates/crd_doc.tmpl"),
	)

	docsIndexTemplate = template.Must(
		template.ParseFS(templateFS, "templates/docs_index.tmpl"),
	)
)

func Docs(c []orktypes.CRDEntry, dryRun bool) error {
	var crds []CRDMeta
	for _, crd := range c {
		if !crd.Enabled {
			continue
		}

		m := CRDMeta{
			Name:        crd.Name,
			Description: crd.Description,
			Group:       crd.APITypes.Group,
			Version:     crd.APITypes.Version,
			Kind:        crd.APITypes.Kind,
			Plural:      crd.APITypes.Plural,
			Namespaced:  crd.Namespaced,
			Namespace:   crd.Namespace,
			Workers:     crd.Workers,
			Resync:      crd.Resync.String(),
			DependsOn:   crd.DependsOn,
		}

		m.Queue.MaxQueueDepth = crd.Queue.MaxQueueDepth
		m.Queue.Default = crd.Queue.Default

		m.API.Object = crd.APITypes.Object
		m.API.List = crd.APITypes.List
		m.API.Alias = crd.APITypes.Alias

		m.Reconciler.Default = crd.ReconcilerConfig.Default
		if crd.ReconcilerConfig.Constructor != nil {
			m.Reconciler.Function = crd.ReconcilerConfig.ConstructorDecl.Function
		}

		crds = append(crds, m)
	}

	if len(crds) == 0 {
		return fmt.Errorf("no enabled CRDs found — cannot generate docs")
	}

	now := time.Now().UTC()

	// index
	if err := renderTemplateToFile(
		docsIndexTemplate,
		DocsTemplateData{Timestamp: now, CRDs: crds},
		filepath.Join(DocsDir, "index.md"),
		false,
		dryRun,
	); err != nil {
		return err
	}

	// per‑CRD docs
	for _, crd := range crds {
		out := filepath.Join(DocsDir, "crds", crd.Name+".md")
		if err := renderTemplateToFile(
			crdDocTemplate,
			DocsTemplateData{Timestamp: now, CRD: crd},
			out,
			false,
			dryRun,
		); err != nil {
			return err
		}
	}

	return nil
}
