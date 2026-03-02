package git_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gitpkg "github.com/robertwritescode/git-w/pkg/git"
	"github.com/robertwritescode/git-w/pkg/testutil"
)

type GitSuite struct {
	testutil.CmdSuite
}

func TestGitSuite(t *testing.T) {
	s := new(GitSuite)
	s.InitRoot(gitpkg.Register)
	testutil.RunSuite(t, s)
}

func (s *GitSuite) TestGitCmd_RunsInAllRepos() {
	tests := []struct {
		name       string
		cmdName    string
		setup      func() (wsDir string, names []string)
		checkNames bool
	}{
		{
			name:       "fetch",
			cmdName:    "fetch",
			setup:      func() (string, []string) { return s.MakeWorkspaceWithNRemoteRepos(2) },
			checkNames: false,
		},
		{
			name:       "pull",
			cmdName:    "pull",
			setup:      func() (string, []string) { return s.MakeWorkspaceWithNRemoteRepos(2) },
			checkNames: true,
		},
		{
			name:       "status",
			cmdName:    "status",
			setup:      func() (string, []string) { return s.MakeWorkspaceWithNLocalRepos(2) },
			checkNames: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, names := tt.setup()
			s.ChangeToDir(wsDir)

			out, err := s.ExecuteCmd(tt.cmdName)
			s.Require().NoError(err)

			if tt.checkNames {
				for _, name := range names {
					s.Assert().Contains(out, name)
				}
			}
		})
	}
}

func (s *GitSuite) TestPush_RequiresRemote() {
	// MakeWorkspaceWithNLocalRepos creates repos without a remote; push should fail.
	wsDir, _ := s.MakeWorkspaceWithNLocalRepos(1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("push")
	s.Require().Error(err)
}

func (s *GitSuite) TestGitCmd_ActiveContext_Scopes() {
	tests := []struct {
		name    string
		cmdName string
	}{
		{"fetch scopes to context", "fetch"},
		{"pull scopes to context", "pull"},
		{"status scopes to context", "status"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, names := s.MakeWorkspaceWithNRemoteRepos(2)

			s.AppendGroup(wsDir, "web", names[0])
			s.SetActiveContext(wsDir, "web")
			s.ChangeToDir(wsDir)

			out, err := s.ExecuteCmd(tt.cmdName)
			s.Require().NoError(err)
			s.Assert().NotContains(out, "["+names[1]+"]")
		})
	}
}

func (s *GitSuite) TestStatus_AliasWorks() {
	wsDir, _ := s.MakeWorkspaceWithNLocalRepos(2)

	s.ChangeToDir(wsDir)
	outStatus, err := s.ExecuteCmd("status")
	s.Require().NoError(err)

	s.ChangeToDir(wsDir)
	outAlias, err := s.ExecuteCmd("st")
	s.Require().NoError(err)

	s.Assert().Equal(outStatus, outAlias)
}

func (s *GitSuite) TestFetch_WorktreeSetUsesBareFetchOnce() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})
	bareAbs := filepath.Join(wsDir, "infra", ".bare")
	s.RunGit("", "clone", "--bare", remoteURL, bareAbs)
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "dev"), "dev")
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "test"), "test")

	cfg := fmt.Sprintf("[workspace]\nname=\"ws\"\n\n[worktrees.infra]\nurl=%q\nbare_path=\"infra/.bare\"\n\n[worktrees.infra.branches]\ndev=\"infra/dev\"\ntest=\"infra/test\"\n", remoteURL)
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(cfg), 0o644))

	out, err := s.ExecuteCmd("fetch", "infra")
	s.Require().NoError(err)
	s.Equal(1, strings.Count(out, "[infra] fetch"))
	s.NotContains(out, "[infra-dev]")
	s.NotContains(out, "[infra-test]")
}

func (s *GitSuite) TestFetch_AllReposDedupesWorktreeSet() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})
	bareAbs := filepath.Join(wsDir, "infra", ".bare")
	s.RunGit("", "clone", "--bare", remoteURL, bareAbs)
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "dev"), "dev")
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "test"), "test")

	regularRemote := s.T().TempDir()
	s.RunGit("", "init", "--bare", regularRemote)
	regularLocal := s.MakeGitRepoAt(wsDir, "", "ops")
	s.RunGit(regularLocal, "remote", "add", "origin", "file://"+regularRemote)
	s.PushToRemote(regularLocal)

	cfg := fmt.Sprintf("[workspace]\nname=\"ws\"\n\n[worktrees.infra]\nurl=%q\nbare_path=\"infra/.bare\"\n\n[worktrees.infra.branches]\ndev=\"infra/dev\"\ntest=\"infra/test\"\n\n[repos.ops]\npath=%q\nurl=%q\n", remoteURL, s.RelPath(wsDir, regularLocal), "file://"+regularRemote)
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(cfg), 0o644))

	out, err := s.ExecuteCmd("fetch")
	s.Require().NoError(err)
	s.Equal(1, strings.Count(out, "[infra] fetch"))
	s.NotContains(out, "[infra-dev]")
	s.NotContains(out, "[infra-test]")
}
