package worktree_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/robertwritescode/git-w/pkg/worktree"
)

type WorktreeDropSuite struct {
	testutil.CmdSuite
}

func TestWorktreeDropSuite(t *testing.T) {
	s := new(WorktreeDropSuite)
	s.InitRoot(worktree.Register)
	testutil.RunSuite(t, s)
}

func (s *WorktreeDropSuite) TestDropCleansManualGroupMembership() {
	wsDir, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev"}, []string{"dev"})
	s.Require().NoError(err)

	s.AppendGroup(wsDir, "mixed", "infra-dev")

	_, err = s.ExecuteCmd("worktree", "drop", "infra")
	s.Require().NoError(err)

	cfg, loadErr := workspace.Load(filepath.Join(wsDir, ".gitw"))
	s.Require().NoError(loadErr)
	s.Assert().NotContains(cfg.Groups["mixed"].Repos, "infra-dev")
}

func (s *WorktreeDropSuite) TestDropSafetyMatrix() {
	tests := []struct {
		name    string
		force   bool
		mutate  func(path string)
		wantErr bool
		errPart string
	}{
		{name: "clean no force", force: false, wantErr: false},
		{
			name:    "dirty no force",
			force:   false,
			wantErr: true,
			errPart: "uncommitted",
			mutate: func(path string) {
				s.Require().NoError(os.WriteFile(filepath.Join(path, "dirty.txt"), []byte("x"), 0o644))
			},
		},
		{
			name:    "dirty force",
			force:   true,
			wantErr: false,
			mutate: func(path string) {
				s.Require().NoError(os.WriteFile(filepath.Join(path, "dirty.txt"), []byte("x"), 0o644))
			},
		},
		{
			name:    "local ahead no force",
			force:   false,
			wantErr: true,
			errPart: "local commits",
			mutate: func(path string) {
				makeBranchLocalAhead(s.T(), path, "dev")
			},
		},
		{
			name:    "local ahead force",
			force:   true,
			wantErr: false,
			mutate: func(path string) {
				makeBranchLocalAhead(s.T(), path, "dev")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev", "test"}, []string{"dev", "test"})
			s.Require().NoError(err)

			targetPath := filepath.Join(wsDir, "infra", "dev")
			if tt.mutate != nil {
				tt.mutate(targetPath)
			}

			args := []string{"worktree", "drop", "infra"}
			if tt.force {
				args = []string{"worktree", "drop", "--force", "infra"}
			}

			_, err = s.ExecuteCmd(args...)
			if tt.wantErr {
				assertSafetyRefusal(s.T(), err, tt.errPart)
				return
			}

			s.Require().NoError(err)
			cfg, loadErr := workspace.Load(filepath.Join(wsDir, ".gitw"))
			s.Require().NoError(loadErr)
			_, exists := cfg.Worktrees["infra"]
			s.Assert().False(exists)
			s.Assert().NoDirExists(filepath.Join(wsDir, "infra", "dev"))
			s.Assert().NoDirExists(filepath.Join(wsDir, "infra", "test"))
		})
	}
}

func (s *WorktreeDropSuite) TestDropSkipsMissingBranchPathInSafetyCheck() {
	wsDir, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev", "test"}, []string{"dev", "test"})
	s.Require().NoError(err)

	s.Require().NoError(os.RemoveAll(filepath.Join(wsDir, "infra", "dev")))

	_, err = s.ExecuteCmd("worktree", "drop", "infra")
	s.Require().NoError(err)

	cfg, loadErr := workspace.Load(filepath.Join(wsDir, ".gitw"))
	s.Require().NoError(loadErr)
	_, exists := cfg.Worktrees["infra"]
	s.Assert().False(exists)
	s.Assert().NoDirExists(filepath.Join(wsDir, "infra", "test"))
}
