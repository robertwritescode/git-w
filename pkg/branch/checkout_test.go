package branch_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	branchpkg "github.com/robertwritescode/git-w/pkg/branch"
	"github.com/robertwritescode/git-w/pkg/testutil"
)

type BranchCheckoutSuite struct {
	testutil.CmdSuite
}

func TestBranchCheckout(t *testing.T) {
	s := new(BranchCheckoutSuite)
	s.InitRoot(branchpkg.Register)
	testutil.RunSuite(t, s)
}

func (s *BranchCheckoutSuite) TestCheckout_CommandRegistered() {
	out, err := s.ExecuteCmd("branch", "--help")
	s.Require().NoError(err)
	s.Contains(out, "checkout")

	aliasOut, err := s.ExecuteCmd("b", "--help")
	s.Require().NoError(err)
	s.Contains(aliasOut, "checkout")
}

func (s *BranchCheckoutSuite) TestCheckout_SubcommandAliases() {
	for _, tt := range checkoutAliasCases() {
		s.Run(tt.name, func() {
			_, err := s.ExecuteCmd(tt.args...)
			s.Require().NoError(err)
		})
	}
}

func (s *BranchCheckoutSuite) TestCheckout_RequiresBranchName() {
	wsDir, _ := s.makeLocalWorkspaceWithDefaultBranch(1)

	_, err := s.runCheckout(wsDir, "")
	s.Require().Error(err)
	s.Contains(err.Error(), "branch name")
}

func (s *BranchCheckoutSuite) TestCheckout_FlagConflicts() {
	wsDir, _ := s.makeLocalWorkspaceWithDefaultBranch(1)

	for _, tt := range flagConflictCases() {
		s.Run(tt.name, func() {
			_, err := s.runCheckout(wsDir, "feature", tt.args...)
			s.Require().Error(err)
		})
	}
}

func (s *BranchCheckoutSuite) TestCheckout_LocalBranch_ChecksOut() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)
	defaultBranch := s.currentBranch(repoPath(wsDir, names[0]))

	for _, name := range names {
		repoDir := repoPath(wsDir, name)
		s.RunGit(repoDir, "checkout", "-b", "feature")
		s.RunGit(repoDir, "checkout", defaultBranch)
	}

	out, err := s.runCheckout(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	for _, name := range names {
		s.Assert().Equal("feature", s.currentBranch(repoPath(wsDir, name)))
	}
	s.Contains(out, "checkout")
}

func (s *BranchCheckoutSuite) TestCheckout_LocalBranch_AlreadyOn_Skipped() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)

	for _, name := range names {
		repoDir := repoPath(wsDir, name)
		s.RunGit(repoDir, "checkout", "-b", "feature")
	}

	out, err := s.runCheckout(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "already on feature")
	s.Contains(out, "skipped")
	s.Contains(out, "2 ok, 0 failed")
}

func (s *BranchCheckoutSuite) TestCheckout_LocalBranch_AlreadyOn_PullStillRuns() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()
	repoDir := repoPath(wsDir, names[0])
	remoteURL := s.remoteURL(repoDir)

	s.RunGit(repoDir, "checkout", "-b", "feature")
	s.RunGit(repoDir, "push", "origin", "feature")

	s.addRemoteCommit(remoteURL, "feature", "update.txt")

	out, err := s.runCheckout(wsDir, "feature", "--pull", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "pull")
	msg := gitOutput(s.T(), repoDir, "log", "-1", "--pretty=%s")
	s.Assert().Equal("update.txt", msg)
}

func (s *BranchCheckoutSuite) TestCheckout_LocalBranch_WithPull() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()
	repoDir := repoPath(wsDir, names[0])
	remoteURL := s.remoteURL(repoDir)

	s.RunGit(repoDir, "checkout", "-b", "feature")
	s.RunGit(repoDir, "push", "origin", "feature")

	defaultBranch := s.currentBranch(repoDir)
	s.RunGit(repoDir, "checkout", defaultBranch)

	s.addRemoteCommit(remoteURL, "feature", "update.txt")

	out, err := s.runCheckout(wsDir, "feature", "--pull", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "checkout")
	s.Contains(out, "pull")

	msg := gitOutput(s.T(), repoDir, "log", "-1", "--pretty=%s")
	s.Assert().Equal("update.txt", msg)
}

