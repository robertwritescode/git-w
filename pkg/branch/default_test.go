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

type BranchDefaultSuite struct {
	testutil.CmdSuite
}

func TestBranchDefault(t *testing.T) {
	s := new(BranchDefaultSuite)
	s.InitRoot(branchpkg.Register)
	testutil.RunSuite(t, s)
}

func (s *BranchDefaultSuite) TestDefault_CommandRegistered() {
	out, err := s.ExecuteCmd("branch", "--help")
	s.Require().NoError(err)
	s.Contains(out, "default")

	out, err = s.ExecuteCmd("b", "--help")
	s.Require().NoError(err)
	s.Contains(out, "default")
}

func (s *BranchDefaultSuite) TestDefault_SubcommandAlias() {
	for _, tt := range defaultAliasCases() {
		s.Run(tt.name, func() {
			_, err := s.ExecuteCmd(tt.args...)
			s.Require().NoError(err)
		})
	}
}

// --- happy path ---

func (s *BranchDefaultSuite) TestDefault_SwitchesToDefaultBranch() {
	wsDir, paths, _ := s.makeLocalWorkspace(2, "feature", "main")

	_, err := s.runDefault(wsDir)
	s.Require().NoError(err)

	s.Equal("main", s.currentBranch(paths[0]))
	s.Equal("main", s.currentBranch(paths[1]))
}

func (s *BranchDefaultSuite) TestDefault_OutputContainsRepoName() {
	wsDir, _, names := s.makeLocalWorkspace(1, "feature", "main")

	out, err := s.runDefault(wsDir)
	s.Require().NoError(err)
	s.Contains(out, "["+names[0]+"] checkout")
}

func (s *BranchDefaultSuite) TestDefault_SummaryLine() {
	wsDir, _, _ := s.makeLocalWorkspace(2, "feature", "main")

	out, err := s.runDefault(wsDir)
	s.Require().NoError(err)
	s.Contains(out, "branch default complete: 2 ok, 0 failed")
}

func (s *BranchDefaultSuite) TestDefault_AlreadyOnDefaultBranch_IsSkipped() {
	wsDir, _, _ := s.makeLocalWorkspace(1, "main", "")

	out, err := s.runDefault(wsDir)
	s.Require().NoError(err)
	s.Contains(out, "skipped")
	s.Contains(out, "1 ok, 0 failed")
}

func (s *BranchDefaultSuite) TestDefault_AlreadyOnBranch_PullStillRuns() {
	wsDir := s.SetupWorkspaceDir()
	dir, name, _ := s.makeRepoWithRemoteAhead(wsDir, "main")
	s.writeDefaultConfig(wsDir, []string{name}, nil, "")

	out, err := s.runDefault(wsDir, "--pull")
	s.Require().NoError(err)
	s.Contains(out, "pull")
	s.Equal("extra commit", s.gitOutput(dir, "log", "-1", "--pretty=%s"))
}

// --- source branch resolution ---

func (s *BranchDefaultSuite) TestDefault_SourceBranchResolution() {
	for _, tt := range sourceBranchResolutionCases() {
		s.Run(tt.name, func() {
			s.assertSourceBranchResolution(tt)
		})
	}
}

func (s *BranchDefaultSuite) assertSourceBranchResolution(tt sourceBranchCase) {
	wsDir, paths, names := s.makeLocalWorkspace(1, tt.startBranch, tt.configBranch)
	if tt.repoOverride != "" && tt.repoOverride != tt.startBranch {
		s.RunGit(paths[0], "checkout", "-b", tt.repoOverride)
		s.RunGit(paths[0], "checkout", tt.startBranch)
	}
	if tt.repoOverride != "" {
		s.writeDefaultConfig(wsDir, names, map[string]string{names[0]: tt.repoOverride}, "")
	}

	_, err := s.runDefault(wsDir)
	s.Require().NoError(err)
	s.Equal(tt.wantBranch, s.currentBranch(paths[0]), "repo should be on expected branch")
}

// --- filtering ---

func (s *BranchDefaultSuite) TestDefault_FiltersByExplicitRepoName() {
	for _, tt := range defaultFilterCases() {
		s.Run(tt.name, func() {
			wsDir, paths, names := s.makeLocalWorkspace(2, "feature", "main")
			s.writeDefaultConfig(wsDir, names, nil, "")

			_, err := s.runDefault(wsDir, tt.args...)
			s.Require().NoError(err)

			for i, shouldSwitch := range tt.expected {
				want := "main"
				if !shouldSwitch {
					want = "feature"
				}
				s.Equal(want, s.currentBranch(paths[i]), "repo %s mismatch", names[i])
			}
		})
	}
}

