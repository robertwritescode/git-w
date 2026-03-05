package workgroup_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workgroup"
)

type CheckoutSuite struct {
	testutil.CmdSuite
}

func TestCheckoutSuite(t *testing.T) {
	s := new(CheckoutSuite)
	s.InitRoot(workgroup.Register)
	testutil.RunSuite(t, s)
}

func (s *CheckoutSuite) TestCheckout_Aliases() {
	for _, tt := range checkoutAliasCases() {
		s.Run(tt.name, func() {
			_, err := s.ExecuteCmd(tt.args...)
			s.Require().NoError(err)
		})
	}
}

func (s *CheckoutSuite) TestCheckout_RequiresName() {
	wsDir, _ := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "checkout")
	s.Require().Error(err)
}

func (s *CheckoutSuite) TestCheckout_LocalBranch_CreatesWorktree() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	repoDir := filepath.Join(wsDir, names[0])
	s.RunGit(repoDir, "checkout", "-b", "feat")
	s.RunGit(repoDir, "checkout", "-")

	out, err := s.ExecuteCmd("workgroup", "checkout", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	treePath := filepath.Join(wsDir, ".workgroup", "feat", names[0])
	s.Assert().DirExists(treePath)

	cur := currentBranchAt(s.T(), treePath)
	s.Assert().Equal("feat", cur)
}

func (s *CheckoutSuite) TestCheckout_RemoteBranch_FetchesAndCreatesWorktree() {
	wsDir, names, remoteURL := s.setupRemoteWorkspace(1)
	s.ChangeToDir(wsDir)

	s.createRemoteBranchOnly(remoteURL, "feat")

	out, err := s.ExecuteCmd("workgroup", "checkout", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	treePath := filepath.Join(wsDir, ".workgroup", "feat", names[0])
	s.Assert().DirExists(treePath)
	s.Contains(out, "fetch")
}

func (s *CheckoutSuite) TestCheckout_MissingBranch_CreatesFromSource() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("workgroup", "checkout", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	treePath := filepath.Join(wsDir, ".workgroup", "feat", names[0])
	s.Assert().DirExists(treePath)
}

func (s *CheckoutSuite) TestCheckout_ExistingWorkgroup_UsesStoredRepos() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	// Create workgroup first
	out, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	// Remove worktree manually to simulate a fresh machine scenario
	treePath := filepath.Join(wsDir, ".workgroup", "feat", names[0])
	s.Require().NoError(exec.Command("git", "-C", filepath.Join(wsDir, names[0]), "worktree", "remove", treePath).Run())

	// Checkout with no explicit repos - should use stored list
	out, err = s.ExecuteCmd("workgroup", "checkout", "feat", "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	s.Assert().DirExists(treePath)
}

func (s *CheckoutSuite) TestCheckout_WritesLocalConfig() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("workgroup", "checkout", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	s.Assert().FileExists(filepath.Join(wsDir, ".gitw.local"))
}

func (s *CheckoutSuite) TestCheckout_ExistingWorktree_Skipped() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("workgroup", "checkout", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	out, err = s.ExecuteCmd("workgroup", "checkout", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	s.Contains(out, "already exists, skipped")
}

func (s *CheckoutSuite) TestCheckout_Summary() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 2)
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("workgroup", "checkout", "feat", names[0], names[1], "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	s.Contains(out, "work checkout complete: 2 ok, 0 failed")
}

func (s *CheckoutSuite) setupRemoteWorkspace(n int) (string, []string, string) {
	wsDir, names := makeWorkspaceWithRemoteRepos(&s.CmdSuite, n)
	remoteURL := remoteURLAt(s.T(), filepath.Join(wsDir, names[0]))
	return wsDir, names, remoteURL
}

func (s *CheckoutSuite) createRemoteBranchOnly(remoteURL, branchName string) {
	cloneDir := s.T().TempDir()
	s.RunGit("", "clone", remoteURL, cloneDir)
	s.RunGit(cloneDir, "checkout", "-b", branchName)
	s.RunGit(cloneDir, "push", "origin", branchName)
}

type checkoutAliasCase struct {
	name string
	args []string
}

func checkoutAliasCases() []checkoutAliasCase {
	return []checkoutAliasCase{
		{name: "co", args: []string{"workgroup", "co", "--help"}},
		{name: "switch", args: []string{"workgroup", "switch", "--help"}},
		{name: "wg co", args: []string{"wg", "co", "--help"}},
	}
}

func currentBranchAt(t *testing.T, repoPath string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse in %s: %v", repoPath, err)
	}
	return strings.TrimSpace(string(out))
}

func remoteURLAt(t *testing.T, repoPath string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
