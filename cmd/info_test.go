package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type InfoSuite struct {
	WorkspaceSuite
}

func TestInfoSuite(t *testing.T) {
	suite.Run(t, new(InfoSuite))
}

// TestInfo_Output covers command invocations that produce table output.
func (s *InfoSuite) TestInfo_Output() {
	tests := []struct {
		name     string
		numRepos int
		cmd      string
	}{
		{"all repos via info", 2, "info"},
		{"all repos via ll alias", 1, "ll"},
		{"empty workspace", 0, "info"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Fresh workspace per sub-test: SetupTest only runs once per method.
			wsDir := s.T().TempDir()
			cfgPath := filepath.Join(wsDir, ".gitworkspace")
			s.Require().NoError(os.WriteFile(cfgPath, []byte("[workspace]\nname = \"testws\"\n"), 0o644))
			changeToDir(s.T(), wsDir)

			dirs := make([]string, tt.numRepos)
			for i := range dirs {
				dirs[i] = testutil.MakeGitRepo(s.T(), s.T().TempDir())
				_, err := execCmd(s.T(), "add", dirs[i])
				s.Require().NoError(err)
			}

			out, err := execCmd(s.T(), tt.cmd)
			s.Require().NoError(err)
			s.Assert().Contains(out, "REPO")
			for _, d := range dirs {
				s.Assert().Contains(out, filepath.Base(d))
			}
		})
	}
}

func (s *InfoSuite) TestInfo_ByGroup() {
	repoDir1 := testutil.MakeGitRepo(s.T(), s.T().TempDir())
	repoDir2 := testutil.MakeGitRepo(s.T(), s.T().TempDir())

	_, err := execCmd(s.T(), "add", repoDir1)
	s.Require().NoError(err)
	_, err = execCmd(s.T(), "add", repoDir2)
	s.Require().NoError(err)

	// Directly assign group to avoid cobra flag state leaking between calls.
	cfgPath := filepath.Join(s.wsDir, ".gitworkspace")
	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	cfg.Groups["mygroup"] = config.GroupConfig{Repos: []string{filepath.Base(repoDir1)}}
	s.Require().NoError(config.Save(cfgPath, cfg))

	out, err := execCmd(s.T(), "info", "mygroup")
	s.Require().NoError(err)
	s.Assert().Contains(out, filepath.Base(repoDir1))
	s.Assert().NotContains(out, filepath.Base(repoDir2))
}

// TestInfo_Errors covers command invocations that must return an error.
func (s *InfoSuite) TestInfo_Errors() {
	tests := []struct {
		name    string
		setup   func()
		args    []string
		wantErr string
	}{
		{
			name:    "group not found",
			setup:   func() {},
			args:    []string{"info", "nonexistent"},
			wantErr: "not found",
		},
		{
			name:    "missing config",
			setup:   func() { changeToDir(s.T(), s.T().TempDir()) },
			args:    []string{"info"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setup()
			_, err := execCmd(s.T(), tt.args...)
			s.Require().Error(err)
			if tt.wantErr != "" {
				s.Assert().Contains(err.Error(), tt.wantErr)
			}
		})
	}
}
