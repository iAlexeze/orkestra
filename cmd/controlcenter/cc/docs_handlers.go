package controlcenter

import (
	"log"
	"net/http"
	"sort"
	"strings"

	ccversion "github.com/orkspace/orkestra-cc/version"
)

// DocsKatalogEntry is a single katalog shown on the docs landing page.
type DocsKatalogEntry struct {
	Name        string
	Description string
	Version     string
	Author      string
	CRDs        []DocsCRDEntry
}

// DocsCRDEntry is a CRD link on the docs landing page.
type DocsCRDEntry struct {
	Name        string
	KatalogName string
	State       string
}

// DocsLandingData is the view-model for the docs landing page.
type DocsLandingData struct {
	Katalogs  []DocsKatalogEntry
	CCVersion string
}

// CRDDocsData is the view-model for a single CRD docs page.
type CRDDocsData struct {
	KatalogName    string
	KatalogVersion string
	CRD            *CRDDetail
	KubectlKind    string
	KubectlGroup   string
	KubectlVersion string
	HasAdmission   bool
	HasConversion  bool
	HasProtection  bool
	HasRBAC        bool
	HasAutoscaler  bool
	HasRollback    bool
	HasProviders   bool
	CCVersion      string
	RuntimeVersion string
}

// handleDocsLanding renders /controlcenter/docs.
func (cc *ControlCenter) handleDocsLanding(w http.ResponseWriter, r *http.Request) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	var entries []DocsKatalogEntry
	for _, inst := range cc.instances {
		if inst.Katalog == nil {
			continue
		}
		kat := inst.Katalog
		var crdEntries []DocsCRDEntry
		for _, c := range kat.CRDs {
			crdEntries = append(crdEntries, DocsCRDEntry{
				Name:        c.Name,
				KatalogName: kat.Name,
				State:       c.State,
			})
		}
		sort.Slice(crdEntries, func(i, j int) bool {
			return crdEntries[i].Name < crdEntries[j].Name
		})
		entries = append(entries, DocsKatalogEntry{
			Name:        kat.Name,
			Description: kat.Description,
			Version:     kat.Version,
			Author:      kat.Author,
			CRDs:        crdEntries,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	cc.renderTemplate(w, "docs.html", DocsLandingData{
		Katalogs:  entries,
		CCVersion: ccversion.Short(),
	})
}

// handleCRDDocs renders /controlcenter/docs/{katalog}/{crd}.
func (cc *ControlCenter) handleCRDDocs(w http.ResponseWriter, r *http.Request, katalogName, crdName string) {
	inst, ok := cc.instanceByKatalogName(katalogName)
	if !ok {
		cc.handleNotFound(w, r)
		return
	}

	crd, err := inst.Client.FetchCRDDetail(crdName)
	if err != nil {
		log.Printf("WARN: docs fetch CRD %s: %v", crdName, err)
		crd = &CRDDetail{
			Name:        crdName,
			State:       "offline",
			Description: "Unable to connect to Orkestra runtime",
			GVK:         "unknown",
			LastError:   err.Error(),
		}
	}

	kubectlKind, kubectlGroup, kubectlVersion := parseGVK(crd.GVK)

	cc.renderTemplate(w, "crd_docs.html", CRDDocsData{
		KatalogName:    katalogName,
		KatalogVersion: inst.Katalog.Version,
		CRD:            crd,
		KubectlKind:    kubectlKind,
		KubectlGroup:   kubectlGroup,
		KubectlVersion: kubectlVersion,
		HasAdmission:   crd.Admission != nil && crd.Admission.WebhooksEnabled,
		HasConversion:  crd.Conversion != nil && crd.Conversion.Enabled,
		HasProtection: (crd.DeletionProtection != nil && crd.DeletionProtection.Enabled) ||
			(crd.NamespaceProtection != nil && crd.NamespaceProtection.Enabled),
		HasRBAC:        crd.RBAC.TotalRules > 0,
		HasAutoscaler:  crd.AutoscalerEnabled && crd.AutoscalerWorkers != nil,
		HasRollback:    crd.Rollback != nil,
		HasProviders:   len(crd.Providers) > 0,
		CCVersion:      ccversion.Short(),
		RuntimeVersion: inst.Katalog.RuntimeVersion,
	})
}

// parseGVK splits "group/version, Kind=Foo" into (kind, group, version).
func parseGVK(gvk string) (kind, group, version string) {
	if gvk == "" || gvk == "unknown" {
		return "Unknown", "", ""
	}
	parts := strings.SplitN(gvk, ", Kind=", 2)
	if len(parts) == 2 {
		kind = parts[1]
		gv := parts[0]
		if idx := strings.LastIndex(gv, "/"); idx >= 0 {
			group = gv[:idx]
			version = gv[idx+1:]
		} else {
			version = gv
		}
	} else {
		kind = gvk
	}
	return
}
