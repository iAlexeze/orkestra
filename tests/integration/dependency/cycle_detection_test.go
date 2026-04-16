//go:build integration

// tests/integration/dependency/cycle_detection_test.go
package dependency_test

import (
	"testing"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func TestCycleDetection_TwoNodeCycle(t *testing.T) {
	k := katalog.NewKatalogForTest([]orktypes.CRDEntry{
		{Name: "a", DependsOn: []string{"b"}},
		{Name: "b", DependsOn: []string{"a"}},
	})
	if err := katalog.DetectCyclesForTest(k); err == nil {
		t.Error("expected cycle error for a ↔ b")
	}
}

func TestCycleDetection_ThreeNodeCycle(t *testing.T) {
	k := katalog.NewKatalogForTest([]orktypes.CRDEntry{
		{Name: "a", DependsOn: []string{"c"}},
		{Name: "b", DependsOn: []string{"a"}},
		{Name: "c", DependsOn: []string{"b"}},
	})
	if err := katalog.DetectCyclesForTest(k); err == nil {
		t.Error("expected cycle error for a → c → b → a")
	}
}

func TestCycleDetection_SelfLoop(t *testing.T) {
	k := katalog.NewKatalogForTest([]orktypes.CRDEntry{
		{Name: "a", DependsOn: []string{"a"}},
	})
	if err := katalog.DetectCyclesForTest(k); err == nil {
		t.Error("expected cycle error for self-loop")
	}
}

func TestCycleDetection_AcyclicGraph_NoError(t *testing.T) {
	k := katalog.NewKatalogForTest([]orktypes.CRDEntry{
		{Name: "db"},
		{Name: "cache"},
		{Name: "app", DependsOn: []string{"db", "cache"}},
	})
	if err := katalog.DetectCyclesForTest(k); err != nil {
		t.Errorf("acyclic graph must not produce cycle error: %v", err)
	}
}

func TestCycleDetection_EmptyGraph_NoError(t *testing.T) {
	k := katalog.NewKatalogForTest([]orktypes.CRDEntry{})
	if err := katalog.DetectCyclesForTest(k); err != nil {
		t.Errorf("empty graph must not produce cycle error: %v", err)
	}
}