func (s *BranchCheckoutSuite) TestCheckout_LocalBranch_PullNoRemote_Skipped() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(1)
	repoDir := repoPath(wsDir, names[0])
	defaultBranch := s.currentBranch(repoDir)

	s.RunGit(repoDir, "checkout", "-b", "feature")
	s.RunGit(repoDir, "checkout", defaultBranch)

	out, err := s.runCheckout(wsDir, "feature", "--pull", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "pull: no remote, skipped")
}

func (s *BranchCheckoutSuite) TestCheckout_RemoteBranch_FetchesAndChecksOut() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()
	repoDir := repoPath(wsDir, names[0])
	remoteURL := s.remoteURL(repoDir)

	s.createRemoteBranchOnly(remoteURL, "remote-feature")

	out, err := s.runCheckout(wsDir, "remote-feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Assert().Equal("remote-feature", s.currentBranch(repoDir))
	s.Contains(out, "fetch")
	s.Contains(out, "checkout")
}

func (s *BranchCheckoutSuite) TestCheckout_RemoteBranch_WithPull() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()
	repoDir := repoPath(wsDir, names[0])
	remoteURL := s.remoteURL(repoDir)

	s.createRemoteBranchOnly(remoteURL, "remote-feature")

	out, err := s.runCheckout(wsDir, "remote-feature", "--pull", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "fetch")
	s.Contains(out, "checkout")
	s.Contains(out, "pull")
}

func (s *BranchCheckoutSuite) TestCheckout_CreatePath_CreatesAndChecksOut() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)

	out, err := s.runCheckout(wsDir, "new-feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	for _, name := range names {
		repoDir := repoPath(wsDir, name)
		s.Assert().True(s.branchExists(repoDir, "new-feature"))
		s.Assert().Equal("new-feature", s.currentBranch(repoDir))
	}
	s.Contains(out, "branch")
	s.Contains(out, "checkout")
}

func (s *BranchCheckoutSuite) TestCheckout_CreatePath_OutputFormat() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(1)

	out, err := s.runCheckout(wsDir, "new-feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "["+names[0]+"] branch")
	s.Contains(out, "["+names[0]+"] checkout")
}

func (s *BranchCheckoutSuite) TestCheckout_CreatePath_WithSyncSource() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()

	out, err := s.runCheckout(wsDir, "new-feature", "--sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	repoDir := repoPath(wsDir, names[0])
	s.Assert().Equal("new-feature", s.currentBranch(repoDir))
	s.Contains(out, "checkout")
	s.Contains(out, "fetch")
	s.Contains(out, "pull")
	s.Contains(out, "branch")
}

func (s *BranchCheckoutSuite) TestCheckout_CreatePath_WithPush() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()
	repoDir := repoPath(wsDir, names[0])
	remoteURL := s.remoteURL(repoDir)

	out, err := s.runCheckout(wsDir, "new-feature", "--push", "--no-sync-source", "--no-upstream")
	s.Require().NoError(err)

	s.Assert().True(s.remoteBranchExists(remoteURL, "new-feature"))
	s.Contains(out, "push")
}

func (s *BranchCheckoutSuite) TestCheckout_CreatePath_WithUpstream() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()
	repoDir := repoPath(wsDir, names[0])

	out, err := s.runCheckout(wsDir, "new-feature", "--push", "--no-sync-source", "--no-upstream")
	s.Require().NoError(err)

	// Push with -u sets tracking upstream implicitly.
	upstream := s.repoUpstreamRemote(repoDir, "new-feature")
	s.Assert().Equal("origin", upstream)
	s.Contains(out, "push")
}

func (s *BranchCheckoutSuite) TestCheckout_CreatePath_NoRemote_SkipsRemoteOps() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(1)

	out, err := s.runCheckout(wsDir, "new-feature", "--push", "--no-sync-source", "--no-upstream")
	s.Require().NoError(err)

	s.Contains(out, "push: no remote, skipped")
	s.Assert().Equal("new-feature", s.currentBranch(repoPath(wsDir, names[0])))
}