func (s *BranchDefaultSuite) TestDefault_FiltersByGroup() {
	wsDir, paths, names := s.makeLocalWorkspace(2, "feature", "main")
	s.AppendGroup(wsDir, "team", names[1])

	_, err := s.runDefault(wsDir, "team")
	s.Require().NoError(err)
	s.Equal("feature", s.currentBranch(paths[0]))
	s.Equal("main", s.currentBranch(paths[1]))
}

// --- pull flag ---

func (s *BranchDefaultSuite) TestDefault_Pull_PullsAfterCheckout() {
	wsDir := s.SetupWorkspaceDir()
	dir, name, _ := s.makeRepoWithRemoteAhead(wsDir, "main")
	s.RunGit(dir, "checkout", "-b", "feature")
	s.writeDefaultConfig(wsDir, []string{name}, nil, "")

	out, err := s.runDefault(wsDir, "--pull")
	s.Require().NoError(err)
	s.Contains(out, "pull")
	s.Equal("extra commit", s.gitOutput(dir, "log", "-1", "--pretty=%s"))
}

func (s *BranchDefaultSuite) TestDefault_Pull_NoRemote_SkipsPull() {
	wsDir, _, _ := s.makeLocalWorkspace(1, "feature", "main")

	out, err := s.runDefault(wsDir, "--pull")
	s.Require().NoError(err)
	s.Contains(out, "no remote")
}

func (s *BranchDefaultSuite) TestDefault_Pull_FailureIsNonFatal() {
	wsDir := s.SetupWorkspaceDir()
	dir, name, remoteURL := s.makeRepoWithRemoteAhead(wsDir, "main")
	s.writeDefaultConfig(wsDir, []string{name}, nil, "")

	// make pull fail by removing the remote bare repo
	barePath := strings.TrimPrefix(remoteURL, "file://")
	s.Require().NoError(os.RemoveAll(barePath))

	out, err := s.runDefault(wsDir, "--pull")
	s.Require().NoError(err) // pull failure is non-fatal
	s.Contains(out, "1 ok, 0 failed")
	s.Equal("main", s.currentBranch(dir))
}

func (s *BranchDefaultSuite) TestDefault_Pull_DefaultFalse() {
	wsDir := s.SetupWorkspaceDir()
	dir, name, _ := s.makeRepoWithRemoteAhead(wsDir, "feature")
	s.writeDefaultConfig(wsDir, []string{name}, nil, "main")

	_, err := s.runDefault(wsDir)
	s.Require().NoError(err)
	s.NotEqual("extra commit", s.gitOutput(dir, "log", "-1", "--pretty=%s"))
}

// --- worktree sets ---

func (s *BranchDefaultSuite) TestDefault_WorktreeSet_SwitchesEachWorktree() {
	wsDir := s.SetupWorkspaceDir()
	remoteURL, devDir, testDir := s.setupInfraWorktreeSet(wsDir)
	s.writeInfraWorktreeConfig(wsDir, remoteURL)

	// put both worktrees on a non-default branch
	s.RunGit(devDir, "checkout", "-b", "feature")
	s.RunGit(testDir, "checkout", "-b", "feature-test")

	_, err := s.runDefault(wsDir)
	s.Require().NoError(err)
	s.Equal("dev", s.currentBranch(devDir))
	s.Equal("test", s.currentBranch(testDir))
}

func (s *BranchDefaultSuite) TestDefault_WorktreeSet_AlreadyOnAssignedBranch() {
	wsDir := s.SetupWorkspaceDir()
	remoteURL, _, _ := s.setupInfraWorktreeSet(wsDir)
	s.writeInfraWorktreeConfig(wsDir, remoteURL)

	out, err := s.runDefault(wsDir)
	s.Require().NoError(err)
	s.Contains(out, "skipped")
	s.Contains(out, "2 ok, 0 failed")
}

func (s *BranchDefaultSuite) TestDefault_WorktreeSet_PullFetchesBareOnce() {
	wsDir := s.SetupWorkspaceDir()
	remoteURL, devDir, _ := s.setupInfraWorktreeSet(wsDir)
	s.writeInfraWorktreeConfig(wsDir, remoteURL)
	s.addRemoteCommit(remoteURL, "dev", "new.txt")

	out, err := s.runDefault(wsDir, "--pull")
	s.Require().NoError(err)
	s.Equal(1, strings.Count(out, "[infra] fetch"))
	s.Equal("extra commit", s.gitOutput(devDir, "log", "-1", "--pretty=%s"))
}

