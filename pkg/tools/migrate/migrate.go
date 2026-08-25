// pkg/migrate/migrate.go
//
// Rewrites a controller-runtime Reconcile method to the Orkestra constructor
// signature. It is intentionally a starting point — the output compiles but
// still requires review for status updates, event recording, and informer
// cache lookups.
package migrate

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// Mode controls how much of the source file migrate rewrites.
type Mode string

const (
	// ModeNative rewrites the full controller-runtime signature to Orkestra's
	// native style: Reconcile(ctx context.Context, req domain.Request) (domain.Result, error),
	// struct fields replaced, call sites adapted. Most invasive; produces fully idiomatic Orkestra code.
	ModeNative Mode = "native"

	// ModeToClient is the minimal migration path. The Reconcile signature,
	// struct fields, and call sites are left completely unchanged. Only
	// SetupWithManager is removed and a constructor using kubeclient.ToClient
	// and domain.ReconcilerFrom is injected. Two lines of new code; zero
	// changes to existing reconciler logic.
	ModeToClient Mode = "toclient"
)

// Result holds the output of a migration rewrite.
type Result struct {
	// Source is the rewritten Go source, gofmt-formatted.
	Source []byte
	// ReceiverType is the struct name from the Reconcile receiver (e.g. "WebAppReconciler").
	ReceiverType string
	// PkgName is the Go package name of the source file.
	PkgName string
	// Warnings are patterns flagged but not automatically rewritten.
	Warnings []string
	// Mode is the migration mode used to produce this result.
	Mode Mode
	// Owns lists types detected in Owns() calls inside SetupWithManager.
	// Each entry is a resource the operator owns and should appear in constructor.managedResources:.
	Owns []DetectedType
	// Watches lists types detected in Watches() calls inside SetupWithManager.
	// Each entry should appear in operatorBox.watch:.
	Watches []DetectedType
	// Primary holds the type information extracted from the For() call in SetupWithManager.
	Primary PrimaryType
}

// DetectedType is a resource type extracted from an Owns() or Watches() call.
type DetectedType struct {
	Kind       string
	APIVersion string // best-effort from import path; may be TODO if not resolvable
}

// PrimaryType holds the type information extracted from the For() call in SetupWithManager.
// All fields are best-effort; unresolvable fields are left empty.
type PrimaryType struct {
	Kind       string // struct name from For(&pkg.Kind{})
	Object     string // same as Kind
	ObjectList string // Kind + "List" by convention
	Version    string // last path segment of import when it looks like a version (e.g. v1alpha1)
	Location   string // full import path of the package (e.g. github.com/org/project/api/v1alpha1)
	Alias      string // import alias used in source (e.g. demov1alpha1)
}

// replacement is a byte-range substitution to apply to source text.
type replacement struct {
	start int
	end   int
	text  string
}

// Rewrite parses src, locates a controller-runtime Reconcile method, and
// returns the source rewritten according to mode.
//
// ModeToClient is the recommended starting point: zero changes to Reconcile
// or call sites, only SetupWithManager removed and a ToClient constructor
// injected. ModeNative performs the full signature and call-site rewrite.
func Rewrite(src []byte, mode Mode) (*Result, error) {
	if mode == ModeToClient {
		return rewriteToClient(src)
	}
	return rewriteNative(src)
}

