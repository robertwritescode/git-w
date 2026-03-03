package worktree_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/worktree"
)

type WorktreeRmSuite struct {
	testutil.CmdSuite
}

func TestWorktreeRmSuite(t *testing.T) {
	s := new(WorktreeRmSuite)
	s.InitRoot(worktree.Register)
	testutil.RunSuite(t, s)
}

func (s *WorktreeRmSuite) TestRmCleansManualGroupMembership() {
	wsDir, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev", "test"}, []string{"dev", "test"})
	s.Require().NoError(err)

	s.AppendGroup(wsDir, "mixed", "infra-test")

	_, err = s.ExecuteCmd("worktree", "rm", "infra-test")
	s.Require().NoError(err)

	cfg, loadErr := config.Load(filepath.Join(wsDir, ".gitw"))
	s.Require().NoError(loadErr)
	s.Assert().NotContains(cfg.Groups["mixed"].Repos, "infra-test")
}

func (s *WorktreeRmSuite) TestRmRefusesLastWorktree() {
	_, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev"}, []string{"dev"})
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("worktree", "rm", "infra-dev")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "last worktree")
}

func (s *WorktreeRmSuite) TestRmSafetyMatrix() {
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
				makeBranchLocalAhead(s.T(), path, "test")
			},
		},
		{
			name:    "local ahead force",
			force:   true,
			wantErr: false,
			mutate: func(path string) {
				makeBranchLocalAhead(s.T(), path, "test")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev", "test"}, []string{"dev", "test"})
			s.Require().NoError(err)

			targetPath := filepath.Join(wsDir, "infra", "test")
			if tt.mutate != nil {
				tt.mutate(targetPath)
			}

			args := []string{"worktree", "rm", "infra-test"}
			if tt.force {
				args = []string{"worktree", "rm", "--force", "infra-test"}
			}

			_, err = s.ExecuteCmd(args...)
			if tt.wantErr {
				assertSafetyRefusal(s.T(), err, tt.errPart)
				return
			}

			s.Require().NoError(err)
			cfg, loadErr := config.Load(filepath.Join(wsDir, ".gitw"))
			s.Require().NoError(loadErr)
			_, exists := cfg.Worktrees["infra"].Branches["test"]
			s.Assert().False(exists)
			s.Assert().NoDirExists(filepath.Join(wsDir, "infra", "test"))
		})
	}
}