func (s *BranchCheckoutSuite) TestCheckout_FiltersByExplicitRepo() {
	for _, tt := range filterCases() {
		tt := tt
		s.Run(tt.name, func() {
			wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)

			args := []string{"feature", "--no-sync-source", "--no-upstream", "--no-push"}
			for _, idx := range tt.indexes {
				args = append(args, names[idx])
			}

			if tt.addGroup {
				s.AppendGroup(wsDir, "test-group", names[0])
				args = append(args, "test-group")
			}

			_, err := s.runCheckout(wsDir, args[0], args[1:]...)
			s.Require().NoError(err)

			s.assertBranchForRepo(wsDir, names[0], "feature", tt.expect0)
			s.assertBranchForRepo(wsDir, names[1], "feature", tt.expect1)
		})
	}
}

func (s *BranchCheckoutSuite) TestCheckout_FiltersByActiveContext() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)
	s.AppendGroup(wsDir, "test-group", names[0])
	s.SetActiveContext(wsDir, "test-group")

	_, err := s.runCheckout(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Assert().Equal("feature", s.currentBranch(repoPath(wsDir, names[0])))
	s.Assert().NotEqual("feature", s.currentBranch(repoPath(wsDir, names[1])))
}

func (s *BranchCheckoutSuite) TestCheckout_NoReposMatched_IsError() {
	wsDir, _ := s.makeLocalWorkspaceWithDefaultBranch(1)

	_, err := s.runCheckout(wsDir, "feature", "nonexistent-repo")
	s.Require().Error(err)
}

func (s *BranchCheckoutSuite) TestCheckout_SummaryLine() {
	wsDir, _ := s.makeLocalWorkspaceWithDefaultBranch(2)

	out, err := s.runCheckout(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "branch checkout complete: 2 ok, 0 failed")
}

func (s *BranchCheckoutSuite) TestCheckout_OutputContainsRepoName() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(1)

	out, err := s.runCheckout(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "["+names[0]+"] checkout")
}

func (s *BranchCheckoutSuite) TestCheckout_WorktreeSet_BranchMissing_CreatesAndChecksOut() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	configExtra := "branch_sync_source = false\nbranch_set_upstream = false\nbranch_push = false\n"
	s.setupInfraWorktreeSet(wsDir, configExtra)

	out, err := s.runCheckout(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err, out)

	devWorktree := filepath.Join(wsDir, "infra", "dev")
	testWorktree := filepath.Join(wsDir, "infra", "test")

	s.Assert().True(s.branchExists(devWorktree, "dev-feature"))
	s.Assert().True(s.branchExists(testWorktree, "test-feature"))
	s.Assert().Equal("dev-feature", s.currentBranch(devWorktree))
	s.Assert().Equal("test-feature", s.currentBranch(testWorktree))
}

func (s *BranchCheckoutSuite) TestCheckout_WorktreeSet_BranchExistsLocally_ChecksOut() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	configExtra := "branch_sync_source = false\nbranch_set_upstream = false\nbranch_push = false\n"
	s.setupInfraWorktreeSet(wsDir, configExtra)

	devWorktree := filepath.Join(wsDir, "infra", "dev")
	testWorktree := filepath.Join(wsDir, "infra", "test")

	s.RunGit(devWorktree, "checkout", "-b", "dev-feature")
	s.RunGit(testWorktree, "checkout", "-b", "test-feature")
	s.RunGit(devWorktree, "checkout", "dev")
	s.RunGit(testWorktree, "checkout", "test")

	out, err := s.runCheckout(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err, out)

	s.Assert().Equal("dev-feature", s.currentBranch(devWorktree))
	s.Assert().Equal("test-feature", s.currentBranch(testWorktree))
}

func (s *BranchCheckoutSuite) TestCheckout_WorktreeSet_SyncFetchesBareOnce() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	configExtra := "branch_set_upstream = false\nbranch_push = false\n"
	s.setupInfraWorktreeSet(wsDir, configExtra)

	out, err := s.runCheckout(wsDir, "feature", "--sync-source")
	s.Require().NoError(err, out)

	s.Assert().Equal(1, strings.Count(out, "[infra] fetch"))
}

func (s *BranchCheckoutSuite) TestCheckout_WorktreeSet_PullFetchesBareOnce() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	configExtra := "branch_sync_source = false\nbranch_set_upstream = false\nbranch_push = false\n"
	s.setupInfraWorktreeSet(wsDir, configExtra)

	devWorktree := filepath.Join(wsDir, "infra", "dev")
	testWorktree := filepath.Join(wsDir, "infra", "test")

	s.RunGit(devWorktree, "checkout", "-b", "dev-feature")
	s.RunGit(testWorktree, "checkout", "-b", "test-feature")

	out, err := s.runCheckout(wsDir, "feature", "--pull")
	s.Require().NoError(err, out)

	s.Assert().Equal(1, strings.Count(out, "[infra] fetch"))
}

