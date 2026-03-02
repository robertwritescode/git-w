package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workspace"
)

type InitSuite struct {
	testutil.CmdSuite
}

func (s *InitSuite) SetupTest() {
	s.CmdSuite.SetupTest()
	s.ChangeToDir(s.T().TempDir())
}

func TestInitSuite(t *testing.T) {
	s := new(InitSuite)
	s.InitRoot(workspace.Register)
	testutil.RunSuite(t, s)
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
			wantGitignore: ".gitw.local",
		},
		{
			name:              "does not duplicate existing gitignore entry",
			args:              []string{"myws"},
			preGitignore:      ".gitw.local\n",
			wantGitignore:     ".gitw.local",
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
			s.ChangeToDir(dir)

			if tt.preGitignore != "" {
				s.Require().NoError(os.WriteFile(
					filepath.Join(dir, ".gitignore"),
					[]byte(tt.preGitignore), 0o644,
				))
			}

			if tt.preGitignoreDir {
				s.Require().NoError(os.MkdirAll(filepath.Join(dir, ".gitignore"), 0o755))
			}

			out, err := s.ExecuteCmd(append([]string{"init"}, tt.args...)...)
			s.Require().NoError(err)

			if tt.wantOutput != "" {
				s.Assert().Contains(out, tt.wantOutput)
			}

			if tt.wantInConfig != "" {
				data, err := os.ReadFile(filepath.Join(dir, ".gitw"))
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
				s.Assert().Contains(out, tt.wantStderr)
			}
		})
	}
}

func (s *InitSuite) TestErrorIfAlreadyExists() {
	_, err := s.ExecuteCmd("init", "first")
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("init", "second")
	s.Require().Error(err)
}
