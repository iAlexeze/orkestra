package katalog

import (
	"strings"
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Helper to create a test CRD entry with all required fields
func testCRDEntry(kind, target string, idpEnabled bool) orktypes.CRDEntry {
	entry := orktypes.CRDEntry{
		APITypes: orktypes.APITypes{
			Group:   "platform.myorg.io",
			Version: "v1",
			Kind:    kind,
			Plural:  kind + "s",
		},
	}
	if idpEnabled || target != "" {
		entry.IDP = &orktypes.IDPConfig{
			Enabled: true,
			Target:  target,
		}
	}
	return entry
}

func TestBuildIndexes(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app":      testCRDEntry("App", "smartapp", true),
			"database": testCRDEntry("Database", "db", true),
		},
	}

	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("setGroupVersionKind: %v", err)
	}

	// Test kind index — stored lowercase for case-insensitive lookups.
	if name, ok := k.kindIndex["app"]; !ok || name != "app" {
		t.Errorf("kindIndex: expected 'app', got '%s'", name)
	}
	if name, ok := k.kindIndex["database"]; !ok || name != "database" {
		t.Errorf("kindIndex: expected 'database', got '%s'", name)
	}

	// Test target index
	if name, ok := k.targetIndex["smartapp"]; !ok || name != "app" {
		t.Errorf("targetIndex: expected 'app', got '%s'", name)
	}
	if name, ok := k.targetIndex["db"]; !ok || name != "database" {
		t.Errorf("targetIndex: expected 'database', got '%s'", name)
	}

	// Test GVK index
	appGVK := schema.GroupVersionKind{
		Group:   "platform.myorg.io",
		Version: "v1",
		Kind:    "App",
	}
	if name, ok := k.gvkIndex[strings.ToLower(appGVK.String())]; !ok || name != "app" {
		t.Errorf("gvkIndex: expected 'app', got '%s'", name)
	}

	// Test GVR index
	appGVR := schema.GroupVersionResource{
		Group:    "platform.myorg.io",
		Version:  "v1",
		Resource: "apps",
	}
	if name, ok := k.gvrIndex[strings.ToLower(appGVR.String())]; !ok || name != "app" {
		t.Errorf("gvrIndex: expected 'app', got '%s'", name)
	}
}

func TestBuildAllIDEnabledCRDs(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app":      testCRDEntry("App", "smartapp", true),
			"database": testCRDEntry("Database", "", false),
			"cache": {
				APITypes: orktypes.APITypes{Kind: "Cache"},
				IDP:      nil,
			},
		},
	}

	idpEnabledCrds := k.BuildIDPEnabledCRDs()

	if len(idpEnabledCrds) != 1 {
		t.Fatalf("expected 1 IDP-enabled CRD, got %d", len(idpEnabledCrds))
	}

	if idpEnabledCrds[0].APITypes.Kind != "App" {
		t.Errorf("expected App, got %s", idpEnabledCrds[0].APITypes.Kind)
	}
	if idpEnabledCrds[0].IDP.Target != "smartapp" {
		t.Errorf("expected target smartapp, got %s", idpEnabledCrds[0].IDP.Target)
	}
}

func TestLookupByKind(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app":      testCRDEntry("App", "", false),
			"database": testCRDEntry("Database", "", false),
		},
	}
	k.setGroupVersionKind()

	tests := []struct {
		name     string
		kind     string
		expected string
	}{
		{"found", "App", "app"},
		{"found", "Database", "database"},
		{"not found", "Unknown", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crd := k.LookupByKind(tt.kind)
			if tt.expected == "" {
				if crd != nil {
					t.Errorf("expected nil, got %+v", crd)
				}
				return
			}
			if crd == nil {
				t.Fatalf("expected CRD, got nil")
			}
			if crd.APITypes.Kind != tt.kind {
				t.Errorf("expected kind %s, got %s", tt.kind, crd.APITypes.Kind)
			}
		})
	}
}

