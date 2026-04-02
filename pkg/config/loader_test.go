package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type LoaderSuite struct {
	suite.Suite
	cfgPath string
}

func (s *LoaderSuite) SetupTest() {
	s.cfgPath = filepath.Join(s.T().TempDir(), ".gitw")
}

func TestLoaderSuite(t *testing.T) {
	testutil.RunSuite(t, new(LoaderSuite))
}

func (s *LoaderSuite) TestRoundTrip() {
	content := `[metarepo]
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

	s.Assert().Equal("myws", cfg.Metarepo.Name)
	s.Assert().Equal("apps/frontend", cfg.Repos["frontend"].Path)
	s.Assert().Equal("https://github.com/org/frontend", cfg.Repos["frontend"].URL)
	s.Assert().Equal("services/backend", cfg.Repos["backend"].Path)
	s.Assert().Equal([]string{"frontend", "backend"}, cfg.Groups["web"].Repos)
}

func (s *LoaderSuite) TestLoadErrors() {
	tests := []struct {
		name      string
		content   string
		wantErrIs error
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
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
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
		localContent string
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
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte("[metarepo]\nname = \"test\"\n"), 0o644))

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
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte("[metarepo]\nname = \"original\"\n"), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	cfg.Metarepo.Name = "updated"
	s.Require().NoError(config.Save(s.cfgPath, cfg))

	_, err = os.Stat(s.cfgPath + ".tmp")
	s.Assert().True(errors.Is(err, os.ErrNotExist))

	cfg2, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().Equal("updated", cfg2.Metarepo.Name)
}

func (s *LoaderSuite) TestInitializesNilMaps() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte("[metarepo]\nname = \"empty\"\n"), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().NotNil(cfg.Repos)
	s.Assert().NotNil(cfg.Groups)
	s.Assert().NotNil(cfg.Worktrees)
}

func (s *LoaderSuite) TestSynthesizesWorktreeReposAndGroups() {
	content := `[metarepo]
name = "myws"

[worktrees.infra]
url = "https://github.com/org/infra"
bare_path = "infra/.bare"

[worktrees.infra.branches]
dev = "infra/dev"
test = "infra/test"
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Require().Contains(cfg.Repos, "infra-dev")
	s.Require().Contains(cfg.Repos, "infra-test")
	s.Assert().Equal("infra/dev", cfg.Repos["infra-dev"].Path)
	s.Assert().Equal("infra/test", cfg.Repos["infra-test"].Path)
	s.Assert().Equal("https://github.com/org/infra", cfg.Repos["infra-dev"].URL)

	s.Require().Contains(cfg.Groups, "infra")
	s.Assert().Equal([]string{"infra-dev", "infra-test"}, cfg.Groups["infra"].Repos)
}

