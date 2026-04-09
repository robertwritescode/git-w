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

type BranchCreateSuite struct {
	testutil.CmdSuite
}

func TestBranchCreate(t *testing.T) {
	s := new(BranchCreateSuite)
	s.InitRoot(branchpkg.Register)
	testutil.RunSuite(t, s)
}

func (s *BranchCreateSuite) TestCreate_CommandRegistered() {
	out, err := s.ExecuteCmd("branch", "--help")
	s.Require().NoError(err)
	s.Contains(out, "create")

	aliasOut, err := s.ExecuteCmd("b", "--help")
	s.Require().NoError(err)
	s.Contains(aliasOut, "create")
}

func (s *BranchCreateSuite) TestCreate_SubcommandAliases() {
	for _, tt := range createAliasCases() {
		s.Run(tt.name, func() {
			_, err := s.ExecuteCmd(tt.args...)
			s.Require().NoError(err)
		})
	}
}

func (s *BranchCreateSuite) TestCreate_RequiresBranchName() {
	_, err := s.ExecuteCmd("branch", "create")
	s.Require().Error(err)
	s.Contains(err.Error(), "branch name")
}

func (s *BranchCreateSuite) TestCreate_CreatesLocalBranchInAllRepos() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)

	_, err := s.runBranchCreate(wsDir, "feature/auth", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.assertBranchInRepos(wsDir, names, "feature/auth")
}

func (s *BranchCreateSuite) TestCreate_OutputContainsRepoName() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(1)

	out, err := s.runBranchCreate(wsDir, "mybranch", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "["+names[0]+"] branch")
}

func (s *BranchCreateSuite) TestCreate_SummaryLine() {
	wsDir, _ := s.makeLocalWorkspaceWithDefaultBranch(2)

	out, err := s.runBranchCreate(wsDir, "feat", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "branch create complete: 2 ok, 0 failed")
}

func (s *BranchCreateSuite) TestCreate_FiltersByExplicitRepoName() {
	for _, tt := range filterCases() {
		s.Run(tt.name, func() { s.runFilterCase(tt) })
	}
}

func (s *BranchCreateSuite) TestCreate_FiltersByActiveContext() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)
	s.AppendGroup(wsDir, "web", names[0])
	s.SetActiveContext(wsDir, "web")

	_, err := s.runBranchCreate(wsDir, "feat", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.assertBranchForRepo(wsDir, names[0], "feat", true)
	s.assertBranchForRepo(wsDir, names[1], "feat", false)
}

func (s *BranchCreateSuite) TestCreate_ResolveBoolFlag() {
	for _, tt := range flagConflictCases() {
		s.Run(tt.name, func() {
			wsDir, _ := s.makeLocalWorkspace(1)

			_, err := s.runBranchCreate(wsDir, "x", tt.args...)
			s.Require().Error(err)
		})
	}
}

func (s *BranchCreateSuite) runFilterCase(tt filterCase) {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)

	targets := s.filterTargets(wsDir, names, tt)

	args := append(targets, "--no-sync-source", "--no-upstream", "--no-push")
	_, err := s.runBranchCreate(wsDir, "feat", args...)
	s.Require().NoError(err)

	s.assertBranchForRepo(wsDir, names[0], "feat", tt.expect0)
	s.assertBranchForRepo(wsDir, names[1], "feat", tt.expect1)
}

func (s *BranchCreateSuite) filterTargets(wsDir string, names []string, tt filterCase) []string {
	if tt.addGroup {
		s.AppendGroup(wsDir, "web", names[0])
		return []string{"web"}
	}

	targets := make([]string, 0, len(tt.indexes))
	for _, idx := range tt.indexes {
		targets = append(targets, names[idx])
	}

	return targets
}

func (s *BranchCreateSuite) TestCreate_SyncSource_CheckoutsAndFetches() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()

	out, err := s.runBranchCreate(wsDir, "feat", "--sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "["+names[0]+"] checkout")
	s.Contains(out, "["+names[0]+"] fetch")
	s.Contains(out, "["+names[0]+"] pull")
}

