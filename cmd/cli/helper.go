//go:build !runtime

package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
)

// ── printTemplateSummary ──────────────────────────────────────────────────────

// printTemplateSummary prints the human-readable default output of `ork template`.
// CRDs are listed in startup order so the user sees the dependency sequence.
func printTemplateSummary(k *katalog.Katalog, crds map[string]orktypes.CRDEntry, startupOrder []string) {
	meta := k.Metadata()

	fmt.Printf("\n%s%sKatalog%s", utils.ColorBold, utils.ColorCyan, utils.ColorReset)
	if meta.Name != "" {
		fmt.Printf(": %s%s%s", utils.ColorBold, meta.Name, utils.ColorReset)
	}
	if meta.Version != "" {
		fmt.Printf(" %s(%s)%s", utils.ColorDim, meta.Version, utils.ColorReset)
	}
	if k.APIVersion != "" {
		fmt.Printf("  %s%s%s", utils.ColorDim, k.APIVersion, utils.ColorReset)
	}
	fmt.Println()

	fmt.Printf("  %d CRD(s) — startup order: %s\n\n",
		len(crds),
		utils.ColorYellow+strings.Join(startupOrder, " → ")+utils.ColorReset,
	)

	for i, name := range startupOrder {
		crd, ok := crds[name]
		if !ok {
			continue
		}
		isLast := i == len(startupOrder)-1
		connector := "├─"
		if isLast {
			connector = "└─"
		}

		gvk := fmt.Sprintf("%s/%s, Kind=%s", crd.APITypes.Group, crd.APITypes.Version, crd.APITypes.Kind)
		fmt.Printf("  %s %s%s%s  %s%s%s\n",
			connector,
			utils.ColorBold, name, utils.ColorReset,
			utils.ColorDim, gvk, utils.ColorReset,
		)

		indent := "  │   "
		if isLast {
			indent = "      "
		}

		// Workers / resync
		fmt.Printf("%sworkers:%s%d%s  resync:%s%s%s",
			indent,
			utils.ColorGreen, crd.Workers, utils.ColorReset,
			utils.ColorGreen, crd.Resync.String(), utils.ColorReset,
		)
		if crd.Queue.MaxQueueDepth > 0 {
			fmt.Printf("  queue:%s%d%s", utils.ColorGreen, crd.Queue.MaxQueueDepth, utils.ColorReset)
		}
		fmt.Println()

		// DependsOn
		if len(crd.DependsOn) > 0 {
			deps := make([]string, 0, len(crd.DependsOn))
			for depName, d := range crd.DependsOn {
				if d.Condition != "" && d.Condition != "started" {
					deps = append(deps, fmt.Sprintf("%s(%s)", depName, d.Condition))
				} else {
					deps = append(deps, depName)
				}
			}
			fmt.Printf("%s%sdependsOn:%s %s\n",
				indent, utils.ColorYellow, utils.ColorReset,
				strings.Join(deps, ", "),
			)
		}

		// Mode / reconciler
		mode := "default"
		if crd.Mode != "" {
			mode = string(crd.Mode)
		}
		if !crd.DefaultReconcile() {
			if crd.CustomHooksEnabled() {
				mode = "hooks"
			} else if crd.ConstructorEnabled() {
				mode = "constructor"
			}
		}
		fmt.Printf("%smode: %s\n", indent, mode)

		// onCreate resources
		if crd.OperatorBox.OnCreate != nil && !crd.OperatorBox.OnCreate.IsEmpty() {
			fmt.Printf("%s%sonCreate:%s  %s\n", indent, utils.ColorCyan, utils.ColorReset,
				summarizeHookTemplates(crd.OperatorBox.OnCreate))
		}

		// onReconcile resources
		if crd.OperatorBox.OnReconcile != nil && !crd.OperatorBox.OnReconcile.IsEmpty() {
			fmt.Printf("%s%sonReconcile:%s  %s\n", indent, utils.ColorCyan, utils.ColorReset,
				summarizeHookTemplates(crd.OperatorBox.OnReconcile))
		}

		// Status fields
		if crd.OperatorBox.Status != nil && len(crd.OperatorBox.Status.Fields) > 0 {
			fieldNames := make([]string, 0, len(crd.OperatorBox.Status.Fields))
			for _, f := range crd.OperatorBox.Status.Fields {
				fieldNames = append(fieldNames, f.Path)
			}
			fmt.Printf("%s%sstatus:%s  %s\n", indent, utils.ColorCyan, utils.ColorReset,
				strings.Join(fieldNames, ", "))
		}

		// Autoscale
		if crd.AutoscaleEnabled() && crd.OperatorBox.Autoscale != nil {
			a := crd.OperatorBox.Autoscale
			if a.Profile != "" {
				fmt.Printf("%s%sautoscale:%s  profile=%s\n", indent, utils.ColorMagenta, utils.ColorReset, a.Profile)
			} else {
				parts := []string{}
				if len(a.Conditions.When) > 0 {
					triggers := make([]string, 0, len(a.Conditions.When))
					for _, c := range a.Conditions.When {
						triggers = append(triggers, c.Field)
					}
					parts = append(parts, "when("+strings.Join(triggers, ", ")+")")
				}
				if a.Do.Workers != nil {
					parts = append(parts, fmt.Sprintf("workers→%d", *a.Do.Workers))
				}
				if a.Do.Resync != nil {
					parts = append(parts, fmt.Sprintf("resync→%s", a.Do.Resync.String()))
				}
				fmt.Printf("%s%sautoscale:%s  %s  interval:%s  cooldown:%s\n",
					indent, utils.ColorMagenta, utils.ColorReset,
					strings.Join(parts, "  "),
					a.EffectiveInterval(), a.EffectiveCooldown(),
				)
			}
		}

		fmt.Println()
	}

	fmt.Printf("  %s✓ Katalog is valid%s\n\n", utils.ColorGreen, utils.ColorReset)
}

