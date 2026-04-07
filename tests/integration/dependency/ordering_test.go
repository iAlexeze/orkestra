//go:build integration

// tests/integration/dependency/ordering_test.go
package dependency_test

import (
	"testing"

	"github.com/ialexeze/orkestra/pkg/katalog"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

func mkCRD(name string, dependsOn ...string) orktypes.CRDEntry {
	return orktypes.CRDEntry{Name: name, DependsOn: dependsOn}
}

func assertBefore(t *testing.T, order []string, first, second string) {
	t.Helper()
	fi, si := -1, -1
	for i, v := range order {
		if v == first {
			fi = i
		}
		if v == second {
			si = i
		}
	}
	if fi < 0 {
		t.Errorf("%q not in order %v", first, order)
		return
	}
	if si < 0 {
		t.Errorf("%q not in order %v", second, order)
		return
	}
	if fi >= si {
		t.Errorf("expected %q before %q in %v", first, second, order)
	}
}

func TestStartupOrder_LinearChain(t *testing.T) {
	k := katalog.NewKatalogForTest([]orktypes.CRDEntry{
		mkCRD("app", "cache"),
		mkCRD("cache", "database"),
		mkCRD("database"),
	})
	order := katalog.NewDependencyGraph(k).StartupOrder()
	assertBefore(t, order, "database", "cache")
	assertBefore(t, order, "cache", "app")
}

func TestStartupOrder_IndependentCRDs_AllPresent(t *testing.T) {
	k := katalog.NewKatalogForTest([]orktypes.CRDEntry{
		mkCRD("alpha"), mkCRD("beta"), mkCRD("gamma"),
	})
	if n := len(katalog.NewDependencyGraph(k).StartupOrder()); n != 3 {
		t.Errorf("expected 3 CRDs in order, got %d", n)
	}
}

func TestStartupOrder_DiamondDependency(t *testing.T) {
	//        base
	//       /    \
	//    left   right
	//       \    /
	//        app
	k := katalog.NewKatalogForTest([]orktypes.CRDEntry{
		mkCRD("app", "left", "right"),
		mkCRD("left", "base"),
		mkCRD("right", "base"),
		mkCRD("base"),
	})
	order := katalog.NewDependencyGraph(k).StartupOrder()
	assertBefore(t, order, "base", "left")
	assertBefore(t, order, "base", "right")
	assertBefore(t, order, "left", "app")
	assertBefore(t, order, "right", "app")
}

func TestShutdownOrder_ReversesStartup(t *testing.T) {
	k := katalog.NewKatalogForTest([]orktypes.CRDEntry{
		mkCRD("app", "db"),
		mkCRD("db"),
	})
	g := katalog.NewDependencyGraph(k)
	startup, shutdown := g.StartupOrder(), g.ShutdownOrder()
	for i := range startup {
		if startup[i] != shutdown[len(shutdown)-1-i] {
			t.Errorf("startup/shutdown mismatch at position %d", i)
		}
	}
}

func TestStartupOrder_SingleCRD(t *testing.T) {
	k := katalog.NewKatalogForTest([]orktypes.CRDEntry{mkCRD("solo")})
	order := katalog.NewDependencyGraph(k).StartupOrder()
	if len(order) != 1 || order[0] != "solo" {
		t.Errorf("expected [solo], got %v", order)
	}
}