func (s *BranchCreateSuite) TestCreate_SyncSource_ConfigDefault() {
	wsDir, names, branch := s.setupRemoteWorkspaceWithDefaultBranch()
	configExtra := fmt.Sprintf("default_branch = %q\nbranch_sync_source = true\n", branch)
	s.writeWorkspaceConfig(wsDir, names, configExtra, nil)

	out, err := s.runBranchCreate(wsDir, "feat", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "["+names[0]+"] checkout")
	s.Contains(out, "["+names[0]+"] fetch")
	s.Contains(out, "["+names[0]+"] pull")
}

func (s *BranchCreateSuite) TestCreate_SyncSource_DisabledSkipsCheckout() {
	wsDir, _, _ := s.setupRemoteWorkspaceWithDefaultBranch()

	out, err := s.runBranchCreate(wsDir, "feat", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.NotContains(out, "checkout")
}

func (s *BranchCreateSuite) TestCreate_SyncSource_SkipsWhenNoRemote() {
	wsDir, names, _ := s.setupLocalWorkspaceWithDefaultBranch()

	_, err := s.runBranchCreate(wsDir, "feat", "--sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.assertBranchForRepo(wsDir, names[0], "feat", true)
}

func (s *BranchCreateSuite) TestCreate_SyncSource_SourceBranchMissing() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()
	s.writeWorkspaceConfig(wsDir, names, "default_branch = \"nonexistent\"\n", nil)

	out, err := s.runBranchCreate(wsDir, "feat", "--sync-source", "--no-upstream", "--no-push")
	s.Require().Error(err)

	s.Contains(out, "checkout error")
}

func (s *BranchCreateSuite) TestCreate_Push_PushesToOrigin() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()
	repoPath := repoPath(wsDir, names[0])
	remoteURL := s.remoteURL(repoPath)

	_, err := s.runBranchCreate(wsDir, "feat", "--push", "--no-sync-source", "--no-upstream")
	s.Require().NoError(err)

	s.Assert().True(s.remoteBranchExists(remoteURL, "feat"))
}

func (s *BranchCreateSuite) TestCreate_Push_SkipsNoRemote() {
	wsDir, _ := s.makeLocalWorkspaceWithDefaultBranch(1)

	out, err := s.runBranchCreate(wsDir, "feat", "--push", "--no-sync-source", "--no-upstream")
	s.Require().NoError(err)

	s.Contains(out, "push: no remote, skipped")
}

func (s *BranchCreateSuite) TestCreate_SetUpstream_Sets() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()
	repoDir := repoPath(wsDir, names[0])
	remoteURL := s.remoteURL(repoDir)
	s.createRemoteBranch(remoteURL, "feat")

	out, err := s.runBranchCreate(wsDir, "feat", "--sync-source", "--allow-upstream", "--no-push")
	s.Require().NoError(err, out)

	remote := s.repoUpstreamRemote(repoDir, "feat")
	s.Assert().Equal("origin", remote)
}

func (s *BranchCreateSuite) TestCreate_PushTakesPrecedenceOverSetUpstream() {
	wsDir, _, _ := s.setupRemoteWorkspaceWithDefaultBranch()

	out, err := s.runBranchCreate(wsDir, "feat", "--push", "--allow-upstream", "--no-sync-source")
	s.Require().NoError(err)

	s.Contains(out, "push")
	s.NotContains(out, "upstream")
}

func (s *BranchCreateSuite) TestCreate_SkipsExistingBranch() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)
	s.RunGit(repoPath(wsDir, names[0]), "branch", "feature")

	out, err := s.runBranchCreate(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Contains(out, "["+names[0]+"] branch: already exists, skipped")
	s.Contains(out, "["+names[1]+"] branch")
	s.Contains(out, "branch create complete: 2 ok, 0 failed")
}