func rewriteNative(src []byte) (*Result, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	res := &Result{PkgName: f.Name.Name}

	fn := findReconcile(f)
	if fn == nil {
		return nil, fmt.Errorf("no Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) found")
	}

	receiverName := "r"
	if len(fn.Recv.List) > 0 {
		res.ReceiverType = typeName(fn.Recv.List[0].Type)
		if len(fn.Recv.List[0].Names) > 0 {
			receiverName = fn.Recv.List[0].Names[0].Name
		}
	}

	res.Primary, res.Owns, res.Watches = extractOwnsWatches(f)

	ctxParam := "ctx"
	params := fn.Type.Params.List
	if len(params) > 0 && len(params[0].Names) > 0 {
		ctxParam = params[0].Names[0].Name
	}

	var reps []replacement

	// Change params to (ctx context.Context, req domain.Request)
	reps = append(reps, replacement{
		start: off(fset, fn.Type.Params.Opening),
		end:   off(fset, fn.Type.Params.Closing) + 1,
		text:  fmt.Sprintf("(%s context.Context, req domain.Request)", ctxParam),
	})

	// Change return type to (domain.Result, error)
	if fn.Type.Results != nil {
		reps = append(reps, replacement{
			start: off(fset, fn.Type.Results.Opening),
			end:   off(fset, fn.Type.Results.Closing) + 1,
			text:  "(domain.Result, error)",
		})
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ReturnStmt:
			if len(x.Results) != 2 || !isCtrlResult(x.Results[0]) {
				return true
			}
			secondText := sliceSrc(src, fset, x.Results[1])
			var text string
			if hasRequeueAfter(x.Results[0]) {
				// Preserve RequeueAfter through domain.Result
				afterExpr := requeueAfterExpr(x.Results[0])
				text = "return domain.Result{RequeueAfter: " + afterExpr + "}, " + secondText
			} else {
				text = "return domain.Result{}, " + secondText
			}
			reps = append(reps, replacement{
				start: off(fset, x.Pos()),
				end:   off(fset, x.End()),
				text:  text,
			})

		}
		return true
	})

	// Flag r.Status().Update() in the whole file
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		upd, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || upd.Sel.Name != "Update" {
			return true
		}
		statusCall, ok := upd.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := statusCall.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Status" {
			reps = append(reps, replacement{
				start: off(fset, call.Pos()),
				end:   off(fset, call.End()),
				text:  "nil /* TODO(ork migrate): replace with r.kube.PatchStatus(ctx, obj, map[string]interface{}{...}) */",
			})
			res.Warnings = append(res.Warnings, "r.Status().Update() flagged — replace with r.kube.PatchStatus")
		}
		return true
	})

	// Remove SetupWithManager entirely
	for _, decl := range f.Decls {
		setupFn, ok := decl.(*ast.FuncDecl)
		if !ok || setupFn.Name.Name != "SetupWithManager" || setupFn.Recv == nil {
			continue
		}
		reps = append(reps, replacement{
			start: off(fset, setupFn.Pos()),
			end:   off(fset, setupFn.End()),
			text: "// SetupWithManager removed — Orkestra provides the informer, workqueue,\n" +
				"// worker pool, leader election, panic recovery, and metrics.\n" +
				"// Delete this file's main.go and scheme registration too.",
		})
		res.Warnings = append(res.Warnings, "SetupWithManager removed — delete main.go and scheme registration")
	}

	result := applyReplacements(src, reps)

	// Rewrite r.Get/Create/Patch → r.kube.* with kubeclient signatures.
	result = rewriteKubeCalls(result, receiverName)

	// Rewrite struct and inject constructor — parse fresh after replacements.
	result, structFound := rewriteStruct(result, res.ReceiverType)
	if !structFound {
		res.Warnings = append(res.Warnings, "reconciler struct not found — update struct fields and add constructor manually")
	}

	result = rewriteImports(result, false)

	formatted, fmtErr := format.Source(result)
	if fmtErr != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("gofmt failed (%v) — check output manually", fmtErr))
		res.Source = result
	} else {
		res.Source = formatted
	}

	res.Mode = ModeNative
	return res, nil
}

