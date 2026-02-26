package cmd

import (
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type RemoveSuite struct {
	WorkspaceSuite
}

func TestRemoveSuite(t *testing.T) {
	suite.Run(t, new(RemoveSuite))
}

func (s *RemoveSuite) TestRemove() {
	tests := []struct {
		name    string
		addRepo bool     // register a real repo before removing
		rmArgs  []string // explicit rm args; derived from added repo when addRepo is true
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
				repoDir := testutil.MakeGitRepo(s.T(), "")
				name = filepath.Base(repoDir)
				_, err := execCmd(s.T(), "add", repoDir)
				s.Require().NoError(err)
			}

			rmArgs := tt.rmArgs
			if len(rmArgs) == 0 {
				rmArgs = []string{name}
			}

			_, err := execCmd(s.T(), append([]string{"rm"}, rmArgs...)...)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)

			cfg, err := config.Load(filepath.Join(s.wsDir, ".gitworkspace"))
			s.Require().NoError(err)
			_, exists := cfg.Repos[name]
			s.Assert().False(exists, "repo should be removed")
		})
	}
}

func (s *RemoveSuite) TestMultiple() {
	repo1Dir := testutil.MakeGitRepo(s.T(), "")
	repo2Dir := testutil.MakeGitRepo(s.T(), "")
	name1 := filepath.Base(repo1Dir)
	name2 := filepath.Base(repo2Dir)

	_, err := execCmd(s.T(), "add", repo1Dir)
	s.Require().NoError(err)
	_, err = execCmd(s.T(), "add", repo2Dir)
	s.Require().NoError(err)

	_, err = execCmd(s.T(), "rm", name1, name2)
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(s.wsDir, ".gitworkspace"))
	s.Require().NoError(err)
	s.Assert().Empty(cfg.Repos)
}

func (s *RemoveSuite) TestUpdatesGroups() {
	repoDir := testutil.MakeGitRepo(s.T(), "")
	name := filepath.Base(repoDir)

	_, err := execCmd(s.T(), "add", "-g", "mygroup", repoDir)
	s.Require().NoError(err)

	_, err = execCmd(s.T(), "rm", name)
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(s.wsDir, ".gitworkspace"))
	s.Require().NoError(err)
	s.Assert().NotContains(cfg.Groups["mygroup"].Repos, name)
}