func (s *BranchCreateSuite) TestCreate_PerRepoDefaultBranch() {
	wsDir, names := s.makeLocalWorkspace(1)
	repoDir := repoPath(wsDir, names[0])
	s.RunGit(repoDir, "branch", "develop")

	defaults := map[string]string{names[0]: "develop"}
	s.writeWorkspaceConfig(wsDir, names, "", defaults)

	_, err := s.runBranchCreate(wsDir, "feat", "--sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Assert().Equal("develop", s.currentBranch(repoDir))
}

func (s *BranchCreateSuite) TestCreate_WorkspaceDefaultBranch() {
	wsDir, names := s.makeLocalWorkspace(1)
	repoDir := repoPath(wsDir, names[0])
	s.RunGit(repoDir, "branch", "trunk")

	workspaceExtra := "default_branch = \"trunk\"\n"
	s.writeWorkspaceConfig(wsDir, names, workspaceExtra, nil)

	_, err := s.runBranchCreate(wsDir, "feat", "--sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Assert().Equal("trunk", s.currentBranch(repoDir))
}

func (s *BranchCreateSuite) TestCreate_DefaultFallbackIsMain() {
	wsDir, names := s.makeLocalWorkspace(1)
	repoDir := repoPath(wsDir, names[0])
	s.RunGit(repoDir, "checkout", "-b", "dev")
	s.RunGit(repoDir, "branch", "main")

	_, err := s.runBranchCreate(wsDir, "feat", "--sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Assert().Equal("main", s.currentBranch(repoDir))
}

func (s *BranchCreateSuite) TestCreate_WorktreeSet_CreatesInEachWorktree() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	configExtra := "branch_sync_source = false\nbranch_set_upstream = false\nbranch_push = false\n"
	s.setupInfraWorktreeSet(wsDir, configExtra)

	out, err := s.runBranchCreate(wsDir, "feature")
	s.Require().NoError(err, out)

	s.Assert().True(s.branchExists(filepath.Join(wsDir, "infra", "dev"), "dev-feature"))
	s.Assert().True(s.branchExists(filepath.Join(wsDir, "infra", "test"), "test-feature"))
}

func (s *BranchCreateSuite) TestCreate_WorktreeSet_SyncFetchesBareOnce() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	configExtra := "branch_set_upstream = false\nbranch_push = false\n"
	s.setupInfraWorktreeSet(wsDir, configExtra)

	out, err := s.runBranchCreate(wsDir, "feature", "--sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err, out)

	s.Equal(1, strings.Count(out, "[infra] fetch"))
	s.Contains(out, "[infra-dev] pull")
	s.Contains(out, "[infra-test] pull")
}

func (s *BranchCreateSuite) TestCreate_WorktreeSet_SourceBranchIsWorktreeBranch() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	configExtra := "default_branch = \"nope\"\nbranch_set_upstream = false\nbranch_push = false\n"
	s.setupInfraWorktreeSet(wsDir, configExtra)

	out, err := s.runBranchCreate(wsDir, "feature", "--sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err, out)
}

func (s *BranchCreateSuite) TestCreate_WorktreeSet_FetchFailureFailsEntireSet() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})
	configExtra := "branch_set_upstream = false\nbranch_push = false\n"
	s.writeInfraWorktreeConfig(wsDir, remoteURL, "infra/.missing", configExtra)

	out, err := s.runBranchCreate(wsDir, "feature", "--sync-source", "--no-upstream", "--no-push")
	s.Require().Error(err)

	s.Contains(out, "fetch error")
	s.Contains(out, "branch create complete: 0 ok, 2 failed")
}

func (s *BranchCreateSuite) TestCreate_WorktreeSet_PartialWorktreeFailure() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	configExtra := "branch_sync_source = false\nbranch_set_upstream = false\nbranch_push = false\n"
	s.setupInfraWorktreePartial(wsDir, configExtra)

	out, err := s.runBranchCreate(wsDir, "feature")
	s.Require().Error(err)

	s.Contains(out, "branch create complete: 1 ok, 1 failed")
	s.Assert().True(s.branchExists(filepath.Join(wsDir, "infra", "test"), "test-feature"))
}

func (s *BranchCreateSuite) TestCreate_PartialFailureReturnsError() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)
	s.Require().NoError(os.RemoveAll(repoPath(wsDir, names[1])))

	out, err := s.runBranchCreate(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().Error(err)

	s.Contains(out, "branch create complete: 1 ok, 1 failed")
	s.Assert().True(s.branchExists(repoPath(wsDir, names[0]), "feature"))
}