func (s *BranchCheckoutSuite) TestCheckout_WorktreeSet_FetchFailure_FailsEntireSet() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	infraRemoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})
	_ = s.cloneInfraBare(wsDir, infraRemoteURL)

	barePath := filepath.Join(wsDir, "infra", ".bare")
	s.Require().NoError(os.RemoveAll(barePath))

	configExtra := "branch_set_upstream = false\nbranch_push = false\n"
	s.writeInfraWorktreeConfig(wsDir, infraRemoteURL, "infra/.bare", configExtra)

	_, err := s.runCheckout(wsDir, "feature", "--sync-source")
	s.Require().Error(err)
}

func (s *BranchCheckoutSuite) TestCheckout_DirtyWorkingTree_ReportsError() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(1)
	repoDir := repoPath(wsDir, names[0])
	defaultBranch := s.currentBranch(repoDir)

	s.RunGit(repoDir, "checkout", "-b", "feature")
	s.addCommit(repoDir, "file1.txt", "content1")
	s.RunGit(repoDir, "checkout", defaultBranch)

	filePath := filepath.Join(repoDir, "file1.txt")
	s.Require().NoError(os.WriteFile(filePath, []byte("different"), 0o644))

	out, err := s.runCheckout(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().Error(err)
	s.Contains(out, "failed")
}

func (s *BranchCheckoutSuite) TestCheckout_PartialFailure_ReturnsError() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)
	s.Require().NoError(os.RemoveAll(repoPath(wsDir, names[1])))

	out, err := s.runCheckout(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().Error(err)

	s.Contains(out, "1 ok, 1 failed")
	s.Assert().Equal("feature", s.currentBranch(repoPath(wsDir, names[0])))
}

func (s *BranchCheckoutSuite) runCheckout(wsDir, branchName string, args ...string) (string, error) {
	s.ChangeToDir(wsDir)
	cmdArgs := []string{"branch", "checkout"}
	if branchName != "" {
		cmdArgs = append(cmdArgs, branchName)
	}
	cmdArgs = append(cmdArgs, args...)
	return s.ExecuteCmd(cmdArgs...)
}

func (s *BranchCheckoutSuite) currentBranch(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD").Output()
	s.Require().NoError(err)

	return strings.TrimSpace(string(out))
}

func (s *BranchCheckoutSuite) branchExists(repoPath, branchName string) bool {
	out, err := exec.Command("git", "-C", repoPath, "branch", "--list", branchName).Output()
	s.Require().NoError(err)

	return strings.TrimSpace(string(out)) != ""
}

func (s *BranchCheckoutSuite) remoteURL(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func (s *BranchCheckoutSuite) repoUpstreamRemote(repoPath, branchName string) string {
	out, err := exec.Command("git", "-C", repoPath, "config", "--get", fmt.Sprintf("branch.%s.remote", branchName)).Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func (s *BranchCheckoutSuite) remoteBranchExists(remoteURL, branchName string) bool {
	out, err := exec.Command("git", "ls-remote", "--heads", remoteURL, branchName).Output()
	s.Require().NoError(err)

	return strings.TrimSpace(string(out)) != ""
}

func (s *BranchCheckoutSuite) makeLocalWorkspace(n int) (string, []string) {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(n)
	return wsDir, names
}

func (s *BranchCheckoutSuite) makeLocalWorkspaceWithDefaultBranch(n int) (string, []string) {
	wsDir, names := s.makeLocalWorkspace(n)
	branch := s.currentBranch(repoPath(wsDir, names[0]))

	extra := fmt.Sprintf("default_branch = %q\n", branch)
	s.writeWorkspaceConfig(wsDir, names, extra, nil)

	return wsDir, names
}

func (s *BranchCheckoutSuite) makeRemoteWorkspace(n int) (string, []string) {
	wsDir, names := s.MakeWorkspaceWithNRemoteRepos(n)
	return wsDir, names
}

func (s *BranchCheckoutSuite) setupRemoteWorkspaceWithDefaultBranch() (string, []string, string) {
	wsDir, names := s.makeRemoteWorkspace(1)
	branch := s.currentBranch(repoPath(wsDir, names[0]))

	extra := fmt.Sprintf("default_branch = %q\n", branch)
	s.writeWorkspaceConfig(wsDir, names, extra, nil)

	return wsDir, names, branch
}

func (s *BranchCheckoutSuite) writeWorkspaceConfig(wsDir string, names []string, workspaceExtra string, repoDefaults map[string]string) {
	var sb strings.Builder
	sb.WriteString("[metarepo]\n")
	sb.WriteString("name = \"test\"\n")
	sb.WriteString(workspaceExtra)
	sb.WriteString("\n")

	for _, name := range names {
		fmt.Fprintf(&sb, "[repos.%s]\npath = %q\n\n", name, name)
		if defaults, ok := repoDefaults[name]; ok {
			sb.WriteString(defaults)
		}
	}

	s.writeWorkspaceFile(wsDir, sb.String())
}

func (s *BranchCheckoutSuite) setupInfraWorktreeSet(wsDir, workspaceExtra string) string {
	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})
	bareAbs := s.cloneInfraBare(wsDir, remoteURL)

	s.addInfraWorktree(bareAbs, wsDir, "dev")
	s.addInfraWorktree(bareAbs, wsDir, "test")
	s.writeInfraWorktreeConfig(wsDir, remoteURL, "infra/.bare", workspaceExtra)

	return remoteURL
}