// rewriteToClient performs the minimal migration: removes SetupWithManager and
// injects a constructor using kubeclient.ToClient + domain.ReconcilerFrom.
// The Reconcile signature, struct fields, and all call sites are untouched.
func rewriteToClient(src []byte) (*Result, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	res := &Result{PkgName: f.Name.Name, Mode: ModeToClient}

	fn := findReconcile(f)
	if fn == nil {
		return nil, fmt.Errorf("no Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) found")
	}
	if len(fn.Recv.List) > 0 {
		res.ReceiverType = typeName(fn.Recv.List[0].Type)
	}

	res.Primary, res.Owns, res.Watches = extractOwnsWatches(f)

	var reps []replacement

	// Remove SetupWithManager.
	for _, decl := range f.Decls {
		setupFn, ok := decl.(*ast.FuncDecl)
		if !ok || setupFn.Name.Name != "SetupWithManager" || setupFn.Recv == nil {
			continue
		}
		reps = append(reps, replacement{
			start: off(fset, setupFn.Pos()),
			end:   off(fset, setupFn.End()),
			text: "// SetupWithManager removed — Orkestra provides the informer, workqueue,\n" +
				"// worker pool, leader election, panic recovery, and metrics.\n" +
				"// Delete this file's main.go and scheme registration too.",
		})
		res.Warnings = append(res.Warnings, "SetupWithManager removed — delete main.go and scheme registration")
	}

	result := applyReplacements(src, reps)

	// Inject constructor using ToClient + ReconcilerFrom.
	if res.ReceiverType != "" {
		constructorName := "New" + res.ReceiverType
		// Only inject if not already present.
		hasConstructor := false
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.Name == constructorName {
				hasConstructor = true
				break
			}
		}
		if !hasConstructor {
			constructor := fmt.Sprintf(`
// %s is the Orkestra constructor. It replaces SetupWithManager — no other
// changes to the reconciler are needed. ToClient returns the same client.Client
// your reconciler already uses; ReconcilerFrom adapts the signature.
func %s(kube kubeclient.Interface) domain.Reconciler {
	return domain.ReconcilerFrom(&%s{
		client: kubeclient.ToClient(kube),
	})
}
`, constructorName, constructorName, res.ReceiverType)
			result = append(result, []byte(constructor)...)
		}
	}

	result = rewriteImportsToClient(result)

	formatted, fmtErr := format.Source(result)
	if fmtErr != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("gofmt failed (%v) — check output manually", fmtErr))
		res.Source = result
	} else {
		res.Source = formatted
	}

	return res, nil
}

// rewriteImportsToClient injects the domain and kubeclient imports needed by
// the generated constructor. The ctrl import is kept — toclient mode leaves
// the Reconcile signature and body unchanged, so ctrl.Request/ctrl.Result stay.
func rewriteImportsToClient(src []byte) []byte {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return src
	}

	hasDomain, hasKubeclient := false, false
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		switch path {
		case "github.com/orkspace/orkestra/domain":
			hasDomain = true
		case "github.com/orkspace/orkestra/pkg/kubeclient":
			hasKubeclient = true
		}
	}

	result := src
	if !hasDomain {
		result = injectImport(result, `"github.com/orkspace/orkestra/domain"`)
	}
	if !hasKubeclient {
		result = injectImport(result, `"github.com/orkspace/orkestra/pkg/kubeclient"`)
	}
	return result
}

