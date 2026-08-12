package migrate

import (
	"strings"
	"testing"
)

// baseline is a minimal controller-runtime reconciler matching the pattern
// that ork migrate targets — embedded client, ctrl.Request signature, req.NamespacedName.
const baseline = `package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	demov1alpha1 "github.com/example/operator/api/v1alpha1"
)

type WebAppReconciler struct {
	client.Client
	Scheme interface{}
}

func (r *WebAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	webapp := &demov1alpha1.WebApp{}
	if err := r.Get(ctx, req.NamespacedName, webapp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	webapp.Status.Phase = "Running"
	if err := r.Status().Update(ctx, webapp); err != nil {
		logger.Error(err, "status update failed")
		return ctrl.Result{}, err
	}

	_ = fmt.Sprintf("reconciled %s", req.String())
	return ctrl.Result{}, nil
}

func (r *WebAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1alpha1.WebApp{}).
		Complete(r)
}
`

func TestRewrite_SignatureChange(t *testing.T) {
	res, err := Rewrite([]byte(baseline))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	src := string(res.Source)

	if !strings.Contains(src, "Reconcile(ctx context.Context, key string) error") {
		t.Error("expected Orkestra signature: Reconcile(ctx context.Context, key string) error")
	}
	if strings.Contains(src, "ctrl.Request") {
		t.Error("ctrl.Request should be removed")
	}
	if strings.Contains(src, "ctrl.Result") {
		t.Error("ctrl.Result should be removed")
	}
}

func TestRewrite_ReturnCollapse(t *testing.T) {
	res, err := Rewrite([]byte(baseline))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	src := string(res.Source)

	if strings.Contains(src, "ctrl.Result{}") {
		t.Error("ctrl.Result{} should be collapsed")
	}
	// Error returns become `return err`, nil returns become `return nil`.
	if !strings.Contains(src, "return err") {
		t.Error("expected collapsed return err")
	}
	if !strings.Contains(src, "return nil") {
		t.Error("expected collapsed return nil")
	}
}

func TestRewrite_ReqNamespacedName(t *testing.T) {
	res, err := Rewrite([]byte(baseline))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	src := string(res.Source)

	// key split injected
	if !strings.Contains(src, `strings.SplitN(key, "/", 2)`) {
		t.Error("expected key split injection")
	}
	if strings.Contains(src, "req.NamespacedName") {
		t.Error("req.NamespacedName should be replaced")
	}
	if !strings.Contains(src, "r.kube.Get(ctx, namespace, name,") {
		t.Error("expected r.kube.Get with extracted namespace and name args")
	}
}

func TestRewrite_ReqString(t *testing.T) {
	res, err := Rewrite([]byte(baseline))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	src := string(res.Source)

	if strings.Contains(src, "req.String()") {
		t.Error("req.String() should be replaced with key")
	}
}

func TestRewrite_SetupWithManagerRemoved(t *testing.T) {
	res, err := Rewrite([]byte(baseline))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	src := string(res.Source)

	// The method declaration must be gone; only the replacement comment remains.
	if strings.Contains(src, "func (r *WebAppReconciler) SetupWithManager") {
		t.Error("SetupWithManager func declaration should be removed")
	}
	if !strings.Contains(src, "SetupWithManager removed") {
		t.Error("expected SetupWithManager removal comment")
	}
}

func TestRewrite_StructRewritten(t *testing.T) {
	res, err := Rewrite([]byte(baseline))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	src := string(res.Source)

	if strings.Contains(src, "client.Client") {
		t.Error("embedded client.Client should be removed from struct")
	}
	if !strings.Contains(src, "kube     kubeclient.Interface") {
		t.Error("expected kube kubeclient.Interface field in struct")
	}
	if !strings.Contains(src, "informer cache.SharedIndexInformer") {
		t.Error("expected informer cache.SharedIndexInformer field in struct")
	}
	if !strings.Contains(src, "ev       event.Recorder") {
		t.Error("expected ev event.Recorder field in struct")
	}
}

func TestRewrite_ConstructorGenerated(t *testing.T) {
	res, err := Rewrite([]byte(baseline))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	src := string(res.Source)

	if !strings.Contains(src, "func NewWebAppReconciler(") {
		t.Error("expected NewWebAppReconciler constructor")
	}
	if !strings.Contains(src, ") domain.Reconciler {") {
		t.Error("expected domain.Reconciler return type on constructor")
	}
}

func TestRewrite_StatusUpdateFlagged(t *testing.T) {
	res, err := Rewrite([]byte(baseline))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	src := string(res.Source)

	if !strings.Contains(src, "TODO(ork migrate)") {
		t.Error("expected TODO(ork migrate) for r.Status().Update()")
	}
	hasWarning := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "PatchStatus") {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("expected PatchStatus warning in result.Warnings")
	}
}

func TestRewrite_ReceiverType(t *testing.T) {
	res, err := Rewrite([]byte(baseline))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}
	if res.ReceiverType != "WebAppReconciler" {
		t.Errorf("ReceiverType = %q, want WebAppReconciler", res.ReceiverType)
	}
	if res.PkgName != "controller" {
		t.Errorf("PkgName = %q, want controller", res.PkgName)
	}
}

func TestRewrite_NoCtrlImport(t *testing.T) {
	res, err := Rewrite([]byte(baseline))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	src := string(res.Source)

	if strings.Contains(src, `"sigs.k8s.io/controller-runtime"`) {
		t.Error("ctrl import should be removed")
	}
}

func TestRewrite_RequeueAfterFlagged(t *testing.T) {
	src := `package controller

import (
	"context"
	"time"
	ctrl "sigs.k8s.io/controller-runtime"
)

type MyReconciler struct{}

func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
`
	res, err := Rewrite([]byte(src))
	if err != nil {
		t.Fatalf("Rewrite: %v", err)
	}

	out := string(res.Source)
	if !strings.Contains(out, "TODO(ork migrate): RequeueAfter removed") {
		t.Error("expected RequeueAfter TODO comment")
	}
	hasWarning := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "RequeueAfter") {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("expected RequeueAfter warning")
	}
}

func TestRewrite_NoReconcileMethod(t *testing.T) {
	src := `package controller

type MyReconciler struct{}

func (r *MyReconciler) DoSomething() {}
`
	_, err := Rewrite([]byte(src))
	if err == nil {
		t.Error("expected error when no Reconcile method found")
	}
}

func TestGenerate_KatalogContainsConstructor(t *testing.T) {
	res := &Result{
		ReceiverType: "WebAppReconciler",
		PkgName:      "controller",
	}
	files := Generate(res, Options{
		ModulePath:   "github.com/example/webapp-operator",
		OperatorName: "webapp-operator",
		OrkVersion:   "v0.8.0",
	})

	if !strings.Contains(files.Katalog, "function: NewWebAppReconciler") {
		t.Error("katalog.yaml should contain constructor function name")
	}
	if !strings.Contains(files.Katalog, "default: false") {
		t.Error("katalog.yaml should have default: false for constructor")
	}
	if !strings.Contains(files.GoMod, "v0.8.0") {
		t.Error("go.mod should contain Orkestra version")
	}
}

func TestToKebab(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"WebAppReconciler", "web-app-reconciler"},
		{"DatabaseOperator", "database-operator"},
		{"MyReconciler", "my-reconciler"},
		{"", "my-operator"},
	}
	for _, c := range cases {
		got := toKebab(c.in)
		if got != c.want {
			t.Errorf("toKebab(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
