package generate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orkspace/orkestra/pkg/generate"
	"github.com/orkspace/orkestra/pkg/konfig"
	rbacv1 "k8s.io/api/rbac/v1"
)

// keep os and filepath for writeTempKatalog

const testNamespace = "test-namespace"

// newKfg returns a minimal Konfig suitable for unit tests.
func newKfg(t *testing.T) *konfig.Konfig {
	t.Helper()
	kfg, err := konfig.Init()
	if err != nil {
		t.Fatalf("konfig.Init: %v", err)
	}
	return kfg
}

// writeTempKatalog writes a minimal katalog YAML to a temp file and returns its path.
func writeTempKatalog(t *testing.T) string {
	t.Helper()
	content := "apiVersion: orkestra.orkspace.io/v1\nkind: Katalog\n"
	path := filepath.Join(t.TempDir(), "katalog.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp katalog: %v", err)
	}
	return path
}

// countOccurrences returns how many times sub appears in s.
func countOccurrences(s, sub string) int {
	return strings.Count(s, sub)
}

// ── RBAC standalone ──────────────────────────────────────────────────────────

func TestRBAC_ContainsNamespaceFirst(t *testing.T) {

	output, err := generate.RBAC(nil, testNamespace, "")
	if err != nil {
		t.Fatalf("RBAC: %v", err)
	}
	s := string(output)

	nsIdx := strings.Index(s, "kind: Namespace")
	saIdx := strings.Index(s, "kind: ServiceAccount")
	if nsIdx < 0 {
		t.Fatal("RBAC output missing kind: Namespace")
	}
	if saIdx < 0 {
		t.Fatal("RBAC output missing kind: ServiceAccount")
	}
	if nsIdx > saIdx {
		t.Error("Namespace must appear before ServiceAccount in RBAC output")
	}
}

func TestRBAC_ContainsExpectedResources(t *testing.T) {
	output, err := generate.RBAC(nil, testNamespace, "")
	if err != nil {
		t.Fatalf("RBAC: %v", err)
	}
	s := string(output)

	for _, want := range []string{
		"kind: Namespace",
		"kind: ServiceAccount",
		"kind: ClusterRole",
		"kind: ClusterRoleBinding",
		testNamespace,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("RBAC output missing %q", want)
		}
	}
}

func TestRBAC_NamespaceAppearsOnce(t *testing.T) {

	output, err := generate.RBAC([]rbacv1.PolicyRule{}, testNamespace, "")
	if err != nil {
		t.Fatalf("RBAC: %v", err)
	}

	if n := countOccurrences(string(output), "kind: Namespace"); n != 1 {
		t.Errorf("expected Namespace exactly once in RBAC output, got %d", n)
	}
}

// ── ConfigMap standalone ─────────────────────────────────────────────────────

func TestConfigMap_ContainsNamespaceFirst(t *testing.T) {
	katalogPath := writeTempKatalog(t)

	output, err := generate.ConfigMap(katalogPath, testNamespace, "")
	if err != nil {
		t.Fatalf("ConfigMap: %v", err)
	}
	s := string(output)

	nsIdx := strings.Index(s, "kind: Namespace")
	cmIdx := strings.Index(s, "kind: ConfigMap")
	if nsIdx < 0 {
		t.Fatal("ConfigMap output missing kind: Namespace")
	}
	if cmIdx < 0 {
		t.Fatal("ConfigMap output missing kind: ConfigMap")
	}
	if nsIdx > cmIdx {
		t.Error("Namespace must appear before ConfigMap in output")
	}
}

func TestConfigMap_NamespaceAppearsOnce(t *testing.T) {
	katalogPath := writeTempKatalog(t)

	output, err := generate.ConfigMap(katalogPath, testNamespace, "")
	if err != nil {
		t.Fatalf("ConfigMap: %v", err)
	}

	if n := countOccurrences(string(output), "kind: Namespace"); n != 1 {
		t.Errorf("expected Namespace exactly once in ConfigMap output, got %d", n)
	}
}