// ── printDependencyGraph ──────────────────────────────────────────────────────

// printDependencyGraph prints a two-part dependency view:
// 1. Ordered startup list (flat)
// 2. Tree view showing the dependency hierarchy
func printDependencyGraph(crds map[string]orktypes.CRDEntry, g *katalog.DependencyGraph, startupOrder []string) {
	fmt.Printf("\n%s%sDependency Graph%s\n\n", utils.ColorBold, utils.ColorCyan, utils.ColorReset)

	// ── Part 1: startup order list ────────────────────────────────────────────
	fmt.Printf("  %sStartup order:%s\n", utils.ColorBold, utils.ColorReset)
	for i, name := range startupOrder {
		crd, ok := crds[name]
		if !ok {
			continue
		}
		deps := crd.DependsOn.Names()
		depStr := ""
		if len(deps) > 0 {
			formatted := make([]string, 0, len(deps))
			for depName, d := range crd.DependsOn {
				if d.Condition != "" && d.Condition != "started" {
					formatted = append(formatted, fmt.Sprintf("%s%s%s(%s%s%s)",
						utils.ColorYellow, depName, utils.ColorReset,
						utils.ColorDim, d.Condition, utils.ColorReset))
				} else {
					formatted = append(formatted, utils.ColorYellow+depName+utils.ColorReset)
				}
			}
			depStr = fmt.Sprintf("  %s←%s %s", utils.ColorDim, utils.ColorReset, strings.Join(formatted, ", "))
		}
		gvk := fmt.Sprintf("%s/%s, Kind=%s", crd.APITypes.Group, crd.APITypes.Version, crd.APITypes.Kind)
		fmt.Printf("    %s%d.%s %-20s %s%s%s%s\n",
			utils.ColorBold, i+1, utils.ColorReset,
			name,
			utils.ColorDim, gvk, utils.ColorReset,
			depStr,
		)
	}

	fmt.Println()

	// ── Part 2: tree view ─────────────────────────────────────────────────────
	fmt.Printf("  %sTree view:%s\n", utils.ColorBold, utils.ColorReset)

	// Find roots (CRDs with no dependencies)
	roots := []string{}
	for _, name := range startupOrder {
		crd, ok := crds[name]
		if !ok {
			continue
		}
		if len(crd.DependsOn) == 0 {
			roots = append(roots, name)
		}
	}

	printed := map[string]bool{}
	for _, root := range roots {
		printGraphNode(crds, g, root, "    ", "", printed)
	}
	fmt.Println()
}