// rewriteKubeCalls rewrites r.Get/Create/Patch (and r.<field>.Get/Create/Patch)
// to r.kube.* with kubeclient signatures:
//
//	Get(ctx, namespace, name, obj)   — splits the controller-runtime ObjectKey
//	Create(ctx, obj)                 — drops variadic opts
//	Patch(ctx, obj, patch)           — drops variadic opts
func rewriteKubeCalls(src []byte, receiverName string) []byte {
	if receiverName == "" {
		return src
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return src
	}

	var reps []replacement

	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		op := sel.Sel.Name
		if op != "Get" && op != "Create" && op != "Patch" {
			return true
		}

		// Match r.Op(...) or r.<field>.Op(...)
		isReceiver := false
		if ident, ok2 := sel.X.(*ast.Ident); ok2 && ident.Name == receiverName {
			isReceiver = true
		} else if outer, ok2 := sel.X.(*ast.SelectorExpr); ok2 {
			if ident, ok3 := outer.X.(*ast.Ident); ok3 && ident.Name == receiverName {
				isReceiver = true
			}
		}
		if !isReceiver {
			return true
		}

		// Rewrite the function selector to r.kube.Op
		reps = append(reps, replacement{
			start: off(fset, call.Fun.Pos()),
			end:   off(fset, call.Fun.End()),
			text:  receiverName + ".kube." + op,
		})

		switch op {
		case "Get":
			// (ctx, ObjectKey{Namespace: X, Name: Y}, obj, opts...) → (ctx, X, Y, obj)
			if len(call.Args) < 3 {
				return true
			}
			ctxText := sliceSrc(src, fset, call.Args[0])
			objText := sliceSrc(src, fset, call.Args[2])
			ns, name := extractObjectKeyFields(src, fset, call.Args[1])
			var newArgs string
			if ns != "" && name != "" {
				newArgs = ctxText + ", " + ns + ", " + name + ", " + objText
			} else {
				keyText := sliceSrc(src, fset, call.Args[1])
				newArgs = ctxText + `, namespace, name, ` + objText +
					` /* TODO(ork migrate): extract namespace+name from: ` + keyText + ` */`
			}
			reps = append(reps, replacement{
				start: off(fset, call.Args[0].Pos()),
				end:   off(fset, call.Args[len(call.Args)-1].End()),
				text:  newArgs,
			})

		case "Create":
			// (ctx, obj, opts...) → (ctx, obj)
			if len(call.Args) > 2 {
				ctxText := sliceSrc(src, fset, call.Args[0])
				objText := sliceSrc(src, fset, call.Args[1])
				reps = append(reps, replacement{
					start: off(fset, call.Args[0].Pos()),
					end:   off(fset, call.Args[len(call.Args)-1].End()),
					text:  ctxText + ", " + objText,
				})
			}

		case "Patch":
			// (ctx, obj, patch, opts...) → (ctx, obj, patch)
			if len(call.Args) > 3 {
				ctxText := sliceSrc(src, fset, call.Args[0])
				objText := sliceSrc(src, fset, call.Args[1])
				patchText := sliceSrc(src, fset, call.Args[2])
				reps = append(reps, replacement{
					start: off(fset, call.Args[0].Pos()),
					end:   off(fset, call.Args[len(call.Args)-1].End()),
					text:  ctxText + ", " + objText + ", " + patchText,
				})
			}
		}
		return true
	})

	return applyReplacements(src, reps)
}

// extractObjectKeyFields extracts Namespace and Name text from a client.ObjectKey composite literal.
func extractObjectKeyFields(src []byte, fset *token.FileSet, n ast.Node) (namespace, name string) {
	lit, ok := n.(*ast.CompositeLit)
	if !ok {
		return "", ""
	}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		val := sliceSrc(src, fset, kv.Value)
		switch key.Name {
		case "Namespace":
			namespace = val
		case "Name":
			name = val
		}
	}
	return
}

// rewriteStruct finds the reconciler struct by name, replaces its fields with
// Orkestra's (informer, kube, ev), and appends a constructor function if one
// named New<ReceiverType> does not already exist.
func rewriteStruct(src []byte, receiverType string) ([]byte, bool) {
	if receiverType == "" {
		return src, false
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return src, false
	}

	var reps []replacement
	foundStruct := false
	constructorName := "New" + receiverType
	hasConstructor := false

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != receiverType {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				foundStruct = true
				reps = append(reps, replacement{
					start: off(fset, st.Fields.Opening),
					end:   off(fset, st.Fields.Closing) + 1,
					text:  "{\n\tkube kubeclient.Interface\n}",
				})
			}
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name == constructorName {
				hasConstructor = true
			}
		}
	}

	if !foundStruct {
		return src, false
	}

	result := applyReplacements(src, reps)

	if !hasConstructor {
		constructor := fmt.Sprintf(`
// %s is the constructor function registered in the Katalog.
func %s(kube kubeclient.Interface) domain.Reconciler {
	return &%s{kube: kube}
}
`, constructorName, constructorName, receiverType)
		result = append(result, []byte(constructor)...)
	}

	return result, true
}