func (s *BranchCreateSuite) TestCreate_NoReposIsError() {
	wsDir := s.T().TempDir()
	writeWorkspaceFile(s, wsDir, "[metarepo]\nname = \"empty\"\n")

	_, err := s.runBranchCreate(wsDir, "feat")
	s.Require().Error(err)
}

func (s *BranchCreateSuite) TestCreate_CheckoutFlag_ChecksOutNewBranch() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)

	out, err := s.runBranchCreate(wsDir, "my-feature", "--checkout", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	for _, name := range names {
		s.Assert().Equal("my-feature", s.currentBranch(repoPath(wsDir, name)))
	}
	s.Contains(out, "checkout")
}

func (s *BranchCreateSuite) TestCreate_CheckoutFlag_AlreadyExists_ChecksOut() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)

	repo0 := repoPath(wsDir, names[0])
	s.RunGit(repo0, "checkout", "-b", "feature")
	defaultBranch := s.currentBranch(repoPath(wsDir, names[1]))
	s.RunGit(repo0, "checkout", defaultBranch)

	out, err := s.runBranchCreate(wsDir, "feature", "--checkout", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Assert().Equal("feature", s.currentBranch(repo0))
	s.Contains(out, "["+names[0]+"] branch: already exists, skipped")
	s.Contains(out, "["+names[0]+"] checkout")
}

func (s *BranchCreateSuite) TestCreate_CheckoutFlag_ShortFlag() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(1)

	out, err := s.runBranchCreate(wsDir, "feat", "-c", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Assert().Equal("feat", s.currentBranch(repoPath(wsDir, names[0])))
	s.Contains(out, "checkout")
}

func (s *BranchCreateSuite) TestCreate_CheckoutFlag_WithPush() {
	wsDir, names, _ := s.setupRemoteWorkspaceWithDefaultBranch()

	out, err := s.runBranchCreate(wsDir, "feat", "--checkout", "--push", "--no-sync-source", "--no-upstream")
	s.Require().NoError(err)

	repo0 := repoPath(wsDir, names[0])
	s.Assert().Equal("feat", s.currentBranch(repo0))

	remoteURL := s.remoteURL(repo0)
	s.Assert().True(s.remoteBranchExists(remoteURL, "feat"))
	s.Contains(out, "push")
	s.Contains(out, "checkout")
}

func (s *BranchCreateSuite) TestCreate_CheckoutFlag_WorktreeSet() {
	wsDir := s.T().TempDir()
	s.ChangeToDir(wsDir)

	configExtra := "branch_sync_source = false\nbranch_set_upstream = false\nbranch_push = false\n"
	s.setupInfraWorktreeSet(wsDir, configExtra)

	out, err := s.runBranchCreate(wsDir, "feature", "--checkout")
	s.Require().NoError(err, out)

	devWorktree := filepath.Join(wsDir, "infra", "dev")
	testWorktree := filepath.Join(wsDir, "infra", "test")

	s.Assert().Equal("dev-feature", s.currentBranch(devWorktree))
	s.Assert().Equal("test-feature", s.currentBranch(testWorktree))
	s.Contains(out, "checkout")
}

func (s *BranchCreateSuite) TestCreate_WithoutCheckoutFlag_AlreadyExists_DoesNotCheckout() {
	wsDir, names := s.makeLocalWorkspaceWithDefaultBranch(2)

	repo0 := repoPath(wsDir, names[0])
	s.RunGit(repo0, "checkout", "-b", "feature")
	defaultBranch := s.currentBranch(repoPath(wsDir, names[1]))
	s.RunGit(repo0, "checkout", defaultBranch)

	out, err := s.runBranchCreate(wsDir, "feature", "--no-sync-source", "--no-upstream", "--no-push")
	s.Require().NoError(err)

	s.Assert().Equal(defaultBranch, s.currentBranch(repo0))
	s.Contains(out, "already exists, skipped")
	s.NotContains(out, "checkout")
}

func (s *BranchCreateSuite) makeLocalWorkspace(n int) (string, []string) {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(n)
	return wsDir, names
}