// printGraphNode recursively prints one CRD node and its dependents in tree form.
func printGraphNode(crds map[string]orktypes.CRDEntry, g *katalog.DependencyGraph, name, indent, connector string, printed map[string]bool) {
	crd, ok := crds[name]
	if !ok {
		return
	}

	gvk := fmt.Sprintf("%s/%s", crd.APITypes.Group, crd.APITypes.Version)
	fmt.Printf("%s%s%s%s%s  %s%s%s\n",
		indent, connector,
		utils.ColorBold, name, utils.ColorReset,
		utils.ColorDim, gvk, utils.ColorReset,
	)

	if printed[name] {
		return
	}
	printed[name] = true

	dependents := g.GetDependents(name)
	sort.Strings(dependents)

	for i, dep := range dependents {
		isLast := i == len(dependents)-1
		childConnector := "├── "
		childIndent := indent + "│   "
		if isLast {
			childConnector = "└── "
			childIndent = indent + "    "
		}

		// Show the condition for this edge
		depCrd, ok := crds[dep]
		if ok {
			if d, exists := depCrd.DependsOn[name]; exists && d.Condition != "" && d.Condition != "started" {
				childConnector += fmt.Sprintf("%s(%s)%s ", utils.ColorDim, d.Condition, utils.ColorReset)
			}
		}
		printGraphNode(crds, g, dep, childIndent, childConnector, printed)
	}
}

// ── printCRDDetail ────────────────────────────────────────────────────────────

// printCRDDetail prints the full expanded state of a single CRD as a
// human-readable document — what the runtime will use for this CRD.
func printCRDDetail(crd orktypes.CRDEntry, g *katalog.DependencyGraph) {
	b := utils.ColorBold
	r := utils.ColorReset
	d := utils.ColorDim
	c := utils.ColorCyan
	y := utils.ColorYellow
	m := utils.ColorMagenta
	gr := utils.ColorGreen

	fmt.Printf("\n%s%s%s\n", b, crd.Name, r)
	fmt.Printf("  %sAPIVersion:%s %s/%s\n", c, r, crd.APITypes.Group, crd.APITypes.Version)
	fmt.Printf("  %sKind:%s      %s  (plural: %s)\n", c, r, crd.APITypes.Kind, crd.APITypes.Plural)
	fmt.Printf("  %sNamespaced:%s %v", c, r, crd.IsNamespaced())
	if crd.Namespace != "" {
		fmt.Printf("  (namespace: %s)", crd.Namespace)
	}
	fmt.Println()
	fmt.Printf("  %sEnabled:%s   %v\n", c, r, crd.IsEnabled())
	fmt.Printf("  %sMode:%s      %s\n", c, r, crdModeLabel(crd))
	fmt.Println()

	// ── Runtime config ───────────────────────────────────────────────────────
	fmt.Printf("  %s%sRuntime%s\n", b, c, r)
	fmt.Printf("    Workers:       %s%d%s\n", gr, crd.Workers, r)
	fmt.Printf("    Resync:        %s%s%s\n", gr, crd.Resync.String(), r)
	if crd.Queue.MaxQueueDepth > 0 {
		fmt.Printf("    MaxQueueDepth: %s%d%s\n", gr, crd.Queue.MaxQueueDepth, r)
	}
	fmt.Println()

	// ── DependsOn ─────────────────────────────────────────────────────────────
	if len(crd.DependsOn) > 0 {
		fmt.Printf("  %s%sDependsOn%s\n", b, y, r)
		for depName, dep := range crd.DependsOn {
			cond := dep.Condition
			if cond == "" {
				cond = "started"
			}
			fmt.Printf("    - %s%s%s  condition: %s\n", y, depName, r, cond)
		}
		fmt.Println()
	}

	// ── OperatorBox.OnCreate ──────────────────────────────────────────────────
	if crd.OperatorBox.OnCreate != nil && !crd.OperatorBox.OnCreate.IsEmpty() {
		fmt.Printf("  %s%sonCreate%s\n", b, c, r)
		printHookTemplateDetail("    ", crd.OperatorBox.OnCreate)
		fmt.Println()
	}

	// ── OperatorBox.OnReconcile ───────────────────────────────────────────────
	if crd.OperatorBox.OnReconcile != nil && !crd.OperatorBox.OnReconcile.IsEmpty() {
		fmt.Printf("  %s%sonReconcile%s\n", b, c, r)
		printHookTemplateDetail("    ", crd.OperatorBox.OnReconcile)
		fmt.Println()
	}

	// ── Status ────────────────────────────────────────────────────────────────
	if crd.OperatorBox.Status != nil && len(crd.OperatorBox.Status.Fields) > 0 {
		fmt.Printf("  %s%sStatus Fields%s\n", b, c, r)
		for _, f := range crd.OperatorBox.Status.Fields {
			fmt.Printf("    - %s%s%s: %s%s%s\n", gr, f.Path, r, d, f.Value, r)
		}
		fmt.Println()
	}

	// ── Autoscale ─────────────────────────────────────────────────────────────
	if crd.AutoscaleEnabled() && crd.OperatorBox.Autoscale != nil {
		a := crd.OperatorBox.Autoscale
		fmt.Printf("  %s%sAutoscale%s\n", b, m, r)
		if a.Profile != "" {
			fmt.Printf("    Profile:  %s%s%s\n", m, a.Profile, r)
		} else {
			fmt.Printf("    Interval: %s\n", a.EffectiveInterval())
			fmt.Printf("    Cooldown: %s\n", a.EffectiveCooldown())
			if len(a.Conditions.When) > 0 {
				fmt.Printf("    When (AND):\n")
				for _, cond := range a.Conditions.When {
					printConditionLine("      ", cond)
				}
			}
			if len(a.Conditions.AnyOf) > 0 {
				fmt.Printf("    AnyOf (OR):\n")
				for _, cond := range a.Conditions.AnyOf {
					printConditionLine("      ", cond)
				}
			}
			if a.Do.Workers != nil {
				fmt.Printf("    Do.Workers:    %s%d%s\n", m, *a.Do.Workers, r)
			}
			if a.Do.QueueDepth != nil {
				fmt.Printf("    Do.QueueDepth: %s%d%s\n", m, *a.Do.QueueDepth, r)
			}
			if a.Do.Resync != nil {
				fmt.Printf("    Do.Resync:     %s%s%s\n", m, a.Do.Resync.String(), r)
			}
		}
		fmt.Println()
	}

	// ── Finalizers ────────────────────────────────────────────────────────────
	if len(crd.OperatorBox.Finalizers) > 0 {
		fmt.Printf("  %s%sFinalizers%s\n", b, c, r)
		for _, f := range crd.OperatorBox.Finalizers {
			fmt.Printf("    - %s\n", f)
		}
		fmt.Println()
	}

	// ── Dependents (what depends on this CRD) ────────────────────────────────
	if g != nil {
		dependents := g.GetDependents(crd.Name)
		if len(dependents) > 0 {
			fmt.Printf("  %s%sRequired by%s\n", b, y, r)
			for _, dep := range dependents {
				fmt.Printf("    - %s%s%s\n", y, dep, r)
			}
			fmt.Println()
		}
	}
}