func TestLookupByTarget(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app":      testCRDEntry("App", "smartapp", true),
			"database": testCRDEntry("Database", "db", true),
		},
	}
	k.setGroupVersionKind()

	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{"found", "smartapp", "App"},
		{"found", "db", "Database"},
		{"not found", "unknown", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crd := k.LookupByTarget(tt.target)
			if tt.expected == "" {
				if crd != nil {
					t.Errorf("expected nil, got %+v", crd)
				}
				return
			}
			if crd == nil {
				t.Fatalf("expected CRD, got nil")
			}
			if crd.APITypes.Kind != tt.expected {
				t.Errorf("expected kind %s, got %s", tt.expected, crd.APITypes.Kind)
			}
		})
	}
}

func TestLookupByName(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app":      testCRDEntry("App", "smartapp", true),
			"database": testCRDEntry("Database", "db", true),
		},
	}
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("setGroupVersionKind: %v", err)
	}

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{"exact match", "app", "App"},
		{"case-insensitive", "Database", "Database"},
		{"whitespace trimmed", "  app  ", "App"},
		{"not found", "unknown", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crd := k.LookupByName(tt.query)
			if tt.expected == "" {
				if crd != nil {
					t.Errorf("expected nil, got %+v", crd)
				}
				return
			}
			if crd == nil {
				t.Fatalf("expected CRD, got nil")
			}
			if crd.APITypes.Kind != tt.expected {
				t.Errorf("expected kind %s, got %s", tt.expected, crd.APITypes.Kind)
			}
		})
	}
}

func TestLookupByTargetOrKind(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app":      testCRDEntry("App", "smartapp", true),
			"database": testCRDEntry("Database", "db", true),
			"cache":    testCRDEntry("Cache", "", false),
		},
	}
	k.setGroupVersionKind()

	tests := []struct {
		name       string
		identifier string
		expected   string
	}{
		{"target match", "smartapp", "App"},
		{"target match", "db", "Database"},
		{"kind match", "Cache", "Cache"},
		{"kind match", "App", "App"},
		{"not found", "unknown", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crd := k.LookupByTargetOrKind(tt.identifier)
			if tt.expected == "" {
				if crd != nil {
					t.Errorf("expected nil, got %+v", crd)
				}
				return
			}
			if crd == nil {
				t.Fatalf("expected CRD, got nil")
			}
			if crd.APITypes.Kind != tt.expected {
				t.Errorf("expected kind %s, got %s", tt.expected, crd.APITypes.Kind)
			}
		})
	}
}

func TestLookupByGVKString(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v1",
					Kind:    "App",
					Plural:  "apps",
				},
			},
		},
	}
	// set defaults
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("failed to set defaults: %v", err)
	}
	k.setGroupVersionKind()

	// Use the exact format that GVKString() returns
	gvk := schema.GroupVersionKind{
		Group:   "platform.myorg.io",
		Version: "v1",
		Kind:    "App",
	}

	// Debug: print what the index keys are
	t.Logf("Index keys: %v", k.gvkIndex)
	t.Logf("GVK.String(): %q", gvk.String())

	crd := k.LookupByGVKString(gvk.String())
	if crd == nil {
		t.Fatal("expected CRD, got nil")
	}
	if crd.APITypes.Kind != "App" {
		t.Errorf("expected App, got %s", crd.APITypes.Kind)
	}

	// Test not found
	unknownGVK := schema.GroupVersionKind{
		Group:   "unknown.io",
		Version: "v1",
		Kind:    "Unknown",
	}
	crd = k.LookupByGVKString(unknownGVK.String())
	if crd != nil {
		t.Errorf("expected nil, got %+v", crd)
	}
}

func TestLookupByGVRString(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v1",
					Kind:    "App",
					Plural:  "apps",
				},
			},
		},
	}
	// set defaults
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("failed to set defaults: %v", err)
	}
	k.setGroupVersionKind()

	gvr := schema.GroupVersionResource{
		Group:    "platform.myorg.io",
		Version:  "v1",
		Resource: "apps",
	}

	// Debug: print what the index keys are
	t.Logf("Index keys: %v", k.gvrIndex)
	t.Logf("GVR.String(): %q", gvr.String())

	crd := k.LookupByGVRString(gvr.String())
	if crd == nil {
		t.Fatal("expected CRD, got nil")
	}
	if crd.APITypes.Kind != "App" {
		t.Errorf("expected App, got %s", crd.APITypes.Kind)
	}

	// Test not found
	unknownGVR := schema.GroupVersionResource{
		Group:    "unknown.io",
		Version:  "v1",
		Resource: "unknowns",
	}
	crd = k.LookupByGVRString(unknownGVR.String())
	if crd != nil {
		t.Errorf("expected nil, got %+v", crd)
	}
}

