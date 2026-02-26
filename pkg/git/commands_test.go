package git_test

import (
	"testing"

	gitpkg "github.com/robertwritescode/git-w/pkg/git"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type GitSuite struct {
	testutil.CmdSuite
}

func (s *GitSuite) SetupTest() {
	s.SetRoot(gitpkg.Register)
}

func TestGitSuite(t *testing.T) {
	suite.Run(t, new(GitSuite))
}

func (s *GitSuite) TestGitCmd_RunsInAllRepos() {
	tests := []struct {
		name       string
		cmdName    string
		setup      func() (wsDir string, names []string)
		checkNames bool
	}{
		{
			name:       "fetch",
			cmdName:    "fetch",
			setup:      func() (string, []string) { return s.MakeWorkspaceWithNRemoteRepos(2) },
			checkNames: false,
		},
		{
			name:       "pull",
			cmdName:    "pull",
			setup:      func() (string, []string) { return s.MakeWorkspaceWithNRemoteRepos(2) },
			checkNames: true,
		},
		{
			name:       "status",
			cmdName:    "status",
			setup:      func() (string, []string) { return s.MakeWorkspaceWithNLocalRepos(2) },
			checkNames: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, names := tt.setup()
			s.ChangeToDir(wsDir)

			out, err := s.ExecuteCmd(tt.cmdName)
			s.Require().NoError(err)

			if tt.checkNames {
				for _, name := range names {
					s.Assert().Contains(out, name)
				}
			}
		})
	}
}

func (s *GitSuite) TestPush_RequiresRemote() {
	// MakeWorkspaceWithNLocalRepos creates repos without a remote; push should fail.
	wsDir, _ := s.MakeWorkspaceWithNLocalRepos(1)
	s.ChangeToDir(wsDir)

	_, err := s.ExecuteCmd("push")
	s.Require().Error(err)
}

func (s *GitSuite) TestGitCmd_ActiveContext_Scopes() {
	tests := []struct {
		name    string
		cmdName string
	}{
		{"fetch scopes to context", "fetch"},
		{"pull scopes to context", "pull"},
		{"status scopes to context", "status"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, names := s.MakeWorkspaceWithNRemoteRepos(2)

			s.AppendGroup(wsDir, "web", names[0])
			s.SetActiveContext(wsDir, "web")
			s.ChangeToDir(wsDir)

			out, err := s.ExecuteCmd(tt.cmdName)
			s.Require().NoError(err)
			s.Assert().NotContains(out, "["+names[1]+"]")
		})
	}
}

func (s *GitSuite) TestStatus_AliasWorks() {
	wsDir, _ := s.MakeWorkspaceWithNLocalRepos(2)

	s.ChangeToDir(wsDir)
	outStatus, err := s.ExecuteCmd("status")
	s.Require().NoError(err)

	s.ChangeToDir(wsDir)
	outAlias, err := s.ExecuteCmd("st")
	s.Require().NoError(err)

	s.Assert().Equal(outStatus, outAlias)
}
