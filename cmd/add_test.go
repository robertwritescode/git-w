package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/suite"
)

// WorkspaceSuite is a base suite that creates a .gitworkspace in a temp dir
// and changes into it before each test.
type WorkspaceSuite struct {
	suite.Suite
	wsDir string
}

func (s *WorkspaceSuite) SetupTest() {
	s.wsDir = s.T().TempDir()
	cfgPath := filepath.Join(s.wsDir, ".gitworkspace")
	s.Require().NoError(os.WriteFile(cfgPath, []byte("[workspace]\nname = \"testws\"\n"), 0o644))
	changeToDir(s.T(), s.wsDir)
}

type AddSuite struct {
	WorkspaceSuite
}

func TestAddSuite(t *testing.T) {
	suite.Run(t, new(AddSuite))
}

func (s *AddSuite) TestAdd() {
	tests := []struct {
		name           string
		addFlags       []string
		wantGroup      string
		checkGitignore bool
	}{
		{
			name: "registers repo",
		},
		{
			name:      "with group membership",
			addFlags:  []string{"-g", "mygroup"},
			wantGroup: "mygroup",
		},
		{
			name:           "updates gitignore",
			checkGitignore: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir := s.T().TempDir()
			s.Require().NoError(os.WriteFile(
				filepath.Join(wsDir, ".gitworkspace"),
				[]byte("[workspace]\nname = \"testws\"\n"), 0o644,
			))
			changeToDir(s.T(), wsDir)

			repoDir := s.T().TempDir()
			testutil.MakeGitRepo(s.T(), repoDir)
			name := filepath.Base(repoDir)

			cmdArgs := append(append([]string{"add"}, tt.addFlags...), repoDir)
			_, err := execCmd(s.T(), cmdArgs...)
			s.Require().NoError(err)

			cfg, err := config.Load(filepath.Join(wsDir, ".gitworkspace"))
			s.Require().NoError(err)

			_, exists := cfg.Repos[name]
			s.Assert().True(exists)

			if tt.wantGroup != "" {
				s.Assert().Contains(cfg.Groups[tt.wantGroup].Repos, name)
			}
			if tt.checkGitignore {
				data, err := os.ReadFile(filepath.Join(wsDir, ".gitignore"))
				s.Require().NoError(err)
				s.Assert().Contains(string(data), cfg.Repos[name].Path)
			}
		})
	}
}

func (s *AddSuite) TestErrorNotGitRepo() {
	notARepo := s.T().TempDir()
	_, err := execCmd(s.T(), "add", notARepo)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "not a git repository")
}

func (s *AddSuite) TestErrorAlreadyRegistered() {
	repoDir := s.T().TempDir()
	testutil.MakeGitRepo(s.T(), repoDir)

	_, err := execCmd(s.T(), "add", repoDir)
	s.Require().NoError(err)

	_, err = execCmd(s.T(), "add", repoDir)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "already registered")
}

func (s *AddSuite) TestDetectsRemoteURL() {
	repoDir := s.T().TempDir()
	testutil.MakeGitRepo(s.T(), repoDir)

	fakeURL := "file:///tmp/fake-origin.git"
	c := exec.Command("git", "-C", repoDir, "remote", "add", "origin", fakeURL)
	s.Require().NoError(c.Run())

	_, err := execCmd(s.T(), "add", repoDir)
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(s.wsDir, ".gitworkspace"))
	s.Require().NoError(err)

	name := filepath.Base(repoDir)
	s.Assert().NotEmpty(cfg.Repos[name].URL)
	s.Assert().Contains(cfg.Repos[name].URL, "fake-origin")
}
