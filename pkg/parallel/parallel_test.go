package parallel

import (
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxWorkers(t *testing.T) {
	tests := []struct {
		name       string
		configured int
		total      int
		want       int
	}{
		{name: "zero total", configured: 8, total: 0, want: 1},
		{name: "configured zero falls back to NumCPU and caps", configured: 0, total: 5, want: min(runtime.NumCPU(), 5)},
		{name: "configured negative falls back to NumCPU and caps", configured: -1, total: 5, want: min(runtime.NumCPU(), 5)},
		{name: "configured above total", configured: 9, total: 4, want: 4},
		{name: "configured within total", configured: 3, total: 7, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MaxWorkers(tt.configured, tt.total))
		})
	}
}

func TestRunFanOut(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	results := RunFanOut(items, 2, func(n int) int {
		return n * 10
	})
	assert.Equal(t, []int{10, 20, 30, 40, 50}, results)
}

func TestRunFanOut_PreservesOrder(t *testing.T) {
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}

	results := RunFanOut(items, 4, func(n int) int {
		return n
	})

	assert.Equal(t, items, results)
}

func TestRunFanOut_ConcurrencyLimit(t *testing.T) {
	var peak atomic.Int32
	var current atomic.Int32
	const workers = 2

	items := make([]int, 20)
	RunFanOut(items, workers, func(_ int) int {
		cur := current.Add(1)
		for {
			p := peak.Load()
			if cur <= p {
				break
			}
			if peak.CompareAndSwap(p, cur) {
				break
			}
		}
		current.Add(-1)
		return 0
	})

	assert.LessOrEqual(t, int(peak.Load()), workers)
}

func TestFormatFailureError_NoFailures(t *testing.T) {
	assert.NoError(t, FormatFailureError(nil, 5))
}

func TestFormatFailureError_WithFailures(t *testing.T) {
	failures := []string{"  [a]: exit 1", "  [b]: exit 2"}
	err := FormatFailureError(failures, 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "2 of 3 repos failed")
	assert.Contains(t, err.Error(), "[a]: exit 1")
	assert.Contains(t, err.Error(), "[b]: exit 2")
}