func (s *BranchDefaultSuite) TestDefault_WorktreeSet_FetchFailureFailsEntireSet() {
	wsDir := s.SetupWorkspaceDir()
	remoteURL, _, _ := s.setupInfraWorktreeSet(wsDir)
	// use a missing bare path so fetch fails
	s.writeInfraWorktreeConfigAt(wsDir, remoteURL, "infra/.missing")

	out, err := s.runDefault(wsDir, "--pull")
	s.Require().Error(err)
	s.Contains(out, "0 ok, 2 failed")
}

func (s *BranchDefaultSuite) TestDefault_WorktreeSet_SourceBranchIsAssignedBranch() {
	wsDir := s.SetupWorkspaceDir()
	remoteURL, devDir, _ := s.setupInfraWorktreeSet(wsDir)
	s.writeInfraWorktreeConfig(wsDir, remoteURL)
	s.RunGit(devDir, "checkout", "-b", "feature")

	_, err := s.runDefault(wsDir)
	s.Require().NoError(err)
	s.Equal("dev", s.currentBranch(devDir))
}

// --- error handling ---

func (s *BranchDefaultSuite) TestDefault_CheckoutFailureReturnsError() {
	wsDir, paths, names := s.makeLocalWorkspace(1, "main", "")
	s.makeDirtyCheckout(paths[0], "feature")
	s.writeDefaultConfig(wsDir, names, map[string]string{names[0]: "feature"}, "")

	_, err := s.runDefault(wsDir)
	s.Require().Error(err)
}

func (s *BranchDefaultSuite) TestDefault_PartialFailureReturnsError() {
	wsDir, paths, names := s.makeLocalWorkspace(2, "main", "")
	s.makeDirtyCheckout(paths[0], "feature")
	s.writeDefaultConfig(wsDir, names, map[string]string{names[0]: "feature"}, "")

	out, err := s.runDefault(wsDir)
	s.Require().Error(err)
	s.Contains(out, "1 ok, 1 failed")
}

func (s *BranchDefaultSuite) TestDefault_NoReposMatched() {
	wsDir, _, _ := s.makeLocalWorkspace(1, "main", "")

	_, err := s.runDefault(wsDir, "nonexistent-repo")
	s.Require().Error(err)
}

// --- helpers ---

func (s *BranchDefaultSuite) makeLocalWorkspace(n int, startBranch, configBranch string) (string, []string, []string) {
	wsDir := s.SetupWorkspaceDir()
	var paths, names []string
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("repo%d", i)
		dir := s.MakeGitRepoAt(wsDir, "", name)
		s.RunGit(dir, "branch", "-M", "main")
		if startBranch != "main" {
			s.RunGit(dir, "checkout", "-b", startBranch)
		}
		paths = append(paths, dir)
		names = append(names, name)
	}
	s.writeDefaultConfig(wsDir, names, nil, configBranch)
	return wsDir, paths, names
}

func (s *BranchDefaultSuite) makeRepoWithRemoteAhead(wsDir, startBranch string) (string, string, string) {
	_, remoteURL := s.CreateBareRepo()
	name := "remoterepo"
	dir := s.MakeGitRepoAt(wsDir, "", name)
	s.RunGit(dir, "branch", "-M", "main")
	s.RunGit(dir, "remote", "add", "origin", remoteURL)
	if startBranch != "main" {
		s.RunGit(dir, "checkout", "-b", startBranch)
		s.RunGit(dir, "push", "-u", "origin", startBranch)
	} else {
		s.RunGit(dir, "push", "-u", "origin", "main")
	}
	s.addRemoteCommit(remoteURL, startBranch, "extra.txt")
	return dir, name, remoteURL
}

func (s *BranchDefaultSuite) writeDefaultConfig(wsDir string, names []string, overrides map[string]string, defaultBranch string) {
	var sb strings.Builder
	sb.WriteString("[metarepo]\n")
	if defaultBranch != "" {
		fmt.Fprintf(&sb, "default_branch = %q\n", defaultBranch)
	}
	for _, n := range names {
		fmt.Fprintf(&sb, "\n[[repo]]\nname = %q\npath = %q\n", n, n)
		if b, ok := overrides[n]; ok {
			fmt.Fprintf(&sb, "default_branch = %q\n", b)
		}
	}
	cfg := filepath.Join(wsDir, ".gitw")
	s.Require().NoError(os.WriteFile(cfg, []byte(sb.String()), 0644))
}

