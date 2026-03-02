package worktree_test

import (
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/robertwritescode/git-w/pkg/worktree"
)

type WorktreeAddSuite struct {
	testutil.CmdSuite
}

func TestWorktreeAddSuite(t *testing.T) {
	s := new(WorktreeAddSuite)
	s.InitRoot(worktree.Register)
	testutil.RunSuite(t, s)
}

func (s *WorktreeAddSuite) TestAddBranch() {
	wsDir, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev", "test"}, []string{"dev"})
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("worktree", "add", "infra", "test")
	s.Require().NoError(err)

	cfg, err := workspace.Load(filepath.Join(wsDir, ".gitw"))
	s.Require().NoError(err)
	s.Assert().Equal("infra/test", cfg.Worktrees["infra"].Branches["test"])
	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "infra", "test")))
}

func (s *WorktreeAddSuite) TestAddErrors() {
	_, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev"}, []string{"dev"})
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("worktree", "add", "infra", "dev")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "already registered")

	_, err = s.ExecuteCmd("worktree", "add", "missing", "dev")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "not found")
}

func (s *WorktreeAddSuite) TestAddBranch_CustomPath() {
	wsDir, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev", "test"}, []string{"dev"})
	s.Require().NoError(err)

	customAbs := filepath.Join(wsDir, "custom", "test-tree")
	_, err = s.ExecuteCmd("worktree", "add", "infra", "test", customAbs)
	s.Require().NoError(err)

	cfg, err := workspace.Load(filepath.Join(wsDir, ".gitw"))
	s.Require().NoError(err)
	s.Assert().Equal(filepath.Join("custom", "test-tree"), cfg.Worktrees["infra"].Branches["test"])
	s.Assert().True(repo.IsGitRepo(customAbs))
}