// printHookTemplateDetail prints a detailed breakdown of resource declarations.
func printHookTemplateDetail(indent string, ht *orktypes.HookTemplates) {
	if ht == nil {
		return
	}
	d := utils.ColorDim
	r := utils.ColorReset
	g := utils.ColorGreen

	printResources := func(kind string, names []string) {
		if len(names) == 0 {
			return
		}
		fmt.Printf("%s%s%s(%d):%s\n", indent, g, kind, len(names), r)
		for _, n := range names {
			fmt.Printf("%s  %s- %s%s\n", indent, d, n, r)
		}
	}

	printResources("deployments", deploymentNameList(ht.Deployments))
	printResources("statefulsets", statefulSetNameList(ht.StatefulSets))
	printResources("services", serviceNameList(ht.Services))
	printResources("configmaps", configMapNameList(ht.ConfigMaps))
	printResources("secrets", secretNameList(ht.Secrets))
	printResources("jobs", jobNameList(ht.Jobs))
	printResources("cronJobs", cronJobNameList(ht.CronJobs))
	printResources("serviceAccounts", serviceAccountNameList(ht.ServiceAccounts))
	printResources("ingresses", ingressNameList(ht.Ingresses))
	printResources("pvcs", pvcNameList(ht.PersistentVolumeClaims))
	printResources("namespaces", namespaceNameList(ht.Namespaces))
	printResources("roles", roleNameList(ht.Roles))
	printResources("clusterRoles", clusterRoleNameList(ht.ClusterRoles))

	if len(ht.CustomResource) > 0 {
		fmt.Printf("%s%scustom(%d):%s\n", indent, g, len(ht.CustomResource), r)
		for _, cr := range ht.CustomResource {
			fmt.Printf("%s  %s- %s/%s  name: %s%s\n", indent, d, cr.APIVersion, cr.Kind, cr.Metadata.Name, r)
		}
	}
}

// ── summarizeHookTemplates ────────────────────────────────────────────────────