func (s *BranchDefaultSuite) setupInfraWorktreeSet(wsDir string) (string, string, string) {
	remoteURL := s.MakeRemoteWithBranches([]string{"dev", "test"})
	bareDir := filepath.Join(wsDir, "infra", ".bare")
	s.Require().NoError(os.MkdirAll(filepath.Dir(bareDir), 0755))
	s.RunGit(wsDir, "clone", "--bare", remoteURL, bareDir)
	devDir := filepath.Join(wsDir, "infra", "dev")
	testDir := filepath.Join(wsDir, "infra", "test")
	s.RunGit(bareDir, "worktree", "add", devDir, "dev")
	s.RunGit(bareDir, "worktree", "add", testDir, "test")
	return remoteURL, devDir, testDir
}

func (s *BranchDefaultSuite) writeInfraWorktreeConfig(wsDir, remoteURL string) {
	s.writeInfraWorktreeConfigAt(wsDir, remoteURL, "infra/.bare")
}

func (s *BranchDefaultSuite) writeInfraWorktreeConfigAt(wsDir, remoteURL, barePath string) {
	cfg := fmt.Sprintf("[metarepo]\ndefault_branch = \"main\"\n\n[worktrees.infra]\nurl = %q\nbare_path = %q\n\n[worktrees.infra.branches]\ndev = \"infra/dev\"\ntest = \"infra/test\"\n", remoteURL, barePath)
	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitw"), []byte(cfg), 0644))
}

func (s *BranchDefaultSuite) makeDirtyCheckout(dir, branch string) {
	s.RunGit(dir, "checkout", "-b", branch)
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "README.md"), []byte("branch version"), 0644))
	s.RunGit(dir, "add", "README.md")
	s.RunGit(dir, "commit", "-m", "branch change")
	s.RunGit(dir, "checkout", "main")
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "README.md"), []byte("dirty"), 0644))
}

func (s *BranchDefaultSuite) addRemoteCommit(remoteURL, branch, filename string) {
	tmp := s.T().TempDir()
	s.RunGit(tmp, "clone", remoteURL, tmp+"/clone")
	clone := tmp + "/clone"
	s.RunGit(clone, "checkout", branch)
	s.Require().NoError(os.WriteFile(filepath.Join(clone, filename), []byte("content"), 0644))
	s.RunGit(clone, "add", ".")
	s.RunGit(clone, "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "extra commit")
	s.RunGit(clone, "push", "origin", branch)
}

func (s *BranchDefaultSuite) runDefault(wsDir string, extra ...string) (string, error) {
	s.ChangeToDir(wsDir)
	args := append([]string{"branch", "default"}, extra...)
	return s.ExecuteCmd(args...)
}

func (s *BranchDefaultSuite) currentBranch(dir string) string {
	return strings.TrimSpace(s.gitOutput(dir, "rev-parse", "--abbrev-ref", "HEAD"))
}

func (s *BranchDefaultSuite) gitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	s.Require().NoError(err)
	return strings.TrimSpace(string(out))
}

// --- case data ---

type defaultAliasCase struct {
	name string
	args []string
}

type sourceBranchCase struct {
	name         string
	startBranch  string
	configBranch string
	repoOverride string
	wantBranch   string
}

type defaultFilterCase struct {
	name     string
	args     []string
	expected []bool // per-repo; true = switched to default
}

func defaultAliasCases() []defaultAliasCase {
	return []defaultAliasCase{
		{name: "full name", args: []string{"branch", "default", "--help"}},
		{name: "branch alias", args: []string{"b", "default", "--help"}},
		{name: "default alias", args: []string{"branch", "d", "--help"}},
		{name: "both aliases", args: []string{"b", "d", "--help"}},
	}
}

func sourceBranchResolutionCases() []sourceBranchCase {
	return []sourceBranchCase{
		{
			name:         "workspace default_branch",
			startBranch:  "feature",
			configBranch: "main",
			wantBranch:   "main",
		},
		{
			name:         "repo-level override wins",
			startBranch:  "feature",
			configBranch: "main",
			repoOverride: "develop",
			wantBranch:   "develop",
		},
		{
			name:         "falls back to main when no config",
			startBranch:  "feature",
			configBranch: "",
			wantBranch:   "main",
		},
	}
}

func defaultFilterCases() []defaultFilterCase {
	return []defaultFilterCase{
		{
			name:     "explicit repo name switches only that repo",
			args:     []string{"repo0"},
			expected: []bool{true, false},
		},
		{
			name:     "both repos by name switches both",
			args:     []string{"repo0", "repo1"},
			expected: []bool{true, true},
		},
	}
}
