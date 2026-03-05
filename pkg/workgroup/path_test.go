package workgroup_test

import (
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workgroup"
)

type PathSuite struct {
	testutil.CmdSuite
}

func TestPathSuite(t *testing.T) {
	s := new(PathSuite)
	s.InitRoot(workgroup.Register)
	testutil.RunSuite(t, s)
}

func (s *PathSuite) TestPath_RequiresName() {
	wsDir, _ := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "path")
	s.Require().Error(err)
}

func (s *PathSuite) TestPath_UnknownWorkgroup_Errors() {
	wsDir, _ := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "path", "nonexistent")
	s.Require().Error(err)
}

func (s *PathSuite) TestPath_WorkgroupRoot() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	out, err := s.ExecuteCmd("workgroup", "path", "feat")
	s.Require().NoError(err)

	expected := filepath.Join(wsDir, ".workgroup", "feat")
	s.Contains(out, expected)
}

func (s *PathSuite) TestPath_RepoSpecific() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	out, err := s.ExecuteCmd("workgroup", "path", "feat", names[0])
	s.Require().NoError(err)

	expected := filepath.Join(wsDir, ".workgroup", "feat", names[0])
	s.Contains(out, expected)
}