// summarizeHookTemplates returns a compact one-line resource summary.
func summarizeHookTemplates(ht *orktypes.HookTemplates) string {
	if ht == nil {
		return ""
	}
	var parts []string
	add := func(n int, label string) {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, label))
		}
	}
	add(len(ht.Deployments), "deployment(s)")
	add(len(ht.StatefulSets), "statefulset(s)")
	add(len(ht.Services), "service(s)")
	add(len(ht.ConfigMaps), "configmap(s)")
	add(len(ht.Secrets), "secret(s)")
	add(len(ht.Jobs), "job(s)")
	add(len(ht.CronJobs), "cronjob(s)")
	add(len(ht.Ingresses), "ingress(es)")
	add(len(ht.PersistentVolumeClaims), "pvc(s)")
	add(len(ht.Namespaces), "namespace(s)")
	add(len(ht.ServiceAccounts), "serviceaccount(s)")
	add(len(ht.Roles), "role(s)")
	add(len(ht.ClusterRoles), "clusterrole(s)")
	if len(ht.CustomResource) > 0 {
		kinds := make([]string, 0, len(ht.CustomResource))
		for _, cr := range ht.CustomResource {
			kinds = append(kinds, cr.Kind)
		}
		parts = append(parts, fmt.Sprintf("custom(%s)", strings.Join(kinds, ", ")))
	}
	return strings.Join(parts, ", ")
}

// ── Condition printer ─────────────────────────────────────────────────────────

func printConditionLine(indent string, cond orktypes.Condition) {
	d := utils.ColorDim
	r := utils.ColorReset
	if cond.Field != "" {
		line := fmt.Sprintf("%s- %s", indent, cond.Field)
		if cond.GreaterThan != "" {
			line += fmt.Sprintf(" > %s%s%s", d, cond.GreaterThan, r)
		}
		if cond.LessThan != "" {
			line += fmt.Sprintf(" < %s%s%s", d, cond.LessThan, r)
		}
		if cond.Equals != "" {
			line += fmt.Sprintf(" == %s%s%s", d, cond.Equals, r)
		}
		fmt.Println(line)
	}
}

// ── Name list helpers (for printHookTemplateDetail) ───────────────────────────

func deploymentNameList(srcs []orktypes.DeploymentTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		if s.Name != "" {
			out = append(out, s.Name)
		} else {
			out = append(out, fmt.Sprintf("image:%s", s.Image))
		}
	}
	return out
}

func statefulSetNameList(srcs []orktypes.StatefulSetTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "statefulset"))
	}
	return out
}

func serviceNameList(srcs []orktypes.ServiceTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "service"))
	}
	return out
}

func configMapNameList(srcs []orktypes.ConfigMapTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "configmap"))
	}
	return out
}

func secretNameList(srcs []orktypes.SecretTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "secret"))
	}
	return out
}

func jobNameList(srcs []orktypes.JobTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "job"))
	}
	return out
}

func cronJobNameList(srcs []orktypes.CronJobTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "cronjob"))
	}
	return out
}

func serviceAccountNameList(srcs []orktypes.ServiceAccountTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "serviceaccount"))
	}
	return out
}

func ingressNameList(srcs []orktypes.IngressTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "ingress"))
	}
	return out
}

func pvcNameList(srcs []orktypes.PVCTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "pvc"))
	}
	return out
}

func namespaceNameList(srcs []orktypes.NamespaceTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "namespace"))
	}
	return out
}

func roleNameList(srcs []orktypes.RoleTemplateSource) []string {
	out := make([]string, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, nameOrFallback(s.Name, "role"))
	}
	return out
}

func clusterRoleNameList(srcs []orktypes.PlaceholderSource) []string {
	out := make([]string, len(srcs))
	for i := range srcs {
		out[i] = fmt.Sprintf("<clusterrole-%d>", i+1)
	}
	return out
}

func nameOrFallback(name, fallback string) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("<%s>", fallback)
}

// ── crdModeLabel ──────────────────────────────────────────────────────────────

func crdModeLabel(crd orktypes.CRDEntry) string {
	if crd.DefaultReconcile() {
		return "default"
	}
	if crd.CustomHooksEnabled() {
		return fmt.Sprintf("hooks(%s)", crd.OperatorBox.Hooks.Function)
	}
	if crd.ConstructorEnabled() {
		return fmt.Sprintf("constructor(%s)", crd.OperatorBox.ConstructorDecl.Function)
	}
	if crd.Mode != "" {
		return string(crd.Mode)
	}
	return "default"
}
