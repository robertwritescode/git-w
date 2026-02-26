package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func changeToDir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

func execCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	cfgFile = ""
	return buf.String(), err
}

type InitSuite struct {
	suite.Suite
	dir string
}

func (s *InitSuite) SetupTest() {
	s.dir = s.T().TempDir()
	changeToDir(s.T(), s.dir)
}

func TestInitSuite(t *testing.T) {
	suite.Run(t, new(InitSuite))
}

func (s *InitSuite) TestWorkspaceCreation() {
	tests := []struct {
		name              string
		args              []string
		preGitignore      string
		preGitignoreDir   bool
		wantOutput        string
		wantInConfig      string
		wantGitignore     string
		wantGitignoreOnce bool
		wantStderr        string
	}{
		{
			name:         "custom name in config",
			args:         []string{"testws"},
			wantOutput:   "testws",
			wantInConfig: `name = "testws"`,
		},
		{
			name:         "defaults to directory name",
			wantInConfig: "name =",
		},
		{
			name:          "adds local entry to gitignore",
			args:          []string{"myws"},
			wantGitignore: ".gitworkspace.local",
		},
		{
			name:              "does not duplicate existing gitignore entry",
			args:              []string{"myws"},
			preGitignore:      ".gitworkspace.local\n",
			wantGitignore:     ".gitworkspace.local",
			wantGitignoreOnce: true,
		},
		{
			name:            "warns when gitignore cannot be written",
			args:            []string{"myws"},
			preGitignoreDir: true,
			wantInConfig:    `name = "myws"`,
			wantStderr:      "warning",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()
			changeToDir(s.T(), dir)

			if tt.preGitignore != "" {
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, ".gitignore"),
					[]byte(tt.preGitignore), 0o644,
				))
			}
			if tt.preGitignoreDir {
				s.Require().NoError(os.MkdirAll(filepath.Join(dir, ".gitignore"), 0o755))
			}

			outBuf := &bytes.Buffer{}
			errBuf := &bytes.Buffer{}
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(errBuf)
			rootCmd.SetArgs(append([]string{"init"}, tt.args...))
			err := rootCmd.Execute()
			rootCmd.SetArgs(nil)
			cfgFile = ""

			s.Require().NoError(err)

			if tt.wantOutput != "" {
				s.Assert().Contains(outBuf.String(), tt.wantOutput)
			}
			if tt.wantInConfig != "" {
				data, err := os.ReadFile(filepath.Join(dir, ".gitworkspace"))
				s.Require().NoError(err)
				s.Assert().Contains(string(data), tt.wantInConfig)
			}
			if tt.wantGitignore != "" {
				data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
				s.Require().NoError(err)
				if tt.wantGitignoreOnce {
					s.Assert().Equal(1, strings.Count(string(data), tt.wantGitignore))
				} else {
					s.Assert().Contains(string(data), tt.wantGitignore)
				}
			}
			if tt.wantStderr != "" {
				s.Assert().Contains(errBuf.String(), tt.wantStderr)
			}
		})
	}
}

func (s *InitSuite) TestErrorIfAlreadyExists() {
	_, err := execCmd(s.T(), "init", "first")
	s.Require().NoError(err)

	_, err = execCmd(s.T(), "init", "second")
	s.Require().Error(err)
}
