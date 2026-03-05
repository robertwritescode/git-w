package workgroup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workgroup"
)

type DropSuite struct {
	testutil.CmdSuite
}

func TestDropSuite(t *testing.T) {
	s := new(DropSuite)
	s.InitRoot(workgroup.Register)
	testutil.RunSuite(t, s)
}

func (s *DropSuite) TestDrop_RequiresName() {
	wsDir, _ := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "drop")
	s.Require().Error(err)
}

func (s *DropSuite) TestDrop_UnknownWorkgroup_Errors() {
	wsDir, _ := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "drop", "nonexistent")
	s.Require().Error(err)
}

func (s *DropSuite) TestDrop_RemovesWorktreeAndConfig() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	treePath := filepath.Join(wsDir, ".workgroup", "feat", names[0])
	s.Assert().DirExists(treePath)

	out, err := s.ExecuteCmd("workgroup", "drop", "feat")
	s.Require().NoError(err, out)

	s.Assert().NoDirExists(treePath)
	s.Contains(out, "Dropped workgroup")

	// Config entry removed
	out, err = s.ExecuteCmd("workgroup", "list")
	s.Require().NoError(err)
	s.Contains(out, "no workgroups")
}

func (s *DropSuite) TestDrop_MissingWorktree_Idempotent() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	// Remove the worktree dir manually
	treePath := filepath.Join(wsDir, ".workgroup", "feat", names[0])
	s.Require().NoError(os.RemoveAll(treePath))

	out, err := s.ExecuteCmd("workgroup", "drop", "feat")
	s.Require().NoError(err, out)
}

func (s *DropSuite) TestDrop_DirtyTree_BlocksWithoutForce() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	treePath := filepath.Join(wsDir, ".workgroup", "feat", names[0])
	s.Require().NoError(os.WriteFile(filepath.Join(treePath, "dirty.txt"), []byte("x"), 0o644))

	_, err = s.ExecuteCmd("workgroup", "drop", "feat")
	s.Require().Error(err)
}

func (s *DropSuite) TestDrop_Force_OverridesSafetyCheck() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	treePath := filepath.Join(wsDir, ".workgroup", "feat", names[0])
	s.Require().NoError(os.WriteFile(filepath.Join(treePath, "dirty.txt"), []byte("x"), 0o644))

	out, err := s.ExecuteCmd("workgroup", "drop", "feat", "--force")
	s.Require().NoError(err, out)

	s.Assert().NoDirExists(treePath)
}

func (s *DropSuite) TestDrop_DeleteBranch() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	out, err := s.ExecuteCmd("workgroup", "drop", "feat", "--delete-branch")
	s.Require().NoError(err, out)

	repoDir := filepath.Join(wsDir, names[0])
	exists, err := branchExists(repoDir, "feat")
	s.Require().NoError(err)
	s.Assert().False(exists, "branch should be deleted")
}
