package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func (s *ExecSuite) addGroup(wsDir, groupName, repoName string) {
	cfgData, err := os.ReadFile(filepath.Join(wsDir, ".gitworkspace"))
	s.Require().NoError(err)
	groupTOML := fmt.Sprintf("\n[groups.%s]\nrepos = [%q]\n", groupName, repoName)
	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitworkspace"),
		append(cfgData, []byte(groupTOML)...),
		0o644,
	))
}

func (s *ExecSuite) setActiveContext(wsDir, ctxName string) {
	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitworkspace.local"),
		[]byte("[context]\nactive = \""+ctxName+"\"\n"),
		0o644,
	))
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
			wsDir, names := s.makeWsWithRepos(2)
			changeToDir(s.T(), wsDir)
			out, err := execCmd(s.T(), tt.args...)
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
		group     string // group "web" containing names[0]; empty = no group
		activeCtx string // active context name; empty = none
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
			wsDir, names := s.makeWsWithRepos(tt.nRepos)
			if tt.group != "" {
				s.addGroup(wsDir, tt.group, names[0])
			}
			if tt.activeCtx != "" {
				s.setActiveContext(wsDir, tt.activeCtx)
			}
			changeToDir(s.T(), wsDir)
			out, err := execCmd(s.T(), tt.args(names)...)
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
		wantMsg string // substring expected in error; empty = any error
	}{
		{"unknown repo", []string{"exec", "nonexistent", "--", "status"}, "nonexistent"},
		{"invalid git subcommand", []string{"exec", "--", "invalid-subcommand"}, ""},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, _ := s.makeWsWithRepos(1)
			changeToDir(s.T(), wsDir)
			_, err := execCmd(s.T(), tt.args...)
			s.Require().Error(err)
			if tt.wantMsg != "" {
				s.Assert().Contains(err.Error(), tt.wantMsg)
			}
		})
	}
}

func (s *ExecSuite) TestExec_Deduplication() {
	wsDir, names := s.makeWsWithRepos(2)
	s.addGroup(wsDir, "web", names[0])

	// Baseline: single-repo run to determine how many prefixed lines git produces.
	changeToDir(s.T(), wsDir)
	outSingle, err := execCmd(s.T(), "exec", names[0], "--", "status")
	s.Require().NoError(err)
	singleCount := strings.Count(outSingle, "["+names[0]+"]")

	// Dedup run: group "web" + names[0] explicitly — should run the repo exactly once.
	changeToDir(s.T(), wsDir)
	outDedup, err := execCmd(s.T(), "exec", "web", names[0], "--", "status")
	s.Require().NoError(err)
	s.Assert().Equal(singleCount, strings.Count(outDedup, "["+names[0]+"]"))
}
