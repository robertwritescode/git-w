package worktree_test

import "testing"

type worktreeCmdSuite interface {
	SetupWorkspaceDir() string
	MakeRemoteWithBranches(branches []string) string
	ExecuteCmd(args ...string) (string, error)
}

func setupClonedWorktreeSet(t *testing.T, s worktreeCmdSuite, setName string, remoteBranches, cloneBranches []string) (string, string, error) {
	t.Helper()

	wsDir := s.SetupWorkspaceDir()
	remoteURL := s.MakeRemoteWithBranches(remoteBranches)
	args := append([]string{"worktree", "clone", remoteURL, setName}, cloneBranches...)
	_, err := s.ExecuteCmd(args...)

	return wsDir, remoteURL, err
}