// ── Bundle ───────────────────────────────────────────────────────────────────

func TestRenderBundle_NamespaceAppearsOnce(t *testing.T) {

	katalogPath := writeTempKatalog(t)

	bundle, err := generate.RenderBundle(nil, katalogPath, testNamespace, "")
	if err != nil {
		t.Fatalf("RenderBundle: %v", err)
	}

	if n := countOccurrences(bundle, "kind: Namespace"); n != 1 {
		t.Errorf("expected Namespace exactly once in bundle, got %d\n\nBundle:\n%s", n, bundle)
	}
}

func TestRenderBundle_ContainsAllResources(t *testing.T) {

	katalogPath := writeTempKatalog(t)

	bundle, err := generate.RenderBundle([]rbacv1.PolicyRule{}, katalogPath, testNamespace, "")
	if err != nil {
		t.Fatalf("RenderBundle: %v", err)
	}

	for _, want := range []string{
		"kind: Namespace",
		"kind: ServiceAccount",
		"kind: ClusterRole",
		"kind: ClusterRoleBinding",
		"kind: ConfigMap",
		testNamespace,
	} {
		if !strings.Contains(bundle, want) {
			t.Errorf("bundle missing %q", want)
		}
	}
}

func TestRenderBundle_NamespaceBeforeEverythingElse(t *testing.T) {

	katalogPath := writeTempKatalog(t)

	bundle, err := generate.RenderBundle(nil, katalogPath, testNamespace, "")
	if err != nil {
		t.Fatalf("RenderBundle: %v", err)
	}

	nsIdx := strings.Index(bundle, "kind: Namespace")
	for _, kind := range []string{"kind: ServiceAccount", "kind: ClusterRole", "kind: ClusterRoleBinding", "kind: ConfigMap"} {
		idx := strings.Index(bundle, kind)
		if idx >= 0 && nsIdx > idx {
			t.Errorf("Namespace must appear before %s in bundle", kind)
		}
	}
}

// TestRenderBundle_WorkloadNamespace verifies that when a workload namespace is
// provided, both namespaces appear exactly once and there are no double ---
// separators (regression for the reconcile→MODIFIED loop fix).
func TestRenderBundle_WorkloadNamespace(t *testing.T) {

	katalogPath := writeTempKatalog(t)
	workloadNS := "myapp-orkestra-ns"

	bundle, err := generate.RenderBundle(nil, katalogPath, testNamespace, workloadNS)
	if err != nil {
		t.Fatalf("RenderBundle: %v", err)
	}

	// Both namespaces must appear exactly once.
	if n := countOccurrences(bundle, "kind: Namespace"); n != 2 {
		t.Errorf("expected 2 Namespace docs (operator + workload), got %d\n\nBundle:\n%s", n, bundle)
	}
	if !strings.Contains(bundle, workloadNS) {
		t.Errorf("bundle missing workload namespace %q", workloadNS)
	}

	// No double --- separators (the specific bug this fixes).
	if strings.Contains(bundle, "---\n---") {
		t.Errorf("bundle contains double --- separator\n\nBundle:\n%s", bundle)
	}
}

// TestRenderBundle_NoDoubleDocSeparators checks that every --- appears exactly
// once between documents, regardless of whether a workload namespace is used.
func TestRenderBundle_NoDoubleDocSeparators(t *testing.T) {

	katalogPath := writeTempKatalog(t)

	for _, workloadNS := range []string{"", "myapp-orkestra-ns"} {
		bundle, err := generate.RenderBundle(nil, katalogPath, testNamespace, workloadNS)
		if err != nil {
			t.Fatalf("RenderBundle (workloadNS=%q): %v", workloadNS, err)
		}
		if strings.Contains(bundle, "---\n---") {
			t.Errorf("double --- separator found (workloadNS=%q)\n\nBundle:\n%s", workloadNS, bundle)
		}
	}
}