func TestMustLookupByTarget(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": testCRDEntry("App", "smartapp", true),
		},
	}
	k.setGroupVersionKind()

	// Should not panic
	crd := k.MustLookupByTarget("smartapp")
	if crd == nil {
		t.Fatal("expected CRD, got nil")
	}
	if crd.APITypes.Kind != "App" {
		t.Errorf("expected App, got %s", crd.APITypes.Kind)
	}

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown target")
		}
	}()
	k.MustLookupByTarget("unknown")
}

func TestMustLookupByKind(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v1",
					Kind:    "App",
					Plural:  "apps",
				},
			},
		},
	}
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("setGroupVersionKind: %v", err)
	}

	// Should not panic
	crd := k.MustLookupByKind("App")
	if crd == nil {
		t.Fatal("expected CRD, got nil")
	}
	if crd.APITypes.Kind != "App" {
		t.Errorf("expected App, got %s", crd.APITypes.Kind)
	}

	// Should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown kind")
		}
	}()
	k.MustLookupByKind("Unknown")
}

func TestIsKindRegistered(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": testCRDEntry("App", "", false),
		},
	}
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("setGroupVersionKind: %v", err)
	}

	if !k.IsKindRegistered("App") {
		t.Error("expected App to be registered")
	}
	if k.IsKindRegistered("Unknown") {
		t.Error("expected Unknown to not be registered")
	}
}

func TestIsTargetRegistered(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": testCRDEntry("App", "smartapp", true),
		},
	}
	k.setGroupVersionKind()

	if !k.IsTargetRegistered("smartapp") {
		t.Error("expected smartapp to be registered")
	}
	if k.IsTargetRegistered("unknown") {
		t.Error("expected unknown to not be registered")
	}
}

func TestListTargets(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app":      testCRDEntry("App", "smartapp", true),
			"database": testCRDEntry("Database", "db", true),
		},
	}
	k.setGroupVersionKind()

	targets := k.ListTargets()
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	found := make(map[string]bool)
	for _, t := range targets {
		found[t] = true
	}
	if !found["smartapp"] {
		t.Error("expected smartapp in targets")
	}
	if !found["db"] {
		t.Error("expected db in targets")
	}
}

func TestListKinds(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app":      testCRDEntry("App", "", false),
			"database": testCRDEntry("Database", "", false),
		},
	}
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("setGroupVersionKind: %v", err)
	}

	kinds := k.ListKinds()
	if len(kinds) != 2 {
		t.Fatalf("expected 2 kinds, got %d", len(kinds))
	}

	found := make(map[string]bool)
	for _, k := range kinds {
		found[k] = true
	}
	// ListKinds reads kindIndex, which is stored lowercase.
	if !found["app"] {
		t.Error("expected app in kinds")
	}
	if !found["database"] {
		t.Error("expected database in kinds")
	}
}

func TestLookup_EmptyKatalog(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{},
	}
	k.setGroupVersionKind()

	if crd := k.LookupByKind("App"); crd != nil {
		t.Errorf("expected nil, got %+v", crd)
	}
	if crd := k.LookupByTarget("smartapp"); crd != nil {
		t.Errorf("expected nil, got %+v", crd)
	}
	if crd := k.LookupByTargetOrKind("App"); crd != nil {
		t.Errorf("expected nil, got %+v", crd)
	}
}

