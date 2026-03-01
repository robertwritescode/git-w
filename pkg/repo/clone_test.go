package repo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/stretchr/testify/suite"
)

type CloneSuite struct {
	testutil.CmdSuite
}

func (s *CloneSuite) SetupTest() {
	s.SetRoot(repo.Register)
}

func TestCloneSuite(t *testing.T) {
	suite.Run(t, new(CloneSuite))
}

func (s *CloneSuite) TestClone() {
	tests := []struct {
		name              string
		extraArgs         []string
		wantGroup         string
		checkGitignore    bool
		wantRepoName      string // if non-empty, assert repo name equals this
		wantRepoIsURLBase bool   // if true, assert repo name equals filepath.Base(bare repo dir)
	}{
		{name: "derives path from URL", wantRepoIsURLBase: true},
		{name: "uses explicit path", extraArgs: []string{"myrepo"}, wantRepoName: "myrepo"},
		{name: "adds to group", extraArgs: []string{"-g", "web"}, wantGroup: "web"},
		{name: "updates gitignore", checkGitignore: true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wsDir := s.T().TempDir()
			s.Require().NoError(os.WriteFile(
				filepath.Join(wsDir, ".gitw"),
				[]byte("[workspace]\nname = \"testws\"\n"), 0o644,
			))

			s.ChangeToDir(wsDir)

			absDir, fileURL := s.CreateBareRepo()

			args := append([]string{"repo", "clone", fileURL}, tt.extraArgs...)
			_, err := s.ExecuteCmd(args...)
			s.Require().NoError(err)

			cfg, err := workspace.Load(filepath.Join(wsDir, ".gitw"))
			s.Require().NoError(err)
			s.Require().NotEmpty(cfg.Repos)

			var repoName string
			var repoCfg workspace.RepoConfig
			for n, rc := range cfg.Repos {
				repoName = n
				repoCfg = rc
				break
			}

			s.Assert().Equal(fileURL, repoCfg.URL)

			cloneDest := filepath.Join(wsDir, repoCfg.Path)
			s.Assert().True(repo.IsGitRepo(cloneDest))

			if tt.wantRepoIsURLBase {
				s.Assert().Equal(filepath.Base(absDir), repoName)
			}

			if tt.wantRepoName != "" {
				s.Assert().Equal(tt.wantRepoName, repoName)
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
		filepath.Join(wsDir, ".gitw"),
		[]byte("[workspace]\nname = \"testws\"\n"), 0o644,
	))

	s.ChangeToDir(wsDir)

	_, fileURL := s.CreateBareRepo()

	_, err := s.ExecuteCmd("repo", "clone", fileURL)
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("repo", "clone", fileURL)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "already registered")
}

func (s *CloneSuite) TestCloneErrorNoArgs() {
	_, err := s.ExecuteCmd("repo", "clone")
	s.Require().Error(err)
}
