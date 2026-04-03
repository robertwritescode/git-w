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

type SyncSuite struct {
	testutil.CmdSuite
}

type syncPushCase struct {
	name     string
	syncPush *bool
	args     []string
	wantPush bool
}

var syncPushCases = func() []syncPushCase {
	trueValue := true
	falseValue := false
	return []syncPushCase{
		{name: "default enabled", syncPush: nil, args: []string{"sync"}, wantPush: true},
		{name: "config disabled", syncPush: &falseValue, args: []string{"sync"}, wantPush: false},
		{name: "push overrides config", syncPush: &falseValue, args: []string{"sync", "--push"}, wantPush: true},
		{name: "no-push overrides config", syncPush: &trueValue, args: []string{"sync", "--no-push"}, wantPush: false},
	}
}()

func TestSyncSuite(t *testing.T) {
	s := new(SyncSuite)
	s.InitRoot(gitpkg.Register)
	testutil.RunSuite(t, s)
}

func (s *SyncSuite) TestSync_AliasWorks() {
	wsDir, names := s.MakeWorkspaceWithNRemoteRepos(1)
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("sync")
	s.Require().NoError(err)
	s.Contains(out, "["+names[0]+"] fetch")

	s.ChangeToDir(wsDir)
	outAlias, err := s.ExecuteCmd("s")
	s.Require().NoError(err)
	s.Contains(outAlias, "["+names[0]+"] fetch")
}

func (s *SyncSuite) TestSync_AllOk() {
	wsDir, names := s.MakeWorkspaceWithNRemoteRepos(2)
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("sync")
	s.Require().NoError(err)

	for _, name := range names {
		s.Contains(out, "["+name+"] fetch")
		s.Contains(out, "["+name+"] pull")
		s.Contains(out, "["+name+"] push")
	}

	s.Contains(out, "sync complete: 2 ok, 0 failed")
}

func (s *SyncSuite) TestSync_FetchFailureSkipsPullPush() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)
	repoGood := filepath.Join(wsDir, names[0])
	repoBad := filepath.Join(wsDir, names[1])
	s.makeRepoRemoteBacked(repoGood)
	s.RunGit(repoBad, "remote", "add", "origin", "file:///definitely/missing/remote.git")
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("sync")
	s.Require().Error(err)

	s.Contains(out, "["+names[0]+"] fetch")
	s.Contains(out, "["+names[0]+"] pull")
	s.Contains(out, "["+names[0]+"] push")
	s.Contains(out, "["+names[1]+"] fetch error:")
	s.NotContains(out, "["+names[1]+"] pull")
	s.NotContains(out, "["+names[1]+"] push")
	s.Contains(out, "sync complete: 1 ok, 1 failed")
}

func (s *SyncSuite) TestSync_PullFailureSkipsPush() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(1)
	repoPath := filepath.Join(wsDir, names[0])
	remoteURL := s.makeRepoRemoteBacked(repoPath)
	s.makeRemoteCommit(remoteURL)
	s.Require().NoError(os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("dirty\n"), 0o644))
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("sync")
	s.Require().Error(err)
	s.Contains(out, "["+names[0]+"] fetch")
	s.Contains(out, "["+names[0]+"] pull error:")
	s.NotContains(out, "["+names[0]+"] push")
	s.Contains(out, "sync complete: 0 ok, 1 failed")
}

func (s *SyncSuite) TestSync_PushFlagPrecedence() {
	for _, tt := range syncPushCases {
		s.Run(tt.name, func() {
			wsDir, repoName := s.makeWorkspaceWithRemoteRepoAndSyncPush(tt.syncPush)
			s.ChangeToDir(wsDir)

			out, err := s.ExecuteCmd(tt.args...)
			s.Require().NoError(err)
			hasPush := strings.Contains(out, "["+repoName+"] push")
			s.Equal(tt.wantPush, hasPush)
		})
	}
}

func (s *SyncSuite) TestSync_NoPushAndPushFlagConflict() {
	wsDir, _ := s.MakeWorkspaceWithNRemoteRepos(1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("sync", "--push", "--no-push")
	s.Require().Error(err)
	s.Contains(err.Error(), "cannot be used together")
}

func (s *SyncSuite) TestSync_FilteringWithActiveContext() {
	wsDir, names := s.MakeWorkspaceWithNRemoteRepos(2)
	s.AppendGroup(wsDir, "web", names[0])
	s.SetActiveContext(wsDir, "web")
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("sync")
	s.Require().NoError(err)
	s.Contains(out, "["+names[0]+"] fetch")
	s.NotContains(out, "["+names[1]+"] fetch")
}

func (s *SyncSuite) TestSync_WorktreeSetFetchOnce() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	s.setupInfraWorktreeSet(wsDir)

	out, err := s.ExecuteCmd("sync", "--no-push")
	s.Require().NoError(err)
	s.Equal(1, strings.Count(out, "[infra] fetch"))
	s.Contains(out, "[infra-dev] pull")
	s.Contains(out, "[infra-test] pull")
	s.NotContains(out, "[infra-dev] push")
	s.NotContains(out, "[infra-test] push")
}