func (s *BranchCheckoutSuite) cloneInfraBare(wsDir, remoteURL string) string {
	bareAbs := filepath.Join(wsDir, "infra", ".bare")
	s.RunGit("", "clone", "--bare", remoteURL, bareAbs)

	return bareAbs
}

func (s *BranchCheckoutSuite) addInfraWorktree(bareAbs, wsDir, branch string) string {
	path := filepath.Join(wsDir, "infra", branch)
	s.RunGit("", "-C", bareAbs, "worktree", "add", path, branch)

	return path
}

func (s *BranchCheckoutSuite) writeInfraWorktreeConfig(wsDir, remoteURL, barePath, workspaceExtra string) {
	cfg := fmt.Sprintf("[metarepo]\nname = \"ws\"\n%s\n[worktrees.infra]\nurl=%q\nbare_path=%q\n\n[worktrees.infra.branches]\ndev=\"infra/dev\"\ntest=\"infra/test\"\n", workspaceExtra, remoteURL, barePath)
	s.writeWorkspaceFile(wsDir, cfg)
}

func (s *BranchCheckoutSuite) addRemoteCommit(remoteURL, branch, filename string) {
	cloneDir := s.T().TempDir()
	s.RunGit("", "clone", remoteURL, cloneDir)
	s.RunGit(cloneDir, "checkout", branch)
	s.addCommit(cloneDir, filename, filename)
	s.RunGit(cloneDir, "push", "origin", branch)
}

func (s *BranchCheckoutSuite) createRemoteBranchOnly(remoteURL, branchName string) {
	cloneDir := s.T().TempDir()
	s.RunGit("", "clone", remoteURL, cloneDir)
	s.RunGit(cloneDir, "checkout", "-b", branchName)
	s.RunGit(cloneDir, "push", "origin", branchName)
}

func (s *BranchCheckoutSuite) addCommit(repoPath, filename, content string) {
	filePath := filepath.Join(repoPath, filename)
	s.Require().NoError(os.WriteFile(filePath, []byte(content), 0o644))
	s.RunGit(repoPath, "add", filename)
	s.RunGit(repoPath, "commit", "-m", content)
}

func (s *BranchCheckoutSuite) assertBranchForRepo(wsDir, name, branch string, want bool) {
	got := s.branchExists(repoPath(wsDir, name), branch)
	s.Assert().Equal(want, got)
}

func (s *BranchCheckoutSuite) writeWorkspaceFile(wsDir, content string) {
	cfgPath := filepath.Join(wsDir, ".gitw")
	s.Require().NoError(os.WriteFile(cfgPath, []byte(content), 0o644))
}

type checkoutAliasCase struct {
	name string
	args []string
}

func checkoutAliasCases() []checkoutAliasCase {
	return []checkoutAliasCase{
		{name: "co", args: []string{"branch", "co", "--help"}},
		{name: "switch", args: []string{"branch", "switch", "--help"}},
		{name: "b co", args: []string{"b", "co", "--help"}},
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}

	return strings.TrimSpace(string(output))
}
