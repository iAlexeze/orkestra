package plan

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	"reflect"
)

// KatalogDiff holds the structured difference between two Katalogs.
type KatalogDiff struct {
	AddedCRDs   []string
	RemovedCRDs []string
	ChangedCRDs []CRDDiff
}

type CRDDiff struct {
	Name    string
	Changes []FieldChange
}

type FieldChange struct {
	Path string
	From string // "" means new
	To   string // "" means removed
}

func (d *KatalogDiff) Empty() bool {
	return len(d.AddedCRDs) == 0 &&
		len(d.RemovedCRDs) == 0 &&
		len(d.ChangedCRDs) == 0
}

func (d *KatalogDiff) Print() {
	for _, name := range d.AddedCRDs {
		fmt.Printf("  %s CRD '%s'  (new)\n", utils.Green("+"), name)
	}
	for _, name := range d.RemovedCRDs {
		fmt.Printf("  %s CRD '%s'  (removed)\n", utils.Red("-"), name)
	}
	for _, crd := range d.ChangedCRDs {
		fmt.Printf("\n  CRD '%s':\n", crd.Name)
		for _, ch := range crd.Changes {
			if ch.From == "" {
				fmt.Printf("    %s %s  (new)\n", utils.Green("+"), ch.Path)
			} else if ch.To == "" {
				fmt.Printf("    %s %s  (removed)\n", utils.Red("-"), ch.Path)
			} else {
				fmt.Printf("    %s %s:  %s → %s\n",
					utils.Yellow("~"), ch.Path,
					utils.Red(ch.From), utils.Green(ch.To))
			}
		}
	}
	fmt.Println()
}

// ComputeKatalogDiff computes the structured diff between two Katalogs.
func ComputeKatalogDiff(from, to *katalog.Katalog) *KatalogDiff {
	diff := &KatalogDiff{}

	fromCRDs := from.CRDNames()
	toCRDs := to.CRDNames()

	fromSet := makeSet(fromCRDs)
	toSet := makeSet(toCRDs)

	for _, name := range toCRDs {
		if !fromSet[name] {
			diff.AddedCRDs = append(diff.AddedCRDs, name)
		}
	}
	for _, name := range fromCRDs {
		if !toSet[name] {
			diff.RemovedCRDs = append(diff.RemovedCRDs, name)
		}
	}

	// Changed CRDs — compare fields
	for _, name := range fromCRDs {
		if !toSet[name] {
			continue
		}
		fromCRD, _ := from.CRDEntry(name)
		toCRD, _ := to.CRDEntry(name)
		changes := diffCRDEntry(name, &fromCRD, &toCRD)
		if len(changes) > 0 {
			diff.ChangedCRDs = append(diff.ChangedCRDs, CRDDiff{
				Name:    name,
				Changes: changes,
			})
		}
	}

	return diff
}