func (s *BranchCreateSuite) makeLocalWorkspaceWithDefaultBranch(n int) (string, []string) {
	wsDir, names := s.makeLocalWorkspace(n)
	branch := s.currentBranch(repoPath(wsDir, names[0]))

	extra := fmt.Sprintf("default_branch = %q\n", branch)
	s.writeWorkspaceConfig(wsDir, names, extra, nil)

	return wsDir, names
}

func (s *BranchCreateSuite) makeRemoteWorkspace(n int) (string, []string) {
	wsDir, names := s.MakeWorkspaceWithNRemoteRepos(n)
	return wsDir, names
}

func (s *BranchCreateSuite) branchExists(repoPath, branchName string) bool {
	out, err := exec.Command("git", "-C", repoPath, "branch", "--list", branchName).Output()
	s.Require().NoError(err)

	return strings.TrimSpace(string(out)) != ""
}

func (s *BranchCreateSuite) remoteBranchExists(remoteURL, branchName string) bool {
	barePath := strings.TrimPrefix(remoteURL, "file://")
	cmd := exec.Command("git", "-C", barePath, "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)

	return cmd.Run() == nil
}

func (s *BranchCreateSuite) repoUpstreamRemote(repoPath, branchName string) string {
	out, err := exec.Command("git", "-C", repoPath, "config", "branch."+branchName+".remote").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func (s *BranchCreateSuite) writeWorkspaceConfig(wsDir string, names []string, workspaceExtra string, repoDefaults map[string]string) {
	builder := workspaceBuilder(workspaceExtra)

	appendRepoEntries(builder, names, repoDefaults)

	writeWorkspaceFile(s, wsDir, builder.String())
}

func workspaceBuilder(workspaceExtra string) *strings.Builder {
	builder := new(strings.Builder)
	builder.WriteString("[metarepo]\nname = \"test\"\n")
	if workspaceExtra != "" {
		builder.WriteString(workspaceExtra)
	}
	builder.WriteString("\n")

	return builder
}

func appendRepoEntries(builder *strings.Builder, names []string, repoDefaults map[string]string) {
	for _, name := range names {
		fmt.Fprintf(builder, "[[repo]]\nname = %q\npath = %q\n", name, "repos/"+name)
		if branch, ok := repoDefaults[name]; ok && branch != "" {
			fmt.Fprintf(builder, "default_branch = %q\n", branch)
		}
		builder.WriteString("\n")
	}
}

func writeWorkspaceFile(s *BranchCreateSuite, wsDir, content string) {
	cfgPath := filepath.Join(wsDir, ".gitw")
	s.Require().NoError(os.WriteFile(cfgPath, []byte(content), 0o644))
}

func (s *BranchCreateSuite) runInWorkspace(wsDir string, args ...string) (string, error) {
	s.ChangeToDir(wsDir)
	return s.ExecuteCmd(args...)
}

func (s *BranchCreateSuite) runBranchCreate(wsDir, branchName string, args ...string) (string, error) {
	cmdArgs := append([]string{"branch", "create", branchName}, args...)
	return s.runInWorkspace(wsDir, cmdArgs...)
}

func repoPath(wsDir, name string) string {
	return filepath.Join(wsDir, "repos", name)
}

func (s *BranchCreateSuite) currentBranch(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD").Output()
	s.Require().NoError(err)

	return strings.TrimSpace(string(out))
}

func (s *BranchCreateSuite) remoteURL(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func (s *BranchCreateSuite) createRemoteBranch(remoteURL, branchName string) {
	cloneDir := s.T().TempDir()
	s.RunGit("", "clone", remoteURL, cloneDir)

	s.RunGit(cloneDir, "checkout", "-b", branchName)
	s.RunGit(cloneDir, "push", "origin", branchName)
}

func (s *BranchCreateSuite) assertBranchForRepo(wsDir, name, branch string, want bool) {
	got := s.branchExists(repoPath(wsDir, name), branch)
	s.Assert().Equal(want, got)
}

func (s *BranchCreateSuite) assertBranchInRepos(wsDir string, names []string, branch string) {
	for _, name := range names {
		s.Assert().True(s.branchExists(repoPath(wsDir, name), branch))
	}
}

func (s *BranchCreateSuite) setupRemoteWorkspaceWithDefaultBranch() (string, []string, string) {
	wsDir, names := s.makeRemoteWorkspace(1)
	branch := s.currentBranch(repoPath(wsDir, names[0]))

	extra := fmt.Sprintf("default_branch = %q\n", branch)
	s.writeWorkspaceConfig(wsDir, names, extra, nil)

	return wsDir, names, branch
}

func (s *BranchCreateSuite) setupLocalWorkspaceWithDefaultBranch() (string, []string, string) {
	wsDir, names := s.makeLocalWorkspace(1)
	branch := s.currentBranch(repoPath(wsDir, names[0]))

	extra := fmt.Sprintf("default_branch = %q\n", branch)
	s.writeWorkspaceConfig(wsDir, names, extra, nil)

	return wsDir, names, branch
}

func (s *BranchCreateSuite) setupInfraWorktreeSet(wsDir, workspaceExtra string) string {
	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})
	bareAbs := s.cloneInfraBare(wsDir, remoteURL)

	s.addInfraWorktree(bareAbs, wsDir, "dev")
	s.addInfraWorktree(bareAbs, wsDir, "test")
	s.writeInfraWorktreeConfig(wsDir, remoteURL, "infra/.bare", workspaceExtra)

	return remoteURL
}

func (s *BranchCreateSuite) setupInfraWorktreePartial(wsDir, workspaceExtra string) string {
	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})
	bareAbs := s.cloneInfraBare(wsDir, remoteURL)

	s.addInfraWorktree(bareAbs, wsDir, "test")
	s.writeInfraWorktreeConfig(wsDir, remoteURL, "infra/.bare", workspaceExtra)

	return remoteURL
}