func TestLookup_CRDWithoutIDP(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"cache": testCRDEntry("Cache", "", false),
		},
	}
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("setGroupVersionKind: %v", err)
	}

	// Should still find by kind
	if crd := k.LookupByKind("Cache"); crd == nil {
		t.Error("expected to find Cache by kind")
	}

	// Should not find by target (no IDP target)
	if crd := k.LookupByTarget("cache"); crd != nil {
		t.Errorf("expected nil for target, got %+v", crd)
	}
}

func TestLookupByKind_CaseInsensitive(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": testCRDEntry("App", "", false),
		},
	}
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("setGroupVersionKind: %v", err)
	}

	// Exact match should work
	if crd := k.LookupByKind("App"); crd == nil {
		t.Error("expected to find App by exact match")
	}

	// Case-insensitive match should work
	if crd := k.LookupByKind("app"); crd == nil {
		t.Error("expected to find App by case-insensitive match")
	}
	if crd := k.LookupByKind("APP"); crd == nil {
		t.Error("expected to find App by case-insensitive match")
	}
}

func TestLookupByAPIVersionAndKind(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v1",
					Kind:    "App",
					Plural:  "apps",
				},
			},
			"database": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v1beta1",
					Kind:    "Database",
					Plural:  "databases",
				},
			},
			"config": {
				APITypes: orktypes.APITypes{
					Group:   "config.myorg.io",
					Version: "v1",
					Kind:    "Config",
					Plural:  "configs",
				},
			},
		},
	}
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("setGroupVersionKind: %v", err)
	}

	tests := []struct {
		name       string
		apiVersion string
		kind       string
		wantKind   string
		wantNil    bool
	}{
		{
			name:       "exact match - core group",
			apiVersion: "platform.myorg.io/v1",
			kind:       "App",
			wantKind:   "App",
			wantNil:    false,
		},
		{
			name:       "exact match - different version",
			apiVersion: "platform.myorg.io/v1beta1",
			kind:       "Database",
			wantKind:   "Database",
			wantNil:    false,
		},
		{
			name:       "exact match - different group",
			apiVersion: "config.myorg.io/v1",
			kind:       "Config",
			wantKind:   "Config",
			wantNil:    false,
		},
		{
			name:       "case insensitive - apiVersion",
			apiVersion: "Platform.myorg.io/V1",
			kind:       "App",
			wantKind:   "App",
			wantNil:    false,
		},
		{
			name:       "case insensitive - kind",
			apiVersion: "platform.myorg.io/v1",
			kind:       "app",
			wantKind:   "App",
			wantNil:    false,
		},
		{
			name:       "case insensitive - both",
			apiVersion: "Platform.myorg.io/V1",
			kind:       "app",
			wantKind:   "App",
			wantNil:    false,
		},
		{
			name:       "with spaces - apiVersion",
			apiVersion: "  platform.myorg.io/v1  ",
			kind:       "App",
			wantKind:   "App",
			wantNil:    false,
		},
		{
			name:       "with spaces - kind",
			apiVersion: "platform.myorg.io/v1",
			kind:       "  App  ",
			wantKind:   "App",
			wantNil:    false,
		},
		{
			name:       "not found - wrong group",
			apiVersion: "unknown.io/v1",
			kind:       "App",
			wantNil:    true,
		},
		{
			name:       "not found - wrong version",
			apiVersion: "platform.myorg.io/v2",
			kind:       "App",
			wantNil:    true,
		},
		{
			name:       "not found - wrong kind",
			apiVersion: "platform.myorg.io/v1",
			kind:       "Unknown",
			wantNil:    true,
		},
		{
			name:       "not found - wrong group and kind",
			apiVersion: "unknown.io/v1",
			kind:       "Unknown",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crd := k.LookupByAPIVersionAndKind(tt.apiVersion, tt.kind)
			if tt.wantNil {
				if crd != nil {
					t.Errorf("expected nil, got %+v", crd)
				}
				return
			}
			if crd == nil {
				t.Fatal("expected CRD, got nil")
			}
			if crd.APITypes.Kind != tt.wantKind {
				t.Errorf("expected %q, got %q", tt.wantKind, crd.APITypes.Kind)
			}
			if wantGroup := strings.Split(tt.apiVersion, "/")[0]; !strings.EqualFold(crd.APITypes.Group, strings.TrimSpace(wantGroup)) {
				t.Errorf("expected group %q, got %q", wantGroup, crd.APITypes.Group)
			}
		})
	}
}

