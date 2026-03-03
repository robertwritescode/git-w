package worktree_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/worktree"
)

type WorktreeCloneSuite struct {
	testutil.CmdSuite
}

func TestWorktreeCloneSuite(t *testing.T) {
	s := new(WorktreeCloneSuite)
	s.InitRoot(worktree.Register)
	testutil.RunSuite(t, s)
}

func (s *WorktreeCloneSuite) TestCloneAndList() {
	wsDir, _, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev", "test"}, []string{"dev", "test"})
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(wsDir, ".gitw"))
	s.Require().NoError(err)
	s.Require().Contains(cfg.Worktrees, "infra")
	s.Require().Contains(cfg.Worktrees["infra"].Branches, "dev")
	s.Require().Contains(cfg.Worktrees["infra"].Branches, "test")

	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "infra", "dev")))
	s.Assert().True(repo.IsGitRepo(filepath.Join(wsDir, "infra", "test")))

	for _, branch := range []string{"dev", "test"} {
		dir := filepath.Join(wsDir, "infra", branch)
		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
		cmd.Dir = dir
		out, cmdErr := cmd.CombinedOutput()
		s.Require().NoError(cmdErr, string(out))
		s.Assert().Equal("origin/"+branch, strings.TrimSpace(string(out)))
	}

	gitignoreData, readErr := os.ReadFile(filepath.Join(wsDir, ".gitignore"))
	s.Require().NoError(readErr)
	s.Assert().Contains(string(gitignoreData), "infra/.bare")
	s.Assert().Contains(string(gitignoreData), "infra/dev")
	s.Assert().Contains(string(gitignoreData), "infra/test")

	listOut, err := s.ExecuteCmd("worktree", "list")
	s.Require().NoError(err)
	s.Assert().Contains(listOut, "infra")

	setOut, err := s.ExecuteCmd("worktree", "ls", "infra")
	s.Require().NoError(err)
	s.Assert().Contains(setOut, "dev")
	s.Assert().Contains(setOut, "test")
}

func (s *WorktreeCloneSuite) TestCloneErrors() {
	tests := []struct {
		name    string
		prepare func() string // sets up invalid state; returns remoteURL to clone
		wantErr string
	}{
		{
			name: "bare path already exists",
			prepare: func() string {
				wsDir := s.SetupWorkspaceDir()
				s.Require().NoError(os.MkdirAll(filepath.Join(wsDir, "infra", ".bare"), 0o755))
				return s.MakeRemoteWithBranches([]string{"dev"})
			},
			wantErr: "bare path already exists",
		},
		{
			name: "set already in config",
			prepare: func() string {
				_, remoteURL, err := setupClonedWorktreeSet(s.T(), s, "infra", []string{"dev"}, []string{"dev"})
				s.Require().NoError(err)
				return remoteURL
			},
			wantErr: "already exists",
		},
		{
			name: "base path outside workspace",
			prepare: func() string {
				s.SetupWorkspaceDir()
				return s.MakeRemoteWithBranches([]string{"dev"})
			},
			wantErr: "inside workspace root",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			remoteURL := tt.prepare()
			basePath := "infra"
			if tt.name == "base path outside workspace" {
				basePath = s.T().TempDir()
			}

			_, err := s.ExecuteCmd("worktree", "clone", remoteURL, basePath, "dev")
			s.Require().Error(err)
			s.Assert().Contains(err.Error(), tt.wantErr)
		})
	}
}

func (s *WorktreeCloneSuite) TestCloneCleanupOnPartialFailure() {
	wsDir := s.SetupWorkspaceDir()

	// A remote with only the "dev" branch — "nonexistent" does not exist, so
	// the second worktree add should fail.
	remoteURL := s.MakeRemoteWithBranches([]string{"dev"})

	_, err := s.ExecuteCmd("worktree", "clone", remoteURL, "infra", "dev", "nonexistent")
	s.Require().Error(err)

	// Cleanup should have removed both the bare repo and the successful "dev" worktree.
	s.Assert().NoDirExists(filepath.Join(wsDir, "infra", ".bare"), "bare dir should be cleaned up")
	s.Assert().NoDirExists(filepath.Join(wsDir, "infra", "dev"), "successful branch dir should be cleaned up")
}
