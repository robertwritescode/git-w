package git_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	gitpkg "github.com/robertwritescode/git-w/pkg/git"
	"github.com/robertwritescode/git-w/pkg/testutil"
)

type InfoSuite struct {
	testutil.CmdSuite
	wsDir string
}

func (s *InfoSuite) SetupTest() {
	s.CmdSuite.SetupTest()
	s.wsDir = s.SetupWorkspaceDir()
}

func TestInfoSuite(t *testing.T) {
	s := new(InfoSuite)
	s.InitRoot(gitpkg.Register)
	testutil.RunSuite(t, s)
}

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
			wsDir := s.T().TempDir()
			cfgPath := filepath.Join(wsDir, ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte("[workspace]\nname = \"testws\"\n"), 0o644))
			s.ChangeToDir(wsDir)

			// Register repos directly in config rather than using add command.
			dirs := make([]string, tt.numRepos)
			if tt.numRepos > 0 {
				cfg, err := config.Load(cfgPath)
				s.Require().NoError(err)
				for i := range dirs {
					dirs[i] = s.MakeGitRepo("")
					relPath, relErr := config.RelPath(cfgPath, dirs[i])
					s.Require().NoError(relErr)
					cfg.Repos[filepath.Base(dirs[i])] = config.RepoConfig{Path: relPath}
				}
				s.Require().NoError(config.Save(cfgPath, cfg))
			}

			out, err := s.ExecuteCmd(tt.cmd)
			s.Require().NoError(err)
			s.Assert().Contains(out, "REPO")
			for _, d := range dirs {
				s.Assert().Contains(out, filepath.Base(d))
			}
		})
	}
}

func (s *InfoSuite) TestInfo_ByGroup() {
	repoDir1 := s.MakeGitRepo("")
	repoDir2 := s.MakeGitRepo("")

	cfgPath := filepath.Join(s.wsDir, ".gitw")
	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)

	rel1, err := config.RelPath(cfgPath, repoDir1)
	s.Require().NoError(err)
	rel2, err := config.RelPath(cfgPath, repoDir2)
	s.Require().NoError(err)
	cfg.Repos[filepath.Base(repoDir1)] = config.RepoConfig{Path: rel1}
	cfg.Repos[filepath.Base(repoDir2)] = config.RepoConfig{Path: rel2}
	cfg.Groups["mygroup"] = config.GroupConfig{Repos: []string{filepath.Base(repoDir1)}}
	s.Require().NoError(config.Save(cfgPath, cfg))

	out, err := s.ExecuteCmd("info", "mygroup")
	s.Require().NoError(err)
	s.Assert().Contains(out, filepath.Base(repoDir1))
	s.Assert().NotContains(out, filepath.Base(repoDir2))
}

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
			setup:   func() { s.ChangeToDir(s.T().TempDir()) },
			args:    []string{"info"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setup()
			_, err := s.ExecuteCmd(tt.args...)
			s.Require().Error(err)
			if tt.wantErr != "" {
				s.Assert().Contains(err.Error(), tt.wantErr)
			}
		})
	}
}