// findReconcile locates a method named Reconcile whose second parameter is
// a selector expression ending in "Request" (ctrl.Request).
func findReconcile(f *ast.File) *ast.FuncDecl {
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Reconcile" || fn.Recv == nil || fn.Type.Params == nil {
			continue
		}
		params := fn.Type.Params.List
		if len(params) != 2 {
			continue
		}
		sel, ok := params[1].Type.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "Request" {
			return fn
		}
	}
	return nil
}

// isCtrlResult reports whether expr is ctrl.Result{...} or ctrl.Result{}.
func isCtrlResult(expr ast.Expr) bool {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return false
	}
	sel, ok := lit.Type.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Result"
}

// hasRequeueAfter reports whether a ctrl.Result composite literal sets RequeueAfter.
func hasRequeueAfter(expr ast.Expr) bool {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return false
	}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "RequeueAfter" {
			return true
		}
	}
	return false
}

// requeueAfterExpr extracts the RequeueAfter value expression text from a ctrl.Result literal.
// Returns "0" if not found.
func requeueAfterExpr(expr ast.Expr) string {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return "0"
	}
	fset := token.NewFileSet()
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "RequeueAfter" {
			var buf strings.Builder
			_ = format.Node(&buf, fset, kv.Value)
			return buf.String()
		}
	}
	return "0"
}

// typeName extracts the base type name from a receiver type expression.
// Handles *T and T.
func typeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return typeName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// off returns the byte offset of a token.Pos within fset.
func off(fset *token.FileSet, pos token.Pos) int {
	return fset.Position(pos).Offset
}

// sliceSrc extracts the source bytes for an AST node.
func sliceSrc(src []byte, fset *token.FileSet, n ast.Node) string {
	return string(src[off(fset, n.Pos()):off(fset, n.End())])
}

// applyReplacements applies all replacements to src in reverse offset order
// so earlier offsets remain valid throughout.
func applyReplacements(src []byte, reps []replacement) []byte {
	sort.Slice(reps, func(i, j int) bool {
		return reps[i].start > reps[j].start
	})

	result := make([]byte, len(src))
	copy(result, src)

	for _, r := range reps {
		result = append(result[:r.start], append([]byte(r.text), result[r.end:]...)...)
	}
	return result
}

// rewriteImports removes the ctrl import and adds strings if needed.
func rewriteImports(src []byte, addStrings bool) []byte {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return src
	}

	var reps []replacement
	hasStrings := false

	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		switch path {
		case "sigs.k8s.io/controller-runtime":
			start := off(fset, imp.Pos())
			end := off(fset, imp.End())
			if end < len(src) && src[end] == '\n' {
				end++
			}
			reps = append(reps, replacement{start: start, end: end, text: ""})
		case "strings":
			hasStrings = true
		}
	}

	result := applyReplacements(src, reps)

	// Inject "strings" import if needed and not already present
	if addStrings && !hasStrings {
		result = injectImport(result, `"strings"`)
	}

	// Inject Orkestra import hints as a block comment so the user knows exactly what to add.
	result = injectImport(result,
		"// TODO(ork migrate): add these imports:\n"+
			"//   \"github.com/orkspace/orkestra/domain\"\n"+
			"//   \"github.com/orkspace/orkestra/pkg/kubeclient\"")

	return result
}