func diffCRDEntry(_ string, from, to *orktypes.CRDEntry) []FieldChange {
	var changes []FieldChange

	// metadata
	if from.Name != to.Name {
		changes = append(changes, FieldChange{
			Path: "name",
			From: fmt.Sprint(from.Name),
			To:   fmt.Sprint(to.Name),
		})
	}
	if from.Namespace != to.Namespace {
		changes = append(changes, FieldChange{
			Path: "namespace",
			From: fmt.Sprint(from.Namespace),
			To:   fmt.Sprint(to.Namespace),
		})
	}
	if from.Description != to.Description {
		changes = append(changes, FieldChange{
			Path: "description",
			From: fmt.Sprint(from.Description),
			To:   fmt.Sprint(to.Description),
		})
	}

	// apitypes
	if !reflect.DeepEqual(from.APITypes, to.APITypes) {
		changes = append(changes, FieldChange{
			Path: "apiTypes",
			From: fmt.Sprint(from.APITypes),
			To:   fmt.Sprint(to.APITypes),
		})
	}

	// spec
	if from.Workers != to.Workers {
		changes = append(changes, FieldChange{
			Path: "workers",
			From: fmt.Sprint(from.Workers),
			To:   fmt.Sprint(to.Workers),
		})
	}
	if from.Resync != to.Resync {
		changes = append(changes, FieldChange{
			Path: "resync",
			From: from.Resync.String(),
			To:   to.Resync.String(),
		})
	}
	if from.CRDFile != to.CRDFile {
		changes = append(changes, FieldChange{
			Path: "crdFile",
			From: from.CRDFile,
			To:   to.CRDFile,
		})
	}
	if !reflect.DeepEqual(from.Queue, to.Queue) {
		changes = append(changes, FieldChange{
			Path: "queue",
			From: fmt.Sprint(from.Queue),
			To:   fmt.Sprint(to.Queue),
		})
	}
	if !reflect.DeepEqual(from.DependsOn, to.DependsOn) {
		changes = append(changes, FieldChange{
			Path: "dependsOn",
			From: fmt.Sprint(from.DependsOn),
			To:   fmt.Sprint(to.DependsOn),
		})
	}

	// namespace restrictions
	if !reflect.DeepEqual(from.AllowedNamespaces, to.AllowedNamespaces) {
		changes = append(changes, FieldChange{
			Path: "allowedNamespaces",
			From: fmt.Sprint(from.AllowedNamespaces),
			To:   fmt.Sprint(to.AllowedNamespaces),
		})
	}
	if !reflect.DeepEqual(from.RestrictedNamespaces, to.RestrictedNamespaces) {
		changes = append(changes, FieldChange{
			Path: "restrictedNamespaces",
			From: fmt.Sprint(from.RestrictedNamespaces),
			To:   fmt.Sprint(to.RestrictedNamespaces),
		})
	}

	// Selectors
	if !reflect.DeepEqual(from.LabelSelector, to.LabelSelector) {
		changes = append(changes, FieldChange{
			Path: "labelSelector",
			From: fmt.Sprint(from.LabelSelector),
			To:   fmt.Sprint(to.LabelSelector),
		})
	}
	if !reflect.DeepEqual(from.FieldSelector, to.FieldSelector) {
		changes = append(changes, FieldChange{
			Path: "fieldSelector",
			From: fmt.Sprint(from.FieldSelector),
			To:   fmt.Sprint(to.FieldSelector),
		})
	}

	// validation, mutation, conversion
	if !reflect.DeepEqual(from.Validation, to.Validation) {
		changes = append(changes, FieldChange{
			Path: "validation",
			From: fmt.Sprint(from.Validation),
			To:   fmt.Sprint(to.Validation),
		})
	}
	if !reflect.DeepEqual(from.Mutation, to.Mutation) {
		changes = append(changes, FieldChange{
			Path: "mutation",
			From: fmt.Sprint(from.Mutation),
			To:   fmt.Sprint(to.Mutation),
		})
	}
	if !reflect.DeepEqual(from.Conversion, to.Conversion) {
		changes = append(changes, FieldChange{
			Path: "conversion",
			From: fmt.Sprint(from.Conversion),
			To:   fmt.Sprint(to.Conversion),
		})
	}

	// Normalize
	if !reflect.DeepEqual(from.Normalize, to.Normalize) {
		changes = append(changes, FieldChange{
			Path: "normalize",
			From: fmt.Sprint(from.Normalize),
			To:   fmt.Sprint(to.Normalize),
		})
	}

	// operatorBox resource counts
	if from.OperatorBox.OnCreate != nil && to.OperatorBox.OnCreate != nil {
		fromCreate := resourceCounts(from.OperatorBox.OnCreate)
		toCreate := resourceCounts(to.OperatorBox.OnCreate)
		for _, change := range diffResourceCounts("operatorBox.onCreate", fromCreate, toCreate) {
			changes = append(changes, change)
		}
	}
	if from.OperatorBox.OnReconcile != nil && to.OperatorBox.OnReconcile != nil {
		fromReconcile := resourceCounts(from.OperatorBox.OnReconcile)
		toReconcile := resourceCounts(to.OperatorBox.OnReconcile)
		for _, change := range diffResourceCounts("operatorBox.onReconcile", fromReconcile, toReconcile) {
			changes = append(changes, change)
		}
	}

	return changes
}

func resourceCounts(ht *orktypes.HookTemplates) map[string]int {
	if ht == nil {
		return nil
	}
	return map[string]int{
		"deployments":            len(ht.Deployments),
		"services":               len(ht.Services),
		"secrets":                len(ht.Secrets),
		"configMaps":             len(ht.ConfigMaps),
		"serviceAccounts":        len(ht.ServiceAccounts),
		"statefulSets":           len(ht.StatefulSets),
		"ingresses":              len(ht.Ingresses),
		"jobs":                   len(ht.Jobs),
		"cronJobs":               len(ht.CronJobs),
		"persistentVolumes":      len(ht.PersistentVolumes),
		"persistentVolumeClaims": len(ht.PersistentVolumeClaims),
		"hpa":                    len(ht.HorizontalPodAutoscalers),
		"pdb":                    len(ht.PodDisruptionBudgets),
		"namespaces":             len(ht.Namespaces),
		"roles":                  len(ht.Roles),
		"roleBindings":           len(ht.RoleBindings),
		"custom":                 len(ht.CustomResource),
		"replicaSets":            len(ht.ReplicaSets),
		"pods":                   len(ht.Pods),
	}
}

func diffResourceCounts(prefix string, from, to map[string]int) []FieldChange {
	var changes []FieldChange
	for key, toVal := range to {
		fromVal := from[key]
		if fromVal != toVal {
			changes = append(changes, FieldChange{
				Path: strings.Join([]string{prefix, key}, "."),
				From: fmt.Sprint(fromVal),
				To:   fmt.Sprint(toVal),
			})
		}
	}
	return changes
}

func makeSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, v := range items {
		s[v] = true
	}
	return s
}
