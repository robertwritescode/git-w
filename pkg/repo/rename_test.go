package repo_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/testutil"
)

type RenameSuite struct {
	testutil.CmdSuite
	wsDir string
}

func (s *RenameSuite) SetupTest() {
	s.CmdSuite.SetupTest()
	s.wsDir = s.SetupWorkspaceDir()
}

func TestRenameSuite(t *testing.T) {
	s := new(RenameSuite)
	s.InitRoot(repo.Register)
	testutil.RunSuite(t, s)
}

func (s *RenameSuite) TestRename() {
	cases := []struct {
		name         string
		setup        func(s *RenameSuite) (oldName, newName string)
		wantErr      string
		wantInGroups map[string][]string
	}{
		{
			name: "renames repo",
			setup: func(s *RenameSuite) (string, string) {
				repoDir := s.MakeGitRepo("")
				_, err := s.ExecuteCmd("repo", "add", repoDir)
				s.Require().NoError(err)
				return filepath.Base(repoDir), "newname"
			},
		},
		{
			name: "updates group membership",
			setup: func(s *RenameSuite) (string, string) {
				repoDir := s.MakeGitRepo("")
				_, err := s.ExecuteCmd("repo", "add", "-g", "mygroup", repoDir)
				s.Require().NoError(err)
				return filepath.Base(repoDir), "newname"
			},
			wantInGroups: map[string][]string{"mygroup": {"newname"}},
		},
		{
			name: "error: old name not found",
			setup: func(s *RenameSuite) (string, string) {
				return "nonexistent", "newname"
			},
			wantErr: "not found",
		},
		{
			name: "error: new name already exists",
			setup: func(s *RenameSuite) (string, string) {
				repo1Dir := s.MakeGitRepo("")
				repo2Dir := s.MakeGitRepo("")
				_, err := s.ExecuteCmd("repo", "add", repo1Dir)
				s.Require().NoError(err)
				_, err = s.ExecuteCmd("repo", "add", repo2Dir)
				s.Require().NoError(err)
				return filepath.Base(repo1Dir), filepath.Base(repo2Dir)
			},
			wantErr: "already exists",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			// Re-initialise workspace for each subtest so they are isolated.
			s.wsDir = s.SetupWorkspaceDir()
			// Re-register commands so they point at the new root.
			s.SetRoot(repo.Register)

			oldName, newName := tc.setup(s)

			_, err := s.ExecuteCmd("repo", "rename", oldName, newName)

			if tc.wantErr != "" {
				s.Require().Error(err)
				s.Assert().Contains(err.Error(), tc.wantErr)
				return
			}

			s.Require().NoError(err)

			cfg, loadErr := config.Load(filepath.Join(s.wsDir, ".gitw"))
			s.Require().NoError(loadErr)

			_, oldExists := cfg.Repos[oldName]
			s.Assert().False(oldExists, fmt.Sprintf("old name %q should not exist after rename", oldName))

			_, newExists := cfg.Repos[newName]
			s.Assert().True(newExists, fmt.Sprintf("new name %q should exist after rename", newName))

			for group, wantRepos := range tc.wantInGroups {
				for _, r := range wantRepos {
					s.Assert().Contains(cfg.Groups[group].Repos, r)
				}

				s.Assert().NotContains(cfg.Groups[group].Repos, oldName)
			}
		})
	}
}
