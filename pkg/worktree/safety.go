package worktree

import (
	"context"

	"github.com/robertwritescode/git-w/pkg/repo"
)

func safetyViolations(ctx context.Context, r repo.Repo) ([]string, error) {
	return repo.SafetyViolations(ctx, r)
}
