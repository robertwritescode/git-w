package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/stretchr/testify/suite"
)

type LoaderSuite struct {
	suite.Suite
	cfgPath string
}

func (s *LoaderSuite) SetupTest() {
	s.cfgPath = filepath.Join(s.T().TempDir(), ".gitworkspace")
}

func TestLoaderSuite(t *testing.T) {
	suite.Run(t, new(LoaderSuite))
}

func (s *LoaderSuite) TestRoundTrip() {
	content := `[workspace]
name = "myws"

[repos.frontend]
path = "apps/frontend"
url = "https://github.com/org/frontend"

[repos.backend]
path = "services/backend"

[groups.web]
repos = ["frontend", "backend"]
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().Equal("myws", cfg.Workspace.Name)
	s.Assert().Equal("apps/frontend", cfg.Repos["frontend"].Path)
	s.Assert().Equal("https://github.com/org/frontend", cfg.Repos["frontend"].URL)
	s.Assert().Equal("services/backend", cfg.Repos["backend"].Path)
	s.Assert().Equal([]string{"frontend", "backend"}, cfg.Groups["web"].Repos)
}

func (s *LoaderSuite) TestLoadErrors() {
	tests := []struct {
		name      string
		content   string // empty = do not create file
		wantErrIs error  // nil = any error is acceptable
	}{
		{
			name:      "missing file",
			wantErrIs: os.ErrNotExist,
		},
		{
			name:    "malformed TOML",
			content: "this is not toml :::",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitworkspace")
			if tt.content != "" {
				s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.content), 0o644))
			}

			_, err := config.Load(cfgPath)
			s.Require().Error(err)
			if tt.wantErrIs != nil {
				s.Assert().ErrorIs(err, tt.wantErrIs)
			}
		})
	}
}

func (s *LoaderSuite) TestLocalFileMerge() {
	tests := []struct {
		name         string
		localContent string // empty = do not create .local file
		wantContext  string
	}{
		{
			name:         "local file overrides context",
			localContent: "[context]\nactive = \"web\"\n",
			wantContext:  "web",
		},
		{
			name:        "missing local file leaves context empty",
			wantContext: "",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitworkspace")
			s.Require().NoError(os.WriteFile(cfgPath, []byte("[workspace]\nname = \"test\"\n"), 0o644))
			if tt.localContent != "" {
				s.Require().NoError(os.WriteFile(cfgPath+".local", []byte(tt.localContent), 0o644))
			}

			cfg, err := config.Load(cfgPath)
			s.Require().NoError(err)
			s.Assert().Equal(tt.wantContext, cfg.Context.Active)
		})
	}
}

func (s *LoaderSuite) TestSaveAtomic() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte("[workspace]\nname = \"original\"\n"), 0o644))
	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	cfg.Workspace.Name = "updated"
	s.Require().NoError(config.Save(s.cfgPath, cfg))

	_, err = os.Stat(s.cfgPath + ".tmp")
	s.Assert().True(os.IsNotExist(err))

	cfg2, err := config.Load(s.cfgPath)
	s.Require().NoError(err)
	s.Assert().Equal("updated", cfg2.Workspace.Name)
}

func (s *LoaderSuite) TestInitializesNilMaps() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte("[workspace]\nname = \"empty\"\n"), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)
	s.Assert().NotNil(cfg.Repos)
	s.Assert().NotNil(cfg.Groups)
}
