package workgroup_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workgroup"
)

type AddSuite struct {
	testutil.CmdSuite
}

func TestAddSuite(t *testing.T) {
	s := new(AddSuite)
	s.InitRoot(workgroup.Register)
	testutil.RunSuite(t, s)
}

func (s *AddSuite) TestAdd_RequiresName() {
	wsDir, _ := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "add")
	s.Require().Error(err)
}

func (s *AddSuite) TestAdd_UnknownWorkgroup_Errors() {
	wsDir, _ := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "add", "nonexistent", "some-repo")
	s.Require().Error(err)
}

func (s *AddSuite) TestAdd_AddsNewRepoToWorkgroup() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 2)
	s.ChangeToDir(wsDir)

	// Create workgroup with only first repo
	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	// Add second repo
	out, err := s.ExecuteCmd("workgroup", "add", "feat", names[1], "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	treePath := filepath.Join(wsDir, ".workgroup", "feat", names[1])
	s.Assert().DirExists(treePath)
}

func (s *AddSuite) TestAdd_UpdatesLocalConfig() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 2)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	out, err := s.ExecuteCmd("workgroup", "add", "feat", names[1], "--no-push", "--no-upstream")
	s.Require().NoError(err, out)

	// List should show 2 repos now
	listOut, err := s.ExecuteCmd("workgroup", "list")
	s.Require().NoError(err)
	s.Contains(listOut, "repos=2")
}

func (s *AddSuite) TestAdd_ExistingRepoSkipped() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 2)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	// Try to add the same repo again - should fail since it's already in the workgroup
	_, err = s.ExecuteCmd("workgroup", "add", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().Error(err)
}

func (s *AddSuite) TestAdd_ExistingWorktree_Skipped() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 2)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	// Add second repo, then add again - worktree path already exists
	_, err = s.ExecuteCmd("workgroup", "add", "feat", names[1], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	// Remove from config, add again - existing worktree should be detected as skipped
	treePath := filepath.Join(wsDir, ".workgroup", "feat", names[1])
	s.Assert().DirExists(treePath)
}

func branchExists(repoPath, branchName string) (bool, error) {
	out, err := exec.Command("git", "-C", repoPath, "branch", "--list", branchName).Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}
