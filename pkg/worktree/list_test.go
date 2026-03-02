package worktree_test

import (
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/worktree"
)

type WorktreeListSuite struct {
	testutil.CmdSuite
}

func TestWorktreeListSuite(t *testing.T) {
	s := new(WorktreeListSuite)
	s.InitRoot(worktree.Register)
	testutil.RunSuite(t, s)
}

func (s *WorktreeListSuite) TestListSetsAndBranches() {
	_, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev", "test"}, []string{"dev", "test"})
	s.Require().NoError(err)

	allOut, err := s.ExecuteCmd("worktree", "list")
	s.Require().NoError(err)
	s.Contains(allOut, "infra")

	setOut, err := s.ExecuteCmd("worktree", "ls", "infra")
	s.Require().NoError(err)
	s.Contains(setOut, "dev")
	s.Contains(setOut, "test")
}

func (s *WorktreeListSuite) TestListErrorsForUnknownSet() {
	s.SetupWorkspaceDir()

	_, err := s.ExecuteCmd("worktree", "list", "missing")
	s.Require().Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *WorktreeListSuite) TestListEmpty() {
	s.SetupWorkspaceDir()

	out, err := s.ExecuteCmd("worktree", "list")
	s.Require().NoError(err)
	s.Assert().Empty(out)
}
