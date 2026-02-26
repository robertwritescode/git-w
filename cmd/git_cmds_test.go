package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type GitCmdsSuite struct {
	suite.Suite
}

func TestGitCmds(t *testing.T) {
	suite.Run(t, new(GitCmdsSuite))
}

// makeWsWithRemoteRepos creates a workspace with n repos, each having a bare remote
// with the initial commit pushed to it (so fetch/pull work). Returns wsDir and names.
func (s *GitCmdsSuite) makeWsWithRemoteRepos(n int) (string, []string) {
	wsDir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitworkspace"),
		[]byte("[workspace]\nname = \"test\"\n"), 0o644,
	))
	changeToDir(s.T(), wsDir)

	names := make([]string, n)
	for i := range names {
		remoteDir := s.T().TempDir()
		testutil.GitInitBare(s.T(), remoteDir)
		repoDir := testutil.MakeGitRepo(s.T(), "file://"+remoteDir)
		pushToRemote(s.T(), repoDir)
		_, err := execCmd(s.T(), "add", repoDir)
		s.Require().NoError(err)
		names[i] = filepath.Base(repoDir)
	}
	return wsDir, names
}

func pushToRemote(t *testing.T, repoDir string) {
	t.Helper()
	cmd := exec.Command("git", "push", "-u", "origin", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("git push: %s", out)
		t.Fatal(err)
	}
}

func (s *GitCmdsSuite) TestGitCmd_RunsInAllRepos() {
	tests := []struct {
		name       string
		cmdName    string
		setup      func() (wsDir string, names []string)
		checkNames bool // whether output is expected to contain repo names (some cmds produce no output)
	}{
		{
			name:       "fetch",
			cmdName:    "fetch",
			setup:      func() (string, []string) { return s.makeWsWithRemoteRepos(2) },
			checkNames: false, // git fetch with nothing new produces no output
		},
		{
			name:       "pull",
			cmdName:    "pull",
			setup:      func() (string, []string) { return s.makeWsWithRemoteRepos(2) },
			checkNames: true, // git pull prints "Already up to date." to stdout
		},
		{
			name:    "status",
			cmdName: "status",
			setup: func() (string, []string) {
				wsDir := s.T().TempDir()
				s.Require().NoError(os.WriteFile(
					filepath.Join(wsDir, ".gitworkspace"),
					[]byte("[workspace]\nname = \"test\"\n"), 0o644,
				))
				changeToDir(s.T(), wsDir)
				names := make([]string, 2)
				for i := range names {
					dir := testutil.MakeGitRepo(s.T(), "")
					_, err := execCmd(s.T(), "add", dir)
					s.Require().NoError(err)
					names[i] = filepath.Base(dir)
				}
				return wsDir, names
			},
			checkNames: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir, names := tt.setup()
			changeToDir(s.T(), wsDir)
			out, err := execCmd(s.T(), tt.cmdName)
			s.Require().NoError(err)
			if tt.checkNames {
				for _, name := range names {
					s.Assert().Contains(out, name)
				}
			}
		})
	}
}

func (s *GitCmdsSuite) TestPush_RequiresRemote() {
	wsDir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitworkspace"),
		[]byte("[workspace]\nname = \"test\"\n"), 0o644,
	))
	changeToDir(s.T(), wsDir)

	repoDir := testutil.MakeGitRepo(s.T(), "")
	_, err := execCmd(s.T(), "add", repoDir)
	s.Require().NoError(err)

	changeToDir(s.T(), wsDir)
	_, err = execCmd(s.T(), "push")
	s.Require().Error(err)
}

func (s *GitCmdsSuite) TestGitCmd_ActiveContext_Scopes() {
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
			wsDir, names := s.makeWsWithRemoteRepos(2)
			cfgData, err := os.ReadFile(filepath.Join(wsDir, ".gitworkspace"))
			s.Require().NoError(err)
			groupTOML := fmt.Sprintf("\n[groups.web]\nrepos = [%q]\n", names[0])
			s.Require().NoError(os.WriteFile(
				filepath.Join(wsDir, ".gitworkspace"),
				append(cfgData, []byte(groupTOML)...),
				0o644,
			))
			s.Require().NoError(os.WriteFile(
				filepath.Join(wsDir, ".gitworkspace.local"),
				[]byte("[context]\nactive = \"web\"\n"),
				0o644,
			))
			changeToDir(s.T(), wsDir)
			out, err := execCmd(s.T(), tt.cmdName)
			s.Require().NoError(err)
			s.Assert().NotContains(out, "["+names[1]+"]")
		})
	}
}

func (s *GitCmdsSuite) TestStatus_AliasWorks() {
	wsDir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitworkspace"),
		[]byte("[workspace]\nname = \"test\"\n"), 0o644,
	))
	changeToDir(s.T(), wsDir)

	for range make([]struct{}, 2) {
		dir := testutil.MakeGitRepo(s.T(), "")
		_, err := execCmd(s.T(), "add", dir)
		s.Require().NoError(err)
	}

	changeToDir(s.T(), wsDir)
	outStatus, err := execCmd(s.T(), "status")
	s.Require().NoError(err)

	changeToDir(s.T(), wsDir)
	outAlias, err := execCmd(s.T(), "st")
	s.Require().NoError(err)

	s.Assert().Equal(outStatus, outAlias)
}
