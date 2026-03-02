package worktree_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func assertSafetyRefusal(t *testing.T, err error, reason string) {
	t.Helper()

	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing")

	if reason != "" {
		require.Contains(t, err.Error(), reason)
	}
}