func (s *BranchCreateSuite) cloneInfraBare(wsDir, remoteURL string) string {
	bareAbs := filepath.Join(wsDir, "infra", ".bare")
	s.RunGit("", "clone", "--bare", remoteURL, bareAbs)

	return bareAbs
}

func (s *BranchCreateSuite) addInfraWorktree(bareAbs, wsDir, branch string) string {
	path := filepath.Join(wsDir, "infra", branch)
	s.RunGit("", "-C", bareAbs, "worktree", "add", path, branch)

	return path
}

func (s *BranchCreateSuite) writeInfraWorktreeConfig(wsDir, remoteURL, barePath, workspaceExtra string) {
	cfg := fmt.Sprintf("[metarepo]\nname = \"ws\"\n%s\n[worktrees.infra]\nurl=%q\nbare_path=%q\n\n[worktrees.infra.branches]\ndev=\"infra/dev\"\ntest=\"infra/test\"\n", workspaceExtra, remoteURL, barePath)
	writeWorkspaceFile(s, wsDir, cfg)
}

type createAliasCase struct {
	name string
	args []string
}

type filterCase struct {
	name     string
	indexes  []int
	addGroup bool
	expect0  bool
	expect1  bool
}

type flagConflictCase struct {
	name string
	args []string
}

func createAliasCases() []createAliasCase {
	return []createAliasCase{
		{name: "alias c", args: []string{"branch", "c", "--help"}},
		{name: "alias cut", args: []string{"branch", "cut", "--help"}},
		{name: "alias new", args: []string{"branch", "new", "--help"}},
	}
}

func filterCases() []filterCase {
	return []filterCase{
		{name: "single explicit", indexes: []int{0}, expect0: true, expect1: false},
		{name: "both explicit", indexes: []int{0, 1}, expect0: true, expect1: true},
		{name: "group", addGroup: true, expect0: true, expect1: false},
	}
}

func flagConflictCases() []flagConflictCase {
	return []flagConflictCase{
		{name: "sync-source + no-sync-source", args: []string{"--sync-source", "--no-sync-source"}},
		{name: "allow-upstream + no-upstream", args: []string{"--allow-upstream", "--no-upstream"}},
		{name: "push + no-push", args: []string{"--push", "--no-push"}},
	}
}
