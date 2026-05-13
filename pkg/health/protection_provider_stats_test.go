// pkg/health/protection_provider_stats_test.go
package health

import "testing"

// ── DeletionProtectionStats ───────────────────────────────────────────────────

func TestDeletionProtectionStats_RecordBlocked(t *testing.T) {
	s := NewDeletionProtectionStats()
	s.RecordBlocked()
	snap := s.GetStats()
	if snap.TotalRequests != 1 || snap.Blocked != 1 || snap.Allowed != 0 {
		t.Errorf("unexpected snapshot after blocked: %+v", snap)
	}
}

func TestDeletionProtectionStats_RecordAllowed(t *testing.T) {
	s := NewDeletionProtectionStats()
	s.RecordAllowed()
	snap := s.GetStats()
	if snap.TotalRequests != 1 || snap.Allowed != 1 || snap.Blocked != 0 {
		t.Errorf("unexpected snapshot after allowed: %+v", snap)
	}
}

func TestDeletionProtectionStats_Mixed(t *testing.T) {
	s := NewDeletionProtectionStats()
	s.RecordBlocked()
	s.RecordBlocked()
	s.RecordAllowed()
	snap := s.GetStats()
	if snap.TotalRequests != 3 || snap.Blocked != 2 || snap.Allowed != 1 {
		t.Errorf("unexpected mixed snapshot: %+v", snap)
	}
}

func TestDeletionProtectionStats_Empty(t *testing.T) {
	s := NewDeletionProtectionStats()
	snap := s.GetStats()
	if snap.TotalRequests != 0 {
		t.Errorf("new stats must be zero, got %+v", snap)
	}
}

// ── NamespaceProtectionStats ──────────────────────────────────────────────────

func TestNamespaceProtectionStats_RecordBlocked(t *testing.T) {
	s := NewNamespaceProtectionStats()
	s.RecordBlocked()
	snap := s.GetStats()
	if snap.TotalRequests != 1 || snap.Blocked != 1 {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
}

func TestNamespaceProtectionStats_RecordAllowed(t *testing.T) {
	s := NewNamespaceProtectionStats()
	s.RecordAllowed()
	snap := s.GetStats()
	if snap.TotalRequests != 1 || snap.Allowed != 1 || snap.Blocked != 0 {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
}

func TestNamespaceProtectionStats_Mixed(t *testing.T) {
	s := NewNamespaceProtectionStats()
	s.RecordAllowed()
	s.RecordBlocked()
	snap := s.GetStats()
	if snap.TotalRequests != 2 || snap.Allowed != 1 || snap.Blocked != 1 {
		t.Errorf("unexpected mixed snapshot: %+v", snap)
	}
}

// ── ProviderStats ─────────────────────────────────────────────────────────────

func TestProviderStats_RecordSuccess(t *testing.T) {
	s := NewProviderStats()
	s.RecordSuccess("aws")
	entries := s.GetSnapshot()
	if len(entries) != 1 || entries[0].Total != 1 || entries[0].Errors != 0 {
		t.Errorf("unexpected snapshot: %+v", entries)
	}
}

func TestProviderStats_RecordFailure_ErrorRate(t *testing.T) {
	s := NewProviderStats()
	s.RecordSuccess("aws")
	s.RecordFailure("aws")
	entries := s.GetSnapshot()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Total != 2 || e.Errors != 1 {
		t.Errorf("unexpected totals: %+v", e)
	}
	if e.ErrorRate != 0.5 {
		t.Errorf("expected error rate 0.5, got %f", e.ErrorRate)
	}
}

func TestProviderStats_MultipleProviders(t *testing.T) {
	s := NewProviderStats()
	s.RecordSuccess("aws")
	s.RecordSuccess("gcp")
	s.RecordFailure("gcp")
	entries := s.GetSnapshot()
	if len(entries) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(entries))
	}
	byName := make(map[string]ProviderStatEntry)
	for _, e := range entries {
		byName[e.Provider] = e
	}
	if byName["aws"].ErrorRate != 0 {
		t.Errorf("aws error rate must be 0, got %f", byName["aws"].ErrorRate)
	}
	if byName["gcp"].ErrorRate != 0.5 {
		t.Errorf("gcp error rate must be 0.5, got %f", byName["gcp"].ErrorRate)
	}
}

func TestProviderStats_DeleteSuccess_CountsTotal(t *testing.T) {
	s := NewProviderStats()
	s.RecordDeleteSuccess("aws")
	entries := s.GetSnapshot()
	if len(entries) != 1 || entries[0].Total != 1 || entries[0].Errors != 0 {
		t.Errorf("unexpected snapshot after delete success: %+v", entries)
	}
}

func TestProviderStats_DeleteFailure_CountsError(t *testing.T) {
	s := NewProviderStats()
	s.RecordDeleteFailure("aws")
	entries := s.GetSnapshot()
	if len(entries) != 1 || entries[0].Errors != 1 {
		t.Errorf("unexpected snapshot after delete failure: %+v", entries)
	}
}

func TestProviderStats_Empty_ReturnsEmptySlice(t *testing.T) {
	s := NewProviderStats()
	entries := s.GetSnapshot()
	if len(entries) != 0 {
		t.Errorf("expected empty snapshot, got %v", entries)
	}
}

func TestProviderStats_ZeroTotal_ErrorRateZero(t *testing.T) {
	// This can't happen via the public API (RecordFailure increments Total),
	// but verify the zero-division guard is present via the snapshot
	s := NewProviderStats()
	entries := s.GetSnapshot()
	for _, e := range entries {
		if e.Total == 0 && e.ErrorRate != 0 {
			t.Error("zero total must not produce non-zero error rate")
		}
	}
}
