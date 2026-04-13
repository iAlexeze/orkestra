// pkg/health/protection_stats.go
package health

import "sync"

// ProtectionStats tracks counters for the /deletion-protection webhook endpoint.
// Thread-safe for concurrent updates from the deletion protection handler.
//
// Follows the same pattern as ConversionStats and AdmissionStats.
// No latency tracking — deletion protection decisions are fast in-memory
// lookups and the blocking count is the meaningful signal.
type ProtectionStats struct {
	mu sync.RWMutex

	TotalRequests int64 // total DELETE admission reviews received
	Blocked       int64 // DELETE requests denied (CRD or deployment protected)
	Allowed       int64 // DELETE requests allowed through
}

// NewProtectionStats creates a new ProtectionStats instance.
func NewProtectionStats() *ProtectionStats {
	return &ProtectionStats{}
}

// RecordBlocked records a DELETE that was denied by the webhook.
func (s *ProtectionStats) RecordBlocked() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalRequests++
	s.Blocked++
}

// RecordAllowed records a DELETE that was allowed through the webhook.
func (s *ProtectionStats) RecordAllowed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalRequests++
	s.Allowed++
}

// GetStats returns a point-in-time snapshot of protection statistics.
func (s *ProtectionStats) GetStats() ProtectionStatsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ProtectionStatsSnapshot{
		TotalRequests: s.TotalRequests,
		Blocked:       s.Blocked,
		Allowed:       s.Allowed,
	}
}

// ProtectionStatsSnapshot is a read-only point-in-time snapshot.
// Serialised into the /katalog/{crd} JSON response under the "protection" key.
type ProtectionStatsSnapshot struct {
	TotalRequests int64 `json:"total"`
	Blocked       int64 `json:"blocked"`
	Allowed       int64 `json:"allowed"`
}
