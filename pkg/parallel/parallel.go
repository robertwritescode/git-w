package parallel

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// MaxWorkers returns the bounded number of concurrent workers to use.
// Falls back to runtime.NumCPU() when configured is 0, and caps at total.
// The minimum return value is 1.
func MaxWorkers(configured, total int) int {
	if total <= 0 {
		return 1
	}

	if configured <= 0 {
		configured = runtime.NumCPU()
	}

	if configured > total {
		return total
	}

	return configured
}

// RunFanOut executes fn for each item using up to workers goroutines.
// Results are returned in the same order as items.
func RunFanOut[T any, R any](items []T, workers int, fn func(T) R) []R {
	results := make([]R, len(items))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, it T) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = fn(it)
		}(i, item)
	}
	wg.Wait()

	return results
}

// FormatFailureError returns a summary error for failed operations, or nil if
// there are no failures. Each entry in failures should be a pre-formatted line.
func FormatFailureError(failures []string, total int) error {
	if len(failures) == 0 {
		return nil
	}

	return fmt.Errorf("%d of %d repos failed:\n%s",
		len(failures), total, strings.Join(failures, "\n"))
}
