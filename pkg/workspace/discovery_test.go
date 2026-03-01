package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/stretchr/testify/suite"
)

type DiscoverySuite struct {
	suite.Suite
	root    string
	cfgPath string
}

func (s *DiscoverySuite) SetupTest() {
	s.root = s.T().TempDir()
	s.cfgPath = filepath.Join(s.root, ".gitw")
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte("[workspace]\nname=\"t\"\n"), 0o644))
}

func TestDiscoverySuite(t *testing.T) {
	suite.Run(t, new(DiscoverySuite))
}

func (s *DiscoverySuite) TestWalksUp() {
	tests := []struct {
		name        string
		relStartDir string // relative to s.root; empty = start at root itself
	}{
		{name: "at root"},
		{name: "one level deep", relStartDir: "subdir"},
		{name: "two levels deep", relStartDir: filepath.Join("a", "b")},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			startDir := s.root
			if tt.relStartDir != "" {
				startDir = filepath.Join(s.root, tt.relStartDir)
				s.Require().NoError(os.MkdirAll(startDir, 0o755))
			}

			found, err := workspace.Discover(startDir)
			s.Require().NoError(err)
			s.Assert().Equal(s.cfgPath, found)
		})
	}
}

func (s *DiscoverySuite) TestNotFound() {
	dir := s.T().TempDir()
	_, err := workspace.Discover(dir)
	s.Require().Error(err)
	s.Assert().ErrorIs(err, workspace.ErrNotFound)
}

func (s *DiscoverySuite) TestEnvVarOverride() {
	s.T().Setenv("GIT_W_CONFIG", "/custom/path/.gitw")

	found, err := workspace.Discover(s.root)
	s.Require().NoError(err)
	s.Assert().Equal("/custom/path/.gitw", found)
}