func (s *LoaderSuite) TestSaveOmitsSynthesizedWorktreeTargets() {
	content := `[metarepo]
name = "myws"

[worktrees.infra]
url = "https://github.com/org/infra"
bare_path = "infra/.bare"

[worktrees.infra.branches]
dev = "infra/dev"
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)
	s.Require().NoError(config.Save(s.cfgPath, cfg))

	data, err := os.ReadFile(s.cfgPath)
	s.Require().NoError(err)
	text := string(data)

	s.Assert().Contains(text, "[worktrees.infra]")
	s.Assert().NotContains(text, "[repos.infra-dev]")
	s.Assert().NotContains(text, "[groups.infra]")
}

func (s *LoaderSuite) TestWorktreeSynthesizedNameConflicts() {
	tests := []struct {
		name    string
		toml    string
		wantErr string
	}{
		{
			name: "repo name conflict",
			toml: `[metarepo]
name = "ws"

[repos.infra-dev]
path = "apps/infra-dev"

[worktrees.infra]
url = "https://github.com/org/infra"
bare_path = "infra/.bare"

[worktrees.infra.branches]
dev = "infra/dev"
`,
			wantErr: "conflicts with existing repo",
		},
		{
			name: "group name conflict",
			toml: `[metarepo]
name = "ws"

[groups.infra]
repos = []

[worktrees.infra]
url = "https://github.com/org/infra"
bare_path = "infra/.bare"

[worktrees.infra.branches]
dev = "infra/dev"
`,
			wantErr: "conflicts with existing group",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			_, err := config.Load(cfgPath)
			s.Require().Error(err)
			s.Assert().Contains(err.Error(), tt.wantErr)
		})
	}
}

func (s *LoaderSuite) TestRejectsInvalidRepoPaths() {
	tests := []struct {
		name    string
		toml    string
		wantErr string
	}{
		{
			name: "absolute repo path",
			toml: `[metarepo]
name = "ws"

[repos.bad]
path = "/tmp/repo"
`,
			wantErr: "path must be relative",
		},
		{
			name: "empty repo path",
			toml: `[metarepo]
name = "ws"

[repos.bad]
path = ""
`,
			wantErr: "path is empty",
		},
		{
			name: "path escapes workspace root",
			toml: `[metarepo]
name = "ws"

[repos.bad]
path = "../outside"
`,
			wantErr: "path resolves outside workspace root",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			_, err := config.Load(cfgPath)
			s.Require().Error(err)
			s.Assert().Contains(err.Error(), tt.wantErr)
		})
	}
}

func (s *LoaderSuite) TestRejectsInvalidWorktreeBarePaths() {
	tests := []struct {
		name    string
		toml    string
		wantErr string
	}{
		{
			name: "absolute bare_path",
			toml: `[metarepo]
name = "ws"

[worktrees.infra]
url = "https://github.com/org/infra"
bare_path = "/tmp/.bare"
`,
			wantErr: "path must be relative",
		},
		{
			name: "bare_path escapes workspace root",
			toml: `[metarepo]
name = "ws"

[worktrees.infra]
url = "https://github.com/org/infra"
bare_path = "../outside/.bare"
`,
			wantErr: "path resolves outside workspace root",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			_, err := config.Load(cfgPath)
			s.Require().Error(err)
			s.Assert().Contains(err.Error(), tt.wantErr)
		})
	}
}

func (s *LoaderSuite) TestResolveRepoPath() {
	tests := []struct {
		name     string
		repoPath string
		wantErr  string
	}{
		{name: "relative path allowed", repoPath: "apps/frontend"},
		{name: "absolute path rejected", repoPath: "/tmp/repo", wantErr: "path must be relative"},
		{name: "escape path rejected", repoPath: "../repo", wantErr: "path resolves outside workspace root"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			_, err := config.ResolveRepoPath(cfgPath, tt.repoPath)

			if tt.wantErr != "" {
				s.Require().Error(err)
				s.Assert().Contains(err.Error(), tt.wantErr)
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *LoaderSuite) TestSaveLocalWorkgroup_RoundTrip() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte("[metarepo]\nname = \"test\"\n"), 0o644))

	wg := config.WorkgroupConfig{
		Repos:  []string{"svc-a", "svc-b"},
		Branch: "fix-bug",
	}

	s.Require().NoError(config.SaveLocalWorkgroup(s.cfgPath, "fix-bug", wg))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	got, ok := cfg.Workgroups["fix-bug"]
	s.Require().True(ok)
	s.Assert().Equal(wg.Repos, got.Repos)
	s.Assert().Equal(wg.Branch, got.Branch)
}

func (s *LoaderSuite) TestSaveLocal_PreservesWorkgroups() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte("[metarepo]\nname = \"test\"\n"), 0o644))

	wg := config.WorkgroupConfig{Repos: []string{"svc-a"}, Branch: "feat"}
	s.Require().NoError(config.SaveLocalWorkgroup(s.cfgPath, "feat", wg))

	s.Require().NoError(config.SaveLocal(s.cfgPath, config.ContextConfig{Active: "grp"}))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().Equal("grp", cfg.Context.Active)
	s.Require().Contains(cfg.Workgroups, "feat")
}

func (s *LoaderSuite) TestRemoveLocalWorkgroup() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte("[metarepo]\nname = \"test\"\n"), 0o644))

	s.Require().NoError(config.SaveLocalWorkgroup(s.cfgPath, "feat-a", config.WorkgroupConfig{Repos: []string{"svc-a"}, Branch: "feat-a"}))
	s.Require().NoError(config.SaveLocalWorkgroup(s.cfgPath, "feat-b", config.WorkgroupConfig{Repos: []string{"svc-b"}, Branch: "feat-b"}))

	s.Require().NoError(config.RemoveLocalWorkgroup(s.cfgPath, "feat-a"))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().NotContains(cfg.Workgroups, "feat-a")
	s.Assert().Contains(cfg.Workgroups, "feat-b")
}

func (s *LoaderSuite) TestInitializesWorkgroupsMap() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte("[metarepo]\nname = \"empty\"\n"), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().NotNil(cfg.Workgroups)
}

func (s *LoaderSuite) TestWorkspacesBlocksParse() {
	content := `[metarepo]
name = "myws"

[[workspace]]
name = "payments"
description = "Payment services"
repos = ["api-service", "gateway"]

[[workspace]]
name = "infra"
repos = ["k8s-config"]
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Require().Len(cfg.Workspaces, 2)
	s.Assert().Equal("payments", cfg.Workspaces[0].Name)
	s.Assert().Equal("Payment services", cfg.Workspaces[0].Description)
	s.Assert().Equal([]string{"api-service", "gateway"}, cfg.Workspaces[0].Repos)
	s.Assert().Equal("infra", cfg.Workspaces[1].Name)
	s.Assert().Equal([]string{"k8s-config"}, cfg.Workspaces[1].Repos)
}
