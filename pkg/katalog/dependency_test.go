// pkg/katalog/dependency_test.go
// White-box unit tests for the DependencyGraph topological sort.
package katalog

import (
	"reflect"
	"sort"
	"testing"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// crd is a minimal CRDEntry builder for dependency tests.
func crd(name string, dependsOn ...string) orktypes.CRDEntry {
	return orktypes.CRDEntry{Name: name, DependsOn: dependsOn}
}

// ── No dependencies ───────────────────────────────────────────────────────────

func TestDependencyGraph_StartupOrder_NoDeps_SingleCRD(t *testing.T) {
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{crd("website")}}
	order := NewDependencyGraph(k).StartupOrder()

	if len(order) != 1 || order[0] != "website" {
		t.Errorf("expected [website], got %v", order)
	}
}

func TestDependencyGraph_StartupOrder_NoDeps_MultiCRD_IsSorted(t *testing.T) {
	// No dependencies → alphabetical order (deterministic).
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		crd("zebra"),
		crd("apple"),
		crd("mango"),
	}}
	order := NewDependencyGraph(k).StartupOrder()

	expected := []string{"apple", "mango", "zebra"}
	if !reflect.DeepEqual(order, expected) {
		t.Errorf("expected %v, got %v", expected, order)
	}
}

// ── Linear chain ──────────────────────────────────────────────────────────────

func TestDependencyGraph_StartupOrder_LinearChain(t *testing.T) {
	// database → cache → application (database starts first)
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		crd("application", "cache"),
		crd("cache", "database"),
		crd("database"),
	}}
	order := NewDependencyGraph(k).StartupOrder()

	assertBefore(t, order, "database", "cache")
	assertBefore(t, order, "cache", "application")
}

// ── Fan-out ───────────────────────────────────────────────────────────────────

func TestDependencyGraph_StartupOrder_FanOut(t *testing.T) {
	// One provider depended on by multiple consumers.
	// provider must come before both consumer-a and consumer-b.
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		crd("consumer-a", "provider"),
		crd("consumer-b", "provider"),
		crd("provider"),
	}}
	order := NewDependencyGraph(k).StartupOrder()

	assertBefore(t, order, "provider", "consumer-a")
	assertBefore(t, order, "provider", "consumer-b")
}

// ── Fan-in ────────────────────────────────────────────────────────────────────

func TestDependencyGraph_StartupOrder_FanIn(t *testing.T) {
	// One consumer depends on two providers.
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		crd("app", "db", "cache"),
		crd("db"),
		crd("cache"),
	}}
	order := NewDependencyGraph(k).StartupOrder()

	assertBefore(t, order, "db", "app")
	assertBefore(t, order, "cache", "app")
}

// ── ShutdownOrder is reverse of StartupOrder ──────────────────────────────────

func TestDependencyGraph_ShutdownOrder_IsReversedStartup(t *testing.T) {
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		crd("app", "db"),
		crd("db"),
	}}
	g := NewDependencyGraph(k)

	startup := g.StartupOrder()
	shutdown := g.ShutdownOrder()

	if len(startup) != len(shutdown) {
		t.Fatalf("startup and shutdown orders have different lengths")
	}
	for i, v := range startup {
		mirror := shutdown[len(shutdown)-1-i]
		if v != mirror {
			t.Errorf("shutdown[%d]=%q is not the reverse of startup[%d]=%q", len(shutdown)-1-i, mirror, i, v)
		}
	}
}

// ── Accessors ─────────────────────────────────────────────────────────────────

func TestDependencyGraph_GetNode(t *testing.T) {
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{crd("website")}}
	g := NewDependencyGraph(k)

	n := g.GetNode("website")
	if n == nil {
		t.Fatal("expected node for 'website'")
	}
	if n.Name != "website" {
		t.Errorf("node name: expected website, got %q", n.Name)
	}
}

func TestDependencyGraph_InDegree_OutDegree(t *testing.T) {
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		crd("app", "db"),
		crd("db"),
	}}
	g := NewDependencyGraph(k)

	// db has no deps (inDegree=0) and one dependent (outDegree=1)
	if g.GetInDegree("db") != 0 {
		t.Errorf("db inDegree: expected 0, got %d", g.GetInDegree("db"))
	}
	if g.GetOutDegree("db") != 1 {
		t.Errorf("db outDegree: expected 1, got %d", g.GetOutDegree("db"))
	}

	// app depends on db (inDegree=1) and no one depends on it (outDegree=0)
	if g.GetInDegree("app") != 1 {
		t.Errorf("app inDegree: expected 1, got %d", g.GetInDegree("app"))
	}
	if g.GetOutDegree("app") != 0 {
		t.Errorf("app outDegree: expected 0, got %d", g.GetOutDegree("app"))
	}
}

func TestDependencyGraph_GetDependents(t *testing.T) {
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		crd("app-a", "db"),
		crd("app-b", "db"),
		crd("db"),
	}}
	g := NewDependencyGraph(k)

	dependents := g.GetDependents("app-a")
	// app-a depends on db, so db is its dependency — GetDependents returns
	// nodes that depend on the given node.
	_ = dependents // just verify no panic
}

func TestDependencyGraph_Validate_NoError(t *testing.T) {
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		crd("app", "db"),
		crd("db"),
	}}
	g := NewDependencyGraph(k)
	if err := g.Validate(); err != nil {
		t.Errorf("expected no validation error, got %v", err)
	}
}

// ── Cycle detection ───────────────────────────────────────────────────────────

func TestDetectCycles_TwoNodeCycle(t *testing.T) {
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		{Name: "a", DependsOn: []string{"b"}},
		{Name: "b", DependsOn: []string{"a"}},
	}}
	if err := k.detectDependencyCycles(); err == nil {
		t.Error("expected cycle detection error for a ↔ b cycle")
	}
}

func TestDetectCycles_ThreeNodeCycle(t *testing.T) {
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		{Name: "a", DependsOn: []string{"c"}},
		{Name: "b", DependsOn: []string{"a"}},
		{Name: "c", DependsOn: []string{"b"}},
	}}
	if err := k.detectDependencyCycles(); err == nil {
		t.Error("expected cycle detection error for a → c → b → a cycle")
	}
}

func TestDetectCycles_NoCycle(t *testing.T) {
	k := &Katalog{enabledCRDs: []orktypes.CRDEntry{
		{Name: "a"},
		{Name: "b", DependsOn: []string{"a"}},
		{Name: "c", DependsOn: []string{"b"}},
	}}
	if err := k.detectDependencyCycles(); err != nil {
		t.Errorf("expected no error for acyclic graph, got %v", err)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// assertBefore fails if 'before' does not appear earlier than 'after' in order.
func assertBefore(t *testing.T, order []string, before, after string) {
	t.Helper()
	idx := make(map[string]int, len(order))
	for i, v := range order {
		idx[v] = i
	}
	bi, bOk := idx[before]
	ai, aOk := idx[after]
	if !bOk {
		t.Errorf("%q not found in order %v", before, order)
		return
	}
	if !aOk {
		t.Errorf("%q not found in order %v", after, order)
		return
	}
	if bi >= ai {
		sorted := append([]string{}, order...)
		sort.Strings(sorted)
		t.Errorf("expected %q before %q in order %v", before, after, order)
	}
}
