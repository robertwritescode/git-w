package repo

import (
	"context"
	"fmt"
)

// SafetyViolations checks a repo for uncommitted changes and unpushed commits,
// returning human-readable violation messages. Used by drop-safety checks.
func SafetyViolations(ctx context.Context, r Repo) ([]string, error) {
	status, err := GetStatus(ctx, r)
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

func hasUncommittedChanges(s RepoStatus) bool {
	return s.Dirty || s.Staged || s.Untracked
}

func hasUnpushedCommits(s RepoStatus) bool {
	return s.RemoteState == LocalAhead || s.RemoteState == Diverged
}
