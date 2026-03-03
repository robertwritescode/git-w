package repo_test

import (
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
)

type RemoveSuite struct {
	testutil.CmdSuite
	wsDir string
}

func (s *RemoveSuite) SetupTest() {
	s.CmdSuite.SetupTest()
	s.wsDir = s.SetupWorkspaceDir()
}

func TestRemoveSuite(t *testing.T) {
	s := new(RemoveSuite)
	s.InitRoot(repo.Register)
	testutil.RunSuite(t, s)
}

func (s *RemoveSuite) TestRemove() {
	tests := []struct {
		name    string
		addRepo bool
		rmArgs  []string
		wantErr bool
	}{
		{
			name:    "removes registered repo",
			addRepo: true,
		},
		{
			name:    "error when repo not found",
			rmArgs:  []string{"nonexistent"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			name := ""
			if tt.addRepo {
				repoDir := s.MakeGitRepo("")
				name = filepath.Base(repoDir)
				_, err := s.ExecuteCmd("repo", "add", repoDir)
				s.Require().NoError(err)
			}

			rmArgs := tt.rmArgs
			if len(rmArgs) == 0 {
				rmArgs = []string{name}
			}

			_, err := s.ExecuteCmd(append([]string{"repo", "unlink"}, rmArgs...)...)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			cfg, err := config.Load(filepath.Join(s.wsDir, ".gitw"))
			s.Require().NoError(err)

			_, exists := cfg.Repos[name]
			s.Assert().False(exists, "repo should be removed")
		})
	}
}

func (s *RemoveSuite) TestMultiple() {
	repo1Dir := s.MakeGitRepo("")
	repo2Dir := s.MakeGitRepo("")
	name1 := filepath.Base(repo1Dir)
	name2 := filepath.Base(repo2Dir)

	_, err := s.ExecuteCmd("repo", "add", repo1Dir)
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("repo", "add", repo2Dir)
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("repo", "unlink", name1, name2)
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(s.wsDir, ".gitw"))
	s.Require().NoError(err)

	s.Assert().Empty(cfg.Repos)
}

func (s *RemoveSuite) TestUpdatesGroups() {
	repoDir := s.MakeGitRepo("")
	name := filepath.Base(repoDir)

	_, err := s.ExecuteCmd("repo", "add", "-g", "mygroup", repoDir)
	s.Require().NoError(err)

	_, err = s.ExecuteCmd("repo", "unlink", name)
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(s.wsDir, ".gitw"))
	s.Require().NoError(err)

	s.Assert().NotContains(cfg.Groups["mygroup"].Repos, name)
}
