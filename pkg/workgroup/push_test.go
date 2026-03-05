package workgroup_test

import (
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workgroup"
)

type PushSuite struct {
	testutil.CmdSuite
}

func TestPushSuite(t *testing.T) {
	s := new(PushSuite)
	s.InitRoot(workgroup.Register)
	testutil.RunSuite(t, s)
}

func (s *PushSuite) TestPush_RequiresName() {
	wsDir, _ := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "push")
	s.Require().Error(err)
}

func (s *PushSuite) TestPush_UnknownWorkgroup_Errors() {
	wsDir, _ := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "push", "nonexistent")
	s.Require().Error(err)
}

func (s *PushSuite) TestPush_WithRemote_Pushes() {
	wsDir, names := makeWorkspaceWithRemoteRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	out, err := s.ExecuteCmd("workgroup", "create", "feat", "--push", "--no-upstream")
	s.Require().NoError(err, out)

	out, err = s.ExecuteCmd("workgroup", "push", "feat")
	s.Require().NoError(err, out)

	s.Contains(out, "push")
	_ = names
}

func (s *PushSuite) TestPush_NoRemote_SkipsWithMessage() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", names[0], "--no-push", "--no-upstream")
	s.Require().NoError(err)

	out, err := s.ExecuteCmd("workgroup", "push", "feat")
	s.Require().NoError(err, out)

	s.Contains(out, "no remote, skipped")
}

func (s *PushSuite) TestPush_Summary() {
	wsDir, names := makeWorkspaceWithLocalRepos(&s.CmdSuite, 2)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("workgroup", "create", "feat", "--no-push", "--no-upstream")
	s.Require().NoError(err)

	out, err := s.ExecuteCmd("workgroup", "push", "feat")
	s.Require().NoError(err, out)

	s.Contains(out, "work push complete: 2 ok, 0 failed")
	_ = names
}
