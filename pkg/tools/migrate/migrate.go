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
}

// replacement is a byte-range substitution to apply to source text.
type replacement struct {
	start int
	end   int
	text  string
}

// Rewrite parses src, locates a controller-runtime Reconcile method, and
// returns the source with the Orkestra constructor signature applied.
func Rewrite(src []byte) (*Result, error) {
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

	ctxParam, reqParam := "ctx", "req"
	params := fn.Type.Params.List
	if len(params) > 0 && len(params[0].Names) > 0 {
		ctxParam = params[0].Names[0].Name
	}
	if len(params) > 1 && len(params[1].Names) > 0 {
		reqParam = params[1].Names[0].Name
	}

	var reps []replacement

	// Change params to (ctx context.Context, key string)
	reps = append(reps, replacement{
		start: off(fset, fn.Type.Params.Opening),
		end:   off(fset, fn.Type.Params.Closing) + 1,
		text:  fmt.Sprintf("(%s context.Context, key string)", ctxParam),
	})

	// Change return type to error
	if fn.Type.Results != nil {
		reps = append(reps, replacement{
			start: off(fset, fn.Type.Results.Opening),
			end:   off(fset, fn.Type.Results.Closing) + 1,
			text:  "error",
		})
	}

	usesNamespacedName := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ReturnStmt:
			if len(x.Results) != 2 || !isCtrlResult(x.Results[0]) {
				return true
			}
			secondText := sliceSrc(src, fset, x.Results[1])
			var text string
			if hasRequeueAfter(x.Results[0]) {
				text = "// TODO(ork migrate): RequeueAfter removed — equivalent is `return err`.\n\t\t// A non-nil error requeues with exponential backoff. To requeue without\n\t\t// signalling failure, return a sentinel error or use a named error type.\n\t\treturn " + secondText
				res.Warnings = append(res.Warnings, "ctrl.Result{RequeueAfter:} found — equivalent is `return err`; non-nil error requeues with backoff")
			} else {
				text = "return " + secondText
			}
			reps = append(reps, replacement{
				start: off(fset, x.Pos()),
				end:   off(fset, x.End()),
				text:  text,
			})

		case *ast.SelectorExpr:
			ident, ok := x.X.(*ast.Ident)
			if !ok || ident.Name != reqParam {
				return true
			}
			if x.Sel.Name == "NamespacedName" {
				usesNamespacedName = true
				reps = append(reps, replacement{
					start: off(fset, x.Pos()),
					end:   off(fset, x.End()),
					text:  "client.ObjectKey{Namespace: namespace, Name: name}",
				})
			}

		case *ast.CallExpr:
			// req.String() → key
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if ok && ident.Name == reqParam && sel.Sel.Name == "String" && len(x.Args) == 0 {
				reps = append(reps, replacement{
					start: off(fset, x.Pos()),
					end:   off(fset, x.End()),
					text:  "key",
				})
			}
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

	// Inject key split at top of Reconcile body when req.NamespacedName was used
	if usesNamespacedName {
		bodyOpen := off(fset, fn.Body.Lbrace) + 1
		reps = append(reps, replacement{
			start: bodyOpen,
			end:   bodyOpen,
			text:  "\n\tparts := strings.SplitN(key, \"/\", 2)\n\tnamespace, name := parts[0], parts[1]\n",
		})
	}

	result := applyReplacements(src, reps)

	// Rewrite r.Get/Create/Patch → r.kube.* with kubeclient signatures.
	result = rewriteKubeCalls(result, receiverName)

	// Rewrite struct and inject constructor — parse fresh after replacements.
	result, structFound := rewriteStruct(result, res.ReceiverType)
	if !structFound {
		res.Warnings = append(res.Warnings, "reconciler struct not found — update struct fields and add constructor manually")
	}

	result = rewriteImports(result, usesNamespacedName)

	formatted, fmtErr := format.Source(result)
	if fmtErr != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("gofmt failed (%v) — check output manually", fmtErr))
		res.Source = result
	} else {
		res.Source = formatted
	}

	return res, nil
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
					text: "{\n\tinformer cache.SharedIndexInformer\n" +
						"\tkube     kubeclient.Interface\n" +
						"\tev       event.Recorder\n}",
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
func %s(
	kube kubeclient.Interface,
	informer cache.SharedIndexInformer,
	ev event.Recorder,
) domain.Reconciler {
	return &%s{
		kube:     kube,
		informer: informer,
		ev:       ev,
	}
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
			"//   \"github.com/orkspace/orkestra/pkg/event\"\n"+
			"//   \"github.com/orkspace/orkestra/pkg/kubeclient\"\n"+
			"//   \"k8s.io/client-go/tools/cache\"")

	return result
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
