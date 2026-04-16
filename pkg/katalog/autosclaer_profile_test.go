package katalog_test

import (
	"testing"
	"time"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func baseline(workers, queue int, resync time.Duration) orktypes.AutoscaleBaseline {
	return orktypes.AutoscaleBaseline{
		Workers:    workers,
		QueueDepth: queue,
		Resync:     resync,
	}
}

func TestAutoscalerProfiles_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		profile        string
		baseline       orktypes.AutoscaleBaseline
		expectErr      bool
		expectWorkers  int
		expectQueue    int
		expectField    string
		expectInterval time.Duration
		expectCooldown time.Duration
	}{
		{
			name:           "burst profile expands correctly",
			profile:        "burst",
			baseline:       baseline(4, 100, 120*time.Second),
			expectWorkers:  16,   // 4 * 4
			expectQueue:    1000, // 100 * 10
			expectField:    "metrics.queueDepth",
			expectInterval: 5 * time.Second,
			expectCooldown: 30 * time.Second,
		},
		{
			name:           "steady profile expands correctly",
			profile:        "steady",
			baseline:       baseline(4, 100, 120*time.Second),
			expectWorkers:  8,   // 4 * 2
			expectQueue:    300, // 100 * 3
			expectField:    "metrics.queueDepth",
			expectInterval: 30 * time.Second,
			expectCooldown: 2 * time.Minute,
		},
		{
			name:           "batch profile expands correctly",
			profile:        "batch",
			baseline:       baseline(4, 100, 120*time.Second),
			expectWorkers:  12,  // 4 * 3
			expectQueue:    800, // 100 * 8
			expectField:    "",  // cron profile uses AnyOf, not When
			expectInterval: 60 * time.Second,
			expectCooldown: 5 * time.Minute,
		},
		{
			name:           "latency-sensitive profile expands correctly",
			profile:        "latency-sensitive",
			baseline:       baseline(4, 100, 120*time.Second),
			expectWorkers:  10, // ceil(4 * 2.5)
			expectQueue:    0,  // no queue override
			expectField:    "metrics.reconcileDurationP95Ms",
			expectInterval: 15 * time.Second,
			expectCooldown: 1 * time.Minute,
		},
		{
			name:           "cost-optimized profile expands correctly",
			profile:        "cost-optimized",
			baseline:       baseline(4, 100, 120*time.Second),
			expectWorkers:  2,  // max(1, 4 * 0.5)
			expectQueue:    50, // 100 * 0.5
			expectField:    "metrics.workersIdlePercent",
			expectInterval: 30 * time.Second,
			expectCooldown: 10 * time.Minute,
		},
		{
			name:      "unknown profile fails fast",
			profile:   "unknown",
			baseline:  baseline(4, 100, 120*time.Second),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			spec, err := katalog.ApplyAutoscalerProfile(tt.profile, tt.baseline)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Validate interval
			if spec.Interval.Duration != tt.expectInterval {
				t.Errorf("expected interval %v, got %v", tt.expectInterval, spec.Interval.Duration)
			}

			// Validate cooldown
			if spec.Cooldown.Duration != tt.expectCooldown {
				t.Errorf("expected cooldown %v, got %v", tt.expectCooldown, spec.Cooldown.Duration)
			}

			// Validate workers override
			if tt.expectWorkers > 0 {
				if spec.Do.Workers == nil || *spec.Do.Workers != tt.expectWorkers {
					t.Errorf("expected workers %d, got %v", tt.expectWorkers, spec.Do.Workers)
				}
			}

			// Validate queue override
			if tt.expectQueue > 0 {
				if spec.Do.QueueDepth == nil || *spec.Do.QueueDepth != tt.expectQueue {
					t.Errorf("expected queueDepth %d, got %v", tt.expectQueue, spec.Do.QueueDepth)
				}
			}

			// Validate condition field (if applicable)
			if tt.expectField != "" {
				if len(spec.Conditions.When) == 0 {
					t.Fatalf("expected When conditions but got none")
				}
				if spec.Conditions.When[0].Field != tt.expectField {
					t.Errorf("expected field %s, got %s", tt.expectField, spec.Conditions.When[0].Field)
				}
			}
		})
	}
}
