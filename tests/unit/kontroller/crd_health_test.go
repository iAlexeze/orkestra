// tests/unit/kontroller/crd_health_test.go
package kontroller_test

import (
	"fmt"
	"testing"

	"github.com/ialexeze/orkestra/pkg/kontroller"
	"github.com/stretchr/testify/assert"
)

func TestCRDHealth_InitialState(t *testing.T) {
	h := kontroller.NewCRDHealth("test")

	assert.False(t, h.IsHealthy())
	assert.False(t, h.Started())
	assert.Equal(t, int64(0), h.TotalReconciles())
	assert.Equal(t, int64(0), h.FailedReconciles())
	assert.Equal(t, float64(0), h.ErrorRate())
}

func TestCRDHealth_RecordSuccess(t *testing.T) {
	h := kontroller.NewCRDHealth("test")

	h.RecordSuccess()

	assert.True(t, h.IsHealthy())
	assert.Equal(t, int64(1), h.TotalReconciles())
	assert.Equal(t, int64(0), h.FailedReconciles())
	assert.Equal(t, int64(0), h.ConsecutiveFails())
}

func TestCRDHealth_RecordFailure(t *testing.T) {
	h := kontroller.NewCRDHealth("test")

	h.RecordFailure(fmt.Errorf("something went wrong"), 3)

	assert.False(t, h.IsHealthy())
	assert.Equal(t, int64(1), h.TotalReconciles())
	assert.Equal(t, int64(1), h.FailedReconciles())
	assert.Equal(t, int64(1), h.ConsecutiveFails())
	assert.Equal(t, "something went wrong", h.LastError())
}

func TestCRDHealth_RecordFailureExceedsThreshold(t *testing.T) {
	h := kontroller.NewCRDHealth("test")

	// First failure — still healthy (threshold=3)
	h.RecordFailure(fmt.Errorf("error 1"), 3)
	assert.True(t, h.IsHealthy())

	// Second failure — still healthy
	h.RecordFailure(fmt.Errorf("error 2"), 3)
	assert.True(t, h.IsHealthy())

	// Third failure — becomes degraded
	h.RecordFailure(fmt.Errorf("error 3"), 3)
	assert.False(t, h.IsHealthy())
}

func TestCRDHealth_SuccessResetsConsecutiveFails(t *testing.T) {
	h := kontroller.NewCRDHealth("test")

	h.RecordFailure(fmt.Errorf("error"), 3)
	h.RecordFailure(fmt.Errorf("error"), 3)
	assert.Equal(t, int64(2), h.ConsecutiveFails())

	h.RecordSuccess()
	assert.Equal(t, int64(0), h.ConsecutiveFails())
	assert.True(t, h.IsHealthy())
}

func TestCRDHealth_ErrorRate(t *testing.T) {
	h := kontroller.NewCRDHealth("test")

	h.RecordSuccess() // 1 total, 0 failed
	assert.Equal(t, float64(0), h.ErrorRate())

	h.RecordFailure(fmt.Errorf("error"), 3) // 2 total, 1 failed
	assert.Equal(t, 0.5, h.ErrorRate())

	h.RecordFailure(fmt.Errorf("error"), 3) // 3 total, 2 failed
	assert.InDelta(t, 0.666, h.ErrorRate(), 0.001)
}