func (s *SyncSuite) TestSync_WorktreeSetBranchFailureDoesNotBlockSibling() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	// Register "dev" path in config but don't create its worktree directory,
	// so git commands for infra-dev fail while infra-test (properly set up) succeeds.
	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})
	bareAbs := filepath.Join(wsDir, "infra", ".bare")
	s.RunGit("", "clone", "--bare", remoteURL, bareAbs)
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "test"), "test")

	cfg := fmt.Sprintf("[metarepo]\nname=\"ws\"\n\n[worktrees.infra]\nurl=%q\nbare_path=\"infra/.bare\"\n\n[worktrees.infra.branches]\ndev=\"infra/dev\"\ntest=\"infra/test\"\n", remoteURL)
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(cfg), 0o644))

	out, err := s.ExecuteCmd("sync", "--no-push")
	s.Require().Error(err)
	s.Contains(out, "[infra-dev] pull error:")
	s.Contains(out, "[infra-test] pull")
	s.Contains(out, "sync complete: 1 ok, 1 failed")
}

func (s *SyncSuite) TestSync_FilteringWithExplicitRepoName() {
	wsDir, names := s.MakeWorkspaceWithNRemoteRepos(2)
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("sync", names[0])
	s.Require().NoError(err)
	s.Contains(out, "["+names[0]+"] fetch")
	s.NotContains(out, "["+names[1]+"] fetch")
	s.Contains(out, "sync complete: 1 ok, 0 failed")
}

func (s *SyncSuite) TestSync_FilteringWithGroupName() {
	wsDir, names := s.MakeWorkspaceWithNRemoteRepos(2)
	s.AppendGroup(wsDir, "web", names[0])
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("sync", "web")
	s.Require().NoError(err)
	s.Contains(out, "["+names[0]+"] fetch")
	s.NotContains(out, "["+names[1]+"] fetch")
	s.Contains(out, "sync complete: 1 ok, 0 failed")
}

func (s *SyncSuite) setupInfraWorktreeSet(wsDir string) string {
	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})
	bareAbs := filepath.Join(wsDir, "infra", ".bare")
	s.RunGit("", "clone", "--bare", remoteURL, bareAbs)
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "dev"), "dev")
	s.RunGit("", "-C", bareAbs, "worktree", "add", filepath.Join(wsDir, "infra", "test"), "test")

	cfg := fmt.Sprintf("[metarepo]\nname=\"ws\"\n\n[worktrees.infra]\nurl=%q\nbare_path=\"infra/.bare\"\n\n[worktrees.infra.branches]\ndev=\"infra/dev\"\ntest=\"infra/test\"\n", remoteURL)
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(cfg), 0o644))
	return remoteURL
}

func (s *SyncSuite) makeRepoRemoteBacked(repoPath string) string {
	_, remoteURL := s.CreateBareRepo()
	s.RunGit(repoPath, "remote", "add", "origin", remoteURL)
	s.PushToRemote(repoPath)
	return remoteURL
}

func (s *SyncSuite) makeRemoteCommit(remoteURL string) {
	clonePath := s.T().TempDir()
	s.RunGit("", "clone", remoteURL, clonePath)
	s.Require().NoError(os.WriteFile(filepath.Join(clonePath, "README.md"), []byte("remote\n"), 0o644))
	s.RunGit(clonePath, "add", "README.md")
	s.RunGit(clonePath, "commit", "-m", "remote change")
	s.RunGit(clonePath, "push", "origin", "HEAD")
}

func (s *SyncSuite) makeWorkspaceWithRemoteRepoAndSyncPush(syncPush *bool) (string, string) {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(1)
	repoPath := filepath.Join(wsDir, names[0])
	s.makeRepoRemoteBacked(repoPath)

	syncLine := ""
	if syncPush != nil {
		syncLine = fmt.Sprintf("sync_push = %t\n", *syncPush)
	}

	cfg := fmt.Sprintf("[metarepo]\nname = \"test\"\n%s\n[[repo]]\nname = %q\npath = %q\n", syncLine, names[0], names[0])
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(cfg), 0o644))
	return wsDir, names[0]
}