// extractOwnsWatches scans SetupWithManager for For(), Owns(), and Watches() call
// chains. Import aliases are resolved to apiVersion strings and import paths using
// the file's import declarations (best-effort; emits TODO when unresolvable).
func extractOwnsWatches(f *ast.File) (primary PrimaryType, owns, watches []DetectedType) {
	imports := buildImportMaps(f)

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "SetupWithManager" || fn.Recv == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			method := sel.Sel.Name
			if len(call.Args) == 0 {
				return true
			}
			switch method {
			case "For":
				kind, pkgAlias := detectKindAndAlias(call.Args[0])
				if kind == "" {
					return true
				}
				info := imports[pkgAlias]
				primary = PrimaryType{
					Kind:       kind,
					Object:     kind,
					ObjectList: kind + "List",
					Version:    importPathVersion(info.Path),
					Location:   info.Path,
					Alias:      pkgAlias,
				}
			case "Owns":
				dt, ok := detectTypeArg(call.Args[0], imports)
				if !ok {
					return true
				}
				owns = append(owns, dt)
			case "Watches":
				dt, ok := detectTypeArg(call.Args[0], imports)
				if !ok {
					return true
				}
				watches = append(watches, dt)
			}
			return true
		})
	}
	return
}

// detectPkgAlias returns the package alias used in a &pkg.Kind{} expression.
// importPathVersion extracts the version segment from an import path when the
// last segment looks like a Go module version tag (e.g. v1, v1alpha1, v2beta2).
func importPathVersion(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if len(last) > 1 && last[0] == 'v' && last[1] >= '0' && last[1] <= '9' {
		return last
	}
	return ""
}

// detectKindAndAlias extracts the struct name and package alias from a
// &pkg.Kind{} or &Kind{} expression. alias is empty when there is no qualifier.
func detectKindAndAlias(arg ast.Expr) (kind, alias string) {
	if unary, ok := arg.(*ast.UnaryExpr); ok {
		arg = unary.X
	}
	lit, ok := arg.(*ast.CompositeLit)
	if !ok {
		return "", ""
	}
	switch t := lit.Type.(type) {
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return t.Sel.Name, ident.Name
		}
	case *ast.Ident:
		return t.Name, ""
	}
	return "", ""
}

// detectTypeArg extracts Kind and APIVersion from a &pkg.Kind{} argument.
func detectTypeArg(arg ast.Expr, imports map[string]importInfo) (DetectedType, bool) {
	kind, alias := detectKindAndAlias(arg)
	if kind == "" {
		return DetectedType{}, false
	}
	apiVersion := "TODO"
	if alias != "" {
		apiVersion = imports[alias].APIVersion
	}
	return DetectedType{Kind: kind, APIVersion: apiVersion}, true
}

// importInfo holds the resolved values for one import declaration.
type importInfo struct {
	Path       string // full import path
	APIVersion string // best-effort Kubernetes apiVersion
}

// buildImportMaps returns alias → importInfo for all imports in one pass,
// replacing the two separate buildImportPathMap / buildImportAliasMap functions.
func buildImportMaps(f *ast.File) map[string]importInfo {
	m := make(map[string]importInfo)
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		var alias string
		if imp.Name != nil && imp.Name.Name != "_" && imp.Name.Name != "." {
			alias = imp.Name.Name
		} else {
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}
		m[alias] = importInfo{Path: path, APIVersion: importPathToAPIVersion(path)}
	}
	return m
}

// importPathToAPIVersion converts a Go import path to a Kubernetes apiVersion.
// Examples:
//
//	k8s.io/api/apps/v1        → apps/v1
//	k8s.io/api/core/v1        → v1
//	k8s.io/api/networking/v1  → networking/v1
//	github.com/org/project/api/v1alpha1 → TODO: github.com/org/project/api/v1alpha1
func importPathToAPIVersion(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "TODO"
	}
	// k8s.io/api/<group>/<version>
	if strings.HasPrefix(path, "k8s.io/api/") {
		group := parts[len(parts)-2]
		version := parts[len(parts)-1]
		if group == "core" {
			return version
		}
		return group + "/" + version
	}
	// For custom API packages, emit a TODO with the path so the user can fill it in.
	return "TODO: " + path
}

// injectImport inserts a line into the first import block found in src.
func injectImport(src []byte, line string) []byte {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return src
	}
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok.String() != "import" {
			continue
		}
		insertAt := off(fset, genDecl.Lparen) + 1
		insertion := "\n\t" + line
		return append(src[:insertAt], append([]byte(insertion), src[insertAt:]...)...)
	}
	return src
}
