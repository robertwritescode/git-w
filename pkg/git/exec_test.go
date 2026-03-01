package git_test

import (
	"strings"
	"testing"

	gitpkg "github.com/robertwritescode/git-w/pkg/git"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type ExecSuite struct {
	testutil.CmdSuite
}

func (s *ExecSuite) SetupTest() {
	s.SetRoot(gitpkg.Register)
}

func TestExecSuite(t *testing.T) {
	suite.Run(t, new(ExecSuite))
}

func (s *ExecSuite) TestExec_RunsInAllRepos() {
	tests := []struct {
		name string
		args []string
	}{
		{"with separator", []string{"exec", "--", "status"}},
		{"without separator", []string{"exec", "status"}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)
			s.ChangeToDir(wsDir)

			out, err := s.ExecuteCmd(tt.args...)
			s.Require().NoError(err)

			for _, name := range names {
				s.Assert().Contains(out, "["+name+"]")
			}
		})
	}
}

func (s *ExecSuite) TestExec_Filtering() {
	tests := []struct {
		name      string
		nRepos    int
		group     string
		activeCtx string
		args      func(names []string) []string
		wantIn    func(names []string) []string
		wantNotIn func(names []string) []string
	}{
		{
			name:      "by repo name",
			nRepos:    2,
			args:      func(names []string) []string { return []string{"exec", names[0], "--", "status"} },
			wantIn:    func(names []string) []string { return names[:1] },
			wantNotIn: func(names []string) []string { return names[1:] },
		},
		{
			name:      "by active context",
			nRepos:    2,
			group:     "web",
			activeCtx: "web",
			args:      func(_ []string) []string { return []string{"exec", "--", "status"} },
			wantIn:    func(names []string) []string { return names[:1] },
			wantNotIn: func(names []string) []string { return names[1:] },
		},
		{
			name:      "by group name",
			nRepos:    2,
			group:     "web",
			args:      func(_ []string) []string { return []string{"exec", "web", "--", "status"} },
			wantIn:    func(names []string) []string { return names[:1] },
			wantNotIn: func(names []string) []string { return names[1:] },
		},
		{
			name:      "mixed repo and group",
			nRepos:    3,
			group:     "web",
			args:      func(names []string) []string { return []string{"exec", "web", names[1], "--", "status"} },
			wantIn:    func(names []string) []string { return names[:2] },
			wantNotIn: func(names []string) []string { return names[2:] },
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, names := s.MakeWorkspaceWithNLocalRepos(tt.nRepos)

			if tt.group != "" {
				s.AppendGroup(wsDir, tt.group, names[0])
			}

			if tt.activeCtx != "" {
				s.SetActiveContext(wsDir, tt.activeCtx)
			}

			s.ChangeToDir(wsDir)

			out, err := s.ExecuteCmd(tt.args(names)...)
			s.Require().NoError(err)

			for _, name := range tt.wantIn(names) {
				s.Assert().Contains(out, "["+name+"]")
			}

			for _, name := range tt.wantNotIn(names) {
				s.Assert().NotContains(out, "["+name+"]")
			}
		})
	}
}

func (s *ExecSuite) TestExec_Error() {
	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{"unknown repo", []string{"exec", "nonexistent", "--", "status"}, "nonexistent"},
		{"invalid git subcommand", []string{"exec", "--", "invalid-subcommand"}, ""},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, _ := s.MakeWorkspaceWithNLocalRepos(1)
			s.ChangeToDir(wsDir)

			_, err := s.ExecuteCmd(tt.args...)
			s.Require().Error(err)

			if tt.wantMsg != "" {
				s.Assert().Contains(err.Error(), tt.wantMsg)
			}
		})
	}
}

func (s *ExecSuite) TestExec_Deduplication() {
	wsDir, names := s.MakeWorkspaceWithNLocalRepos(2)
	s.AppendGroup(wsDir, "web", names[0])

	s.ChangeToDir(wsDir)
	outSingle, err := s.ExecuteCmd("exec", names[0], "--", "status")
	s.Require().NoError(err)

	singleCount := strings.Count(outSingle, "["+names[0]+"]")

	s.ChangeToDir(wsDir)
	outDedup, err := s.ExecuteCmd("exec", "web", names[0], "--", "status")
	s.Require().NoError(err)

	s.Assert().Equal(singleCount, strings.Count(outDedup, "["+names[0]+"]"))
}
