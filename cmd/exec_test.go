package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type ExecSuite struct {
	suite.Suite
}

func TestExec(t *testing.T) {
	suite.Run(t, new(ExecSuite))
}

// makeWsWithRepos creates a workspace in a temp dir, registers n real git repos,
// changes into the workspace dir, and returns wsDir and the repo names.
func (s *ExecSuite) makeWsWithRepos(n int) (string, []string) {
	wsDir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitworkspace"),
		[]byte("[workspace]\nname = \"test\"\n"), 0o644,
	))
	changeToDir(s.T(), wsDir)

	names := make([]string, n)
	for i := range names {
		dir := testutil.MakeGitRepo(s.T(), "")
		_, err := execCmd(s.T(), "add", dir)
		s.Require().NoError(err)
		names[i] = filepath.Base(dir)
	}
	return wsDir, names
}

func (s *ExecSuite) TestExec_RunsInAllRepos() {
	cases := []struct {
		name string
		args []string
	}{
		{"with separator", []string{"exec", "--", "status"}},
		{"without separator", []string{"exec", "status"}},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			wsDir, names := s.makeWsWithRepos(2)
			changeToDir(s.T(), wsDir)
			out, err := execCmd(s.T(), tc.args...)
			s.Require().NoError(err)
			for _, name := range names {
				s.Assert().Contains(out, "["+name+"]")
			}
		})
	}
}

func (s *ExecSuite) TestExec_FilterByRepoName() {
	wsDir, names := s.makeWsWithRepos(2)
	changeToDir(s.T(), wsDir)
	out, err := execCmd(s.T(), "exec", names[0], "--", "status")
	s.Require().NoError(err)
	s.Assert().Contains(out, "["+names[0]+"]")
	s.Assert().NotContains(out, "["+names[1]+"]")
}

func (s *ExecSuite) TestExec_UnknownRepo_Error() {
	wsDir, _ := s.makeWsWithRepos(1)
	changeToDir(s.T(), wsDir)
	_, err := execCmd(s.T(), "exec", "nonexistent", "--", "status")
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "nonexistent")
}

func (s *ExecSuite) TestExec_NonZeroGitExit_PropagatesError() {
	wsDir, _ := s.makeWsWithRepos(1)
	changeToDir(s.T(), wsDir)
	_, err := execCmd(s.T(), "exec", "--", "invalid-subcommand")
	s.Require().Error(err)
}
