package worktree

import (
	"fmt"

	"github.com/robertwritescode/git-w/pkg/repo"
)

func safetyViolations(r repo.Repo) ([]string, error) {
	status, err := repo.GetStatus(r)
	if err != nil {
		return nil, fmt.Errorf("checking status for %q: %w", r.Name, err)
	}

	var violations []string
	if hasUncommittedChanges(status) {
		violations = append(violations, "working tree has uncommitted changes")
	}

	if hasUnpushedCommits(status) {
		violations = append(violations, "branch has local commits not fully pushed")
	}

	return violations, nil
}

func hasUncommittedChanges(status repo.RepoStatus) bool {
	return status.Dirty || status.Staged || status.Untracked
}

func hasUnpushedCommits(status repo.RepoStatus) bool {
	return status.RemoteState == repo.LocalAhead || status.RemoteState == repo.Diverged
}
