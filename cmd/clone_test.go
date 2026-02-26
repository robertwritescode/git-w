package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type CloneSuite struct {
	WorkspaceSuite
}

func TestCloneSuite(t *testing.T) {
	suite.Run(t, new(CloneSuite))
}

func createBareRepo(t *testing.T) (absDir, fileURL string) {
	t.Helper()
	dir := t.TempDir()
	testutil.GitInitBare(t, dir)
	return dir, "file://" + dir
}

func (s *CloneSuite) TestClone() {
	tests := []struct {
		name           string
		extraArgs      []string
		wantGroup      string
		checkGitignore bool
	}{
		{name: "derives path from URL"},
		{name: "uses explicit path", extraArgs: []string{"myrepo"}},
		{name: "adds to group", extraArgs: []string{"-g", "web"}, wantGroup: "web"},
		{name: "updates gitignore", checkGitignore: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir := s.T().TempDir()
			s.Require().NoError(os.WriteFile(
				filepath.Join(wsDir, ".gitworkspace"),
				[]byte("[workspace]\nname = \"testws\"\n"), 0o644,
			))
			changeToDir(s.T(), wsDir)

			_, fileURL := createBareRepo(s.T())

			args := append([]string{"clone", fileURL}, tt.extraArgs...)
			_, err := execCmd(s.T(), args...)
			s.Require().NoError(err)

			cfg, err := config.Load(filepath.Join(wsDir, ".gitworkspace"))
			s.Require().NoError(err)
			s.Require().NotEmpty(cfg.Repos)

			var repoName string
			var repoCfg config.RepoConfig
			for n, rc := range cfg.Repos {
				repoName = n
				repoCfg = rc
				break
			}

			s.Assert().Equal(fileURL, repoCfg.URL)

			cloneDest := filepath.Join(wsDir, repoCfg.Path)
			s.Assert().True(isGitRepo(cloneDest))

			if tt.name == "derives path from URL" {
				s.Assert().Equal(deriveClonePath(fileURL), repoName)
			}
			if tt.name == "uses explicit path" {
				s.Assert().Equal("myrepo", repoName)
			}
			if tt.wantGroup != "" {
				s.Assert().Contains(cfg.Groups[tt.wantGroup].Repos, repoName)
			}
			if tt.checkGitignore {
				data, err := os.ReadFile(filepath.Join(wsDir, ".gitignore"))
				s.Require().NoError(err)
				s.Assert().Contains(string(data), repoCfg.Path)
			}
		})
	}
}

func (s *CloneSuite) TestCloneErrorAlreadyRegistered() {
	wsDir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitworkspace"),
		[]byte("[workspace]\nname = \"testws\"\n"), 0o644,
	))
	changeToDir(s.T(), wsDir)

	_, fileURL := createBareRepo(s.T())

	_, err := execCmd(s.T(), "clone", fileURL)
	s.Require().NoError(err)

	_, err = execCmd(s.T(), "clone", fileURL)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "already registered")
}

func (s *CloneSuite) TestCloneErrorNoArgs() {
	_, err := execCmd(s.T(), "clone")
	s.Require().Error(err)
}