func TestLookupByAPIVersionAndKind_EmptyInput(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v1",
					Kind:    "App",
					Plural:  "apps",
				},
			},
		},
	}
	k.setGroupVersionKind()

	tests := []struct {
		name       string
		apiVersion string
		kind       string
	}{
		{"empty apiVersion", "", "App"},
		{"empty kind", "platform.myorg.io/v1", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crd := k.LookupByAPIVersionAndKind(tt.apiVersion, tt.kind)
			if crd != nil {
				t.Errorf("expected nil for empty input, got %+v", crd)
			}
		})
	}
}

func TestLookupByAPIVersionAndKind_IndexExists(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v1",
					Kind:    "App",
					Plural:  "apps",
				},
			},
		},
	}
	// set defaults
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("failed to set defaults: %v", err)
	}
	k.setGroupVersionKind()

	// Check that the index was built correctly
	expectedKey := strings.ToLower("platform.myorg.io/v1" + "App")
	if _, ok := k.apiVersionIndex[expectedKey]; !ok {
		t.Errorf("index key %q not found in apiVersionIndex", expectedKey)
	}

	// Verify the lookup works with the exact key
	crd := k.LookupByAPIVersionAndKind("platform.myorg.io/v1", "App")
	if crd == nil {
		t.Fatal("expected CRD, got nil")
	}
	if crd.APITypes.Kind != "App" {
		t.Errorf("expected App, got %s", crd.APITypes.Kind)
	}
}

func TestLookupByAPIVersionAndKind_MultipleEntries(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v1",
					Kind:    "App",
					Plural:  "apps",
				},
			},
			"appv2": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v2",
					Kind:    "App",
					Plural:  "apps",
				},
			},
			"database": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v1",
					Kind:    "Database",
					Plural:  "databases",
				},
			},
		},
	}
	// set defaults
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("failed to set defaults: %v", err)
	}
	k.setGroupVersionKind()

	// Should find the v1 App
	crd := k.LookupByAPIVersionAndKind("platform.myorg.io/v1", "App")
	if crd == nil {
		t.Fatal("expected App v1, got nil")
	}
	if crd.APITypes.Version != "v1" {
		t.Errorf("expected version v1, got %s", crd.APITypes.Version)
	}

	// Should find the v2 App
	crd = k.LookupByAPIVersionAndKind("platform.myorg.io/v2", "App")
	if crd == nil {
		t.Fatal("expected App v2, got nil")
	}
	if crd.APITypes.Version != "v2" {
		t.Errorf("expected version v2, got %s", crd.APITypes.Version)
	}

	// Should find the Database
	crd = k.LookupByAPIVersionAndKind("platform.myorg.io/v1", "Database")
	if crd == nil {
		t.Fatal("expected Database, got nil")
	}
	if crd.APITypes.Kind != "Database" {
		t.Errorf("expected Database, got %s", crd.APITypes.Kind)
	}
}

func TestLookupByAPIVersionAndKind_InvalidAPIVersion(t *testing.T) {
	k := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": {
				APITypes: orktypes.APITypes{
					Group:   "platform.myorg.io",
					Version: "v1",
					Kind:    "App",
					Plural:  "apps",
				},
			},
		},
	}
	// set defaults
	if err := k.setGroupVersionKind(); err != nil {
		t.Fatalf("failed to set defaults: %v", err)
	}
	k.setGroupVersionKind()

	// Test with malformed apiVersion
	tests := []struct {
		name       string
		apiVersion string
		kind       string
	}{
		{"no version", "platform.myorg.io", "App"},
		{"multiple slashes", "platform/myorg/io/v1", "App"},
		{"empty group", "/v1", "App"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crd := k.LookupByAPIVersionAndKind(tt.apiVersion, tt.kind)
			if crd != nil {
				t.Errorf("expected nil for malformed apiVersion %q, got %+v", tt.apiVersion, crd)
			}
		})
	}
}
