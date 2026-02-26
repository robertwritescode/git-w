package cmd

import (
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/robertwritescode/git-workspace/internal/testutil"
	"github.com/stretchr/testify/suite"
)

type RenameSuite struct {
	WorkspaceSuite
}

func TestRenameSuite(t *testing.T) {
	suite.Run(t, new(RenameSuite))
}

func (s *RenameSuite) TestRename() {
	repoDir := s.T().TempDir()
	testutil.MakeGitRepo(s.T(), repoDir)
	oldName := filepath.Base(repoDir)

	_, err := execCmd(s.T(), "add", repoDir)
	s.Require().NoError(err)

	cfg0, _ := config.Load(filepath.Join(s.wsDir, ".gitworkspace"))
	originalPath := cfg0.Repos[oldName].Path

	_, err = execCmd(s.T(), "rename", oldName, "newname")
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(s.wsDir, ".gitworkspace"))
	s.Require().NoError(err)

	_, oldExists := cfg.Repos[oldName]
	s.Assert().False(oldExists)
	_, newExists := cfg.Repos["newname"]
	s.Assert().True(newExists)
	s.Assert().Equal(originalPath, cfg.Repos["newname"].Path)
}

func (s *RenameSuite) TestUpdatesGroups() {
	repoDir := s.T().TempDir()
	testutil.MakeGitRepo(s.T(), repoDir)
	oldName := filepath.Base(repoDir)

	_, err := execCmd(s.T(), "add", "-g", "mygroup", repoDir)
	s.Require().NoError(err)

	_, err = execCmd(s.T(), "rename", oldName, "newname")
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(s.wsDir, ".gitworkspace"))
	s.Require().NoError(err)
	s.Assert().Contains(cfg.Groups["mygroup"].Repos, "newname")
	s.Assert().NotContains(cfg.Groups["mygroup"].Repos, oldName)
}

func (s *RenameSuite) TestErrorOldNotFound() {
	_, err := execCmd(s.T(), "rename", "nonexistent", "newname")
	s.Require().Error(err)
}

func (s *RenameSuite) TestErrorNewExists() {
	repo1Dir := s.T().TempDir()
	repo2Dir := s.T().TempDir()
	testutil.MakeGitRepo(s.T(), repo1Dir)
	testutil.MakeGitRepo(s.T(), repo2Dir)
	name1 := filepath.Base(repo1Dir)
	name2 := filepath.Base(repo2Dir)

	_, err := execCmd(s.T(), "add", repo1Dir)
	s.Require().NoError(err)
	_, err = execCmd(s.T(), "add", repo2Dir)
	s.Require().NoError(err)

	_, err = execCmd(s.T(), "rename", name1, name2)
	s.Require().Error(err)
}
