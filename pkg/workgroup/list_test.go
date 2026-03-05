package workgroup_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workgroup"
)

type ListSuite struct {
	testutil.CmdSuite
}

func TestListSuite(t *testing.T) {
	s := new(ListSuite)
	s.InitRoot(workgroup.Register)
	testutil.RunSuite(t, s)
}

func (s *ListSuite) TestList_EmptyState() {
	wsDir, _ := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("workgroup", "list")
	s.Require().NoError(err)
	s.Contains(out, "no workgroups")
}

func (s *ListSuite) TestList_ShowsWorkgroups() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 2)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat-a", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("workgroup", "create", "feat-b", names[1], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	out, err := s.ExecuteCmd("workgroup", "list")
	s.Require().NoError(err)

	s.Contains(out, "feat-a")
	s.Contains(out, "feat-b")
}

func (s *ListSuite) TestList_ShowsBranchAndRepoCount() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 2)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "my-feat", "--no-push", "--no-upstream")
	s.Require().NoError(err)

	out, err := s.ExecuteCmd("workgroup", "list")
	s.Require().NoError(err)

	s.Contains(out, "branch=my-feat")
	s.Contains(out, "repos=2")
	_ = names
}

func (s *ListSuite) TestList_DeterministicOrder() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 2)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "zzz", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("workgroup", "create", "aaa", names[1], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	out, err := s.ExecuteCmd("workgroup", "list")
	s.Require().NoError(err)

	aaaIdx := strings.Index(out, "aaa")
	zzzIdx := strings.Index(out, "zzz")
	s.Assert().GreaterOrEqual(aaaIdx, 0)
	s.Assert().GreaterOrEqual(zzzIdx, 0)
	s.Assert().Less(aaaIdx, zzzIdx)
}

func (s *ListSuite) TestList_ShowsConfiguredPath() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	cfgPath := filepath.Join(wsDir, ".gitw")
	out, err := s.ExecuteCmd("workgroup", "create", "feat", "--no-push", "--no-upstream", "--config", cfgPath)
	s.Require().NoError(err, out)

	out, err = s.ExecuteCmd("workgroup", "list", "--config", cfgPath)
	s.Require().NoError(err)

	s.Contains(out, "feat")
	_ = names
}
