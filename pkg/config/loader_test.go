package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

[[repo]]
name = "frontend"
path = "apps/frontend"
clone_url = "https://github.com/org/frontend"

[[repo]]
name = "backend"
path = "services/backend"

[groups.web]
repos = ["frontend", "backend"]
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().Equal("myws", cfg.Metarepo.Name)
	s.Assert().Equal("apps/frontend", cfg.Repos["frontend"].Path)
	s.Assert().Equal("https://github.com/org/frontend", cfg.Repos["frontend"].CloneURL)
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
	s.Assert().Equal("https://github.com/org/infra", cfg.Repos["infra-dev"].CloneURL)

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
	s.Assert().NotContains(text, `name = "infra-dev"`)
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

[[repo]]
name = "infra-dev"
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

[[repo]]
name = "bad"
path = "/tmp/repo"
`,
			wantErr: "path must be relative",
		},
		{
			name: "empty repo path",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "bad"
path = ""
`,
			wantErr: "path is empty",
		},
		{
			name: "path escapes workspace root",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "bad"
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

func (s *LoaderSuite) TestAgenticFrameworksValidation() {
	tests := []struct {
		name        string
		toml        string
		wantErr     bool
		errContains string
		wantFWs     []string
	}{
		{
			name:    "known value gsd",
			toml:    "[metarepo]\nname = \"ws\"\nagentic_frameworks = [\"gsd\"]\n",
			wantFWs: []string{"gsd"},
		},
		{
			name:        "unknown value",
			toml:        "[metarepo]\nname = \"ws\"\nagentic_frameworks = [\"speckit\"]\n",
			wantErr:     true,
			errContains: "speckit",
		},
		{
			name:    "missing field defaults to gsd",
			toml:    "[metarepo]\nname = \"ws\"\n",
			wantFWs: []string{"gsd"},
		},
		{
			name:    "multi-value known",
			toml:    "[metarepo]\nname = \"ws\"\nagentic_frameworks = [\"gsd\", \"gsd\"]\n",
			wantFWs: []string{"gsd", "gsd"},
		},
		{
			name:        "multi-value with unknown",
			toml:        "[metarepo]\nname = \"ws\"\nagentic_frameworks = [\"gsd\", \"badvalue\"]\n",
			wantErr:     true,
			errContains: "badvalue",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			cfg, err := config.Load(cfgPath)
			if tt.wantErr {
				s.Require().Error(err)
				if tt.errContains != "" {
					s.Assert().Contains(err.Error(), tt.errContains)
				}
				return
			}
			s.Require().NoError(err)
			if tt.wantFWs != nil {
				s.Assert().Equal(tt.wantFWs, cfg.Metarepo.AgenticFrameworks)
			}
		})
	}
}

func (s *LoaderSuite) TestRepoArrayOfTablesFormat() {
	tests := []struct {
		name    string
		toml    string
		wantErr string
		check   func(*LoaderSuite, *config.WorkspaceConfig)
	}{
		{
			name: "valid [[repo]] entries load into map",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "api-service"
path = "repos/api-service"
clone_url = "https://github.com/org/api"

[[repo]]
name = "gateway"
path = "repos/gateway"
`,
			check: func(s *LoaderSuite, cfg *config.WorkspaceConfig) {
				s.Assert().Equal("repos/api-service", cfg.Repos["api-service"].Path)
				s.Assert().Equal("https://github.com/org/api", cfg.Repos["api-service"].CloneURL)
				s.Assert().Equal("repos/gateway", cfg.Repos["gateway"].Path)
			},
		},
		{
			name: "missing name field produces error",
			toml: `[metarepo]
name = "ws"

[[repo]]
path = "repos/no-name"
`,
			wantErr: "missing required name field",
		},
		{
			name: "duplicate name produces error",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "api"
path = "repos/api"

[[repo]]
name = "api"
path = "repos/api2"
`,
			wantErr: "duplicate [[repo]] name",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			cfg, err := config.Load(cfgPath)
			if tt.wantErr != "" {
				s.Require().Error(err)
				s.Assert().Contains(err.Error(), tt.wantErr)
				return
			}
			s.Require().NoError(err)
			if tt.check != nil {
				tt.check(s, cfg)
			}
		})
	}
}

func (s *LoaderSuite) TestRepoByName() {
	toml := `[metarepo]
name = "ws"

[[repo]]
name = "frontend"
path = "apps/frontend"
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(toml), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	rc, ok := cfg.RepoByName("frontend")
	s.Assert().True(ok)
	s.Assert().Equal("apps/frontend", rc.Path)

	_, ok = cfg.RepoByName("missing")
	s.Assert().False(ok)
}

func (s *LoaderSuite) TestSaveRoundTripsRepoList() {
	toml := `[metarepo]
name = "ws"

[[repo]]
name = "api"
path = "repos/api"

[[repo]]
name = "frontend"
path = "repos/frontend"
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(toml), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)
	s.Require().NoError(config.Save(s.cfgPath, cfg))

	data, err := os.ReadFile(s.cfgPath)
	s.Require().NoError(err)
	text := string(data)

	s.Assert().Contains(text, "[[repo]]")
	s.Assert().Contains(text, `name = "api"`)
	s.Assert().Contains(text, `name = "frontend"`)
	s.Assert().NotContains(text, "[repos.")
}

func (s *LoaderSuite) TestFullV2ConfigLoad() {
	content := `[metarepo]
name = "platform-work"
default_remotes = ["origin"]
agentic_frameworks = ["gsd"]

[[workspace]]
name = "payments-platform"
description = "Payment processing and related services"
repos = ["api-service", "payment-lib"]

[[workspace]]
name = "platform-infra"
repos = ["infra-dev", "infra-test"]
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().Equal("platform-work", cfg.Metarepo.Name)
	s.Assert().Equal([]string{"origin"}, cfg.Metarepo.DefaultRemotes)
	s.Assert().Equal([]string{"gsd"}, cfg.Metarepo.AgenticFrameworks)
	s.Require().Len(cfg.Workspaces, 2)
	s.Assert().Equal("payments-platform", cfg.Workspaces[0].Name)
	s.Assert().Equal("Payment processing and related services", cfg.Workspaces[0].Description)
	s.Assert().Equal([]string{"api-service", "payment-lib"}, cfg.Workspaces[0].Repos)
	s.Assert().Equal("platform-infra", cfg.Workspaces[1].Name)
}

func (s *LoaderSuite) TestAliasFieldValidation() {
	tests := []struct {
		name      string
		content   string
		wantErr   string
		wantNoErr bool
	}{
		{
			name: "both fields set is valid",
			content: `[metarepo]
name = "ws"

[[repo]]
name = "svc-a"
path = "svc-a"
track_branch = "main"
upstream = "origin"
`,
			wantNoErr: true,
		},
		{
			name: "neither field set is valid",
			content: `[metarepo]
name = "ws"

[[repo]]
name = "svc-a"
path = "svc-a"
`,
			wantNoErr: true,
		},
		{
			name: "track_branch without upstream is invalid",
			content: `[metarepo]
name = "ws"

[[repo]]
name = "svc-a"
path = "svc-a"
track_branch = "main"
`,
			wantErr: `"svc-a": track_branch and upstream must both be set or both be absent`,
		},
		{
			name: "upstream without track_branch is invalid",
			content: `[metarepo]
name = "ws"

[[repo]]
name = "svc-a"
path = "svc-a"
upstream = "origin"
`,
			wantErr: `"svc-a": track_branch and upstream must both be set or both be absent`,
		},
		{
			name: "duplicate track_branch in same upstream group is invalid",
			content: `[metarepo]
name = "ws"

[[repo]]
name = "svc-a"
path = "svc-a"
track_branch = "main"
upstream = "origin"

[[repo]]
name = "svc-b"
path = "svc-b"
track_branch = "main"
upstream = "origin"
`,
			wantErr: `track_branch "main" already used`,
		},
		{
			name: "same track_branch in different upstream groups is valid",
			content: `[metarepo]
name = "ws"

[[repo]]
name = "svc-a"
path = "svc-a"
track_branch = "main"
upstream = "origin"

[[repo]]
name = "svc-b"
path = "svc-b"
track_branch = "main"
upstream = "upstream"
`,
			wantNoErr: true,
		},
		{
			name: "multiple repos with different track_branches in same upstream is valid",
			content: `[metarepo]
name = "ws"

[[repo]]
name = "svc-a"
path = "svc-a"
track_branch = "main"
upstream = "origin"

[[repo]]
name = "svc-b"
path = "svc-b"
track_branch = "dev"
upstream = "origin"
`,
			wantNoErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.content), 0o644))

			_, err := config.Load(cfgPath)

			if tt.wantNoErr {
				s.Assert().NoError(err)
				return
			}

			s.Require().Error(err)
			s.Assert().Contains(err.Error(), tt.wantErr)
		})
	}
}

func (s *LoaderSuite) TestAliasFieldsRoundTrip() {
	content := `[metarepo]
name = "ws"

[[repo]]
name = "svc-a"
path = "svc-a"
track_branch = "main"
upstream = "origin"

[[repo]]
name = "svc-b"
path = "svc-b"
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().Equal("main", cfg.Repos["svc-a"].TrackBranch)
	s.Assert().Equal("origin", cfg.Repos["svc-a"].Upstream)
	s.Assert().True(cfg.Repos["svc-a"].IsAlias())
	s.Assert().Equal("", cfg.Repos["svc-b"].TrackBranch)
	s.Assert().Equal("", cfg.Repos["svc-b"].Upstream)
	s.Assert().False(cfg.Repos["svc-b"].IsAlias())
}

func (s *LoaderSuite) TestPathConventionWarnings() {
	tests := []struct {
		name         string
		toml         string
		wantWarnings int
		wantContains []string
	}{
		{
			name: "conforming repos/x",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "api"
path = "repos/api"
`,
			wantWarnings: 0,
		},
		{
			name: "conforming with dot-slash",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "api"
path = "./repos/api"
`,
			wantWarnings: 0,
		},
		{
			name: "conforming with trailing slash",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "api"
path = "repos/api/"
`,
			wantWarnings: 0,
		},
		{
			name: "non-conforming apps/frontend",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "frontend"
path = "apps/frontend"
`,
			wantWarnings: 1,
			wantContains: []string{"apps/frontend", "repos/frontend", "git w migrate"},
		},
		{
			name: "non-conforming three segments repos/org/repo",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "my-repo"
path = "repos/org/my-repo"
`,
			wantWarnings: 1,
			wantContains: []string{"repos/org/my-repo", "repos/my-repo"},
		},
		{
			name: "non-conforming bare name",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "api"
path = "my-repo"
`,
			wantWarnings: 1,
			wantContains: []string{"repos/my-repo"},
		},
		{
			name: "multiple repos mixed",
			toml: `[metarepo]
name = "ws"

[[repo]]
name = "good"
path = "repos/good"

[[repo]]
name = "bad"
path = "apps/bad"
`,
			wantWarnings: 1,
			wantContains: []string{"apps/bad"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			cfg, err := config.Load(cfgPath)
			s.Require().NoError(err)
			s.Assert().Len(cfg.Warnings, tt.wantWarnings)

			for _, want := range tt.wantContains {
				found := false
				for _, w := range cfg.Warnings {
					if strings.Contains(w, want) {
						found = true
						break
					}
				}
				s.Assert().Truef(found, "expected warning containing %q, got: %v", want, cfg.Warnings)
			}
		})
	}
}

func (s *LoaderSuite) TestPathConventionWarnings_SkipsSynthesizedRepos() {
	content := `[metarepo]
name = "ws"

[worktrees.infra]
url = "https://github.com/org/infra"
bare_path = "infra/.bare"

[worktrees.infra.branches]
dev = "infra/dev"
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().Empty(cfg.Warnings, "synthesized worktree repos should not produce path warnings")
}

func (s *LoaderSuite) TestRemoteBlocksParse() {
	trueVal := true

	tests := []struct {
		name  string
		toml  string
		check func(s *LoaderSuite, cfg *config.WorkspaceConfig)
		noErr bool
	}{
		{
			name:  "no remote blocks",
			toml:  "[metarepo]\nname = \"ws\"\n",
			noErr: true,
			check: func(s *LoaderSuite, cfg *config.WorkspaceConfig) {
				s.Assert().Empty(cfg.Remotes)
			},
		},
		{
			name: "single remote no branch rules",
			toml: `[metarepo]
name = "ws"

[[remote]]
name       = "origin"
kind       = "github"
direction  = "both"
push_mode  = "branch"
critical   = true
`,
			noErr: true,
			check: func(s *LoaderSuite, cfg *config.WorkspaceConfig) {
				s.Require().Len(cfg.Remotes, 1)
				s.Assert().Equal("origin", cfg.Remotes[0].Name)
				s.Assert().Equal("github", cfg.Remotes[0].Kind)
				s.Assert().Equal("both", cfg.Remotes[0].Direction)
				s.Assert().Equal("branch", cfg.Remotes[0].PushMode)
				s.Assert().True(cfg.Remotes[0].Critical)
				s.Assert().Empty(cfg.Remotes[0].BranchRules)
			},
		},
		{
			name: "single remote with branch rules",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"
kind = "github"

[[remote.branch_rule]]
pattern = "wip/*"
action  = "block"
reason  = "WIP branches must not be pushed to org"

[[remote.branch_rule]]
pattern = "feature/**"
action  = "warn"
`,
			noErr: true,
			check: func(s *LoaderSuite, cfg *config.WorkspaceConfig) {
				s.Require().Len(cfg.Remotes, 1)
				s.Require().Len(cfg.Remotes[0].BranchRules, 2)
				s.Assert().Equal("wip/*", cfg.Remotes[0].BranchRules[0].Pattern)
				s.Assert().Equal(config.ActionBlock, cfg.Remotes[0].BranchRules[0].Action)
				s.Assert().Equal("WIP branches must not be pushed to org", cfg.Remotes[0].BranchRules[0].Reason)
				s.Assert().Equal("feature/**", cfg.Remotes[0].BranchRules[1].Pattern)
				s.Assert().Equal(config.ActionWarn, cfg.Remotes[0].BranchRules[1].Action)
			},
		},
		{
			name: "multiple remotes",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"
kind = "github"

[[remote]]
name = "personal"
kind = "gitea"
`,
			noErr: true,
			check: func(s *LoaderSuite, cfg *config.WorkspaceConfig) {
				s.Require().Len(cfg.Remotes, 2)
				s.Assert().Equal("origin", cfg.Remotes[0].Name)
				s.Assert().Equal("personal", cfg.Remotes[1].Name)
			},
		},
		{
			name: "branch rule *bool fields",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"
kind = "github"

[[remote.branch_rule]]
pattern   = "wip/*"
untracked = true
action    = "block"

[[remote.branch_rule]]
pattern = "main"
action  = "allow"
`,
			noErr: true,
			check: func(s *LoaderSuite, cfg *config.WorkspaceConfig) {
				s.Require().Len(cfg.Remotes[0].BranchRules, 2)
				s.Require().NotNil(cfg.Remotes[0].BranchRules[0].Untracked)
				s.Assert().True(*cfg.Remotes[0].BranchRules[0].Untracked)
				s.Assert().Nil(cfg.Remotes[0].BranchRules[1].Untracked)
			},
		},
	}

	_ = trueVal

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			cfg, err := config.Load(cfgPath)
			if !tt.noErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			if tt.check != nil {
				tt.check(s, cfg)
			}
		})
	}
}

func (s *LoaderSuite) TestRemoteRoundTrip() {
	content := `[metarepo]
name = "ws"

[[remote]]
name      = "origin"
kind      = "github"
direction = "both"
push_mode = "branch"
critical  = true

[[remote.branch_rule]]
pattern = "wip/*"
action  = "block"
reason  = "WIP branches must not be pushed to org"

[[remote.branch_rule]]
pattern = "**"
action  = "allow"

[[remote]]
name = "personal"
kind = "gitea"
url  = "https://gitea.example.com"
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)
	s.Require().Len(cfg.Remotes, 2)

	s.Require().NoError(config.Save(s.cfgPath, cfg))

	cfg2, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Require().Len(cfg2.Remotes, 2)
	s.Assert().Equal(cfg.Remotes[0].Name, cfg2.Remotes[0].Name)
	s.Assert().Equal(cfg.Remotes[0].Kind, cfg2.Remotes[0].Kind)
	s.Assert().Equal(cfg.Remotes[0].Direction, cfg2.Remotes[0].Direction)
	s.Assert().Equal(cfg.Remotes[0].PushMode, cfg2.Remotes[0].PushMode)
	s.Assert().Equal(cfg.Remotes[0].Critical, cfg2.Remotes[0].Critical)
	s.Require().Len(cfg2.Remotes[0].BranchRules, 2)
	s.Assert().Equal(cfg.Remotes[0].BranchRules[0].Pattern, cfg2.Remotes[0].BranchRules[0].Pattern)
	s.Assert().Equal(cfg.Remotes[0].BranchRules[0].Action, cfg2.Remotes[0].BranchRules[0].Action)
	s.Assert().Equal(cfg.Remotes[0].BranchRules[1].Pattern, cfg2.Remotes[0].BranchRules[1].Pattern)
	s.Assert().Equal(cfg.Remotes[1].Name, cfg2.Remotes[1].Name)
	s.Assert().Equal(cfg.Remotes[1].Kind, cfg2.Remotes[1].Kind)
	s.Assert().Equal(cfg.Remotes[1].URL, cfg2.Remotes[1].URL)
}

func (s *LoaderSuite) TestSyncPairBlocksParse() {
	tests := []struct {
		name  string
		toml  string
		check func(s *LoaderSuite, cfg *config.WorkspaceConfig)
	}{
		{
			name: "no sync_pair blocks",
			toml: "[metarepo]\nname = \"ws\"\n",
			check: func(s *LoaderSuite, cfg *config.WorkspaceConfig) {
				s.Assert().Empty(cfg.SyncPairs)
			},
		},
		{
			name: "single pair no refs",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"
`,
			check: func(s *LoaderSuite, cfg *config.WorkspaceConfig) {
				s.Require().Len(cfg.SyncPairs, 1)
				s.Assert().Equal("origin", cfg.SyncPairs[0].From)
				s.Assert().Equal("personal", cfg.SyncPairs[0].To)
				s.Assert().Empty(cfg.SyncPairs[0].Refs)
			},
		},
		{
			name: "single pair with refs",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"
refs = ["**"]
`,
			check: func(s *LoaderSuite, cfg *config.WorkspaceConfig) {
				s.Require().Len(cfg.SyncPairs, 1)
				s.Assert().Equal([]string{"**"}, cfg.SyncPairs[0].Refs)
			},
		},
		{
			name: "multiple pairs",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"

[[sync_pair]]
from = "personal"
to   = "contractor"
`,
			check: func(s *LoaderSuite, cfg *config.WorkspaceConfig) {
				s.Require().Len(cfg.SyncPairs, 2)
				s.Assert().Equal("origin", cfg.SyncPairs[0].From)
				s.Assert().Equal("personal", cfg.SyncPairs[0].To)
				s.Assert().Equal("personal", cfg.SyncPairs[1].From)
				s.Assert().Equal("contractor", cfg.SyncPairs[1].To)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			cfg, err := config.Load(cfgPath)
			s.Require().NoError(err)
			if tt.check != nil {
				tt.check(s, cfg)
			}
		})
	}
}

func (s *LoaderSuite) TestSyncPairRoundTrip() {
	content := `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"
refs = ["main", "develop"]

[[sync_pair]]
from = "personal"
to   = "contractor"
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)
	s.Require().Len(cfg.SyncPairs, 2)

	s.Require().NoError(config.Save(s.cfgPath, cfg))

	cfg2, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Require().Len(cfg2.SyncPairs, 2)
	s.Assert().Equal(cfg.SyncPairs[0].From, cfg2.SyncPairs[0].From)
	s.Assert().Equal(cfg.SyncPairs[0].To, cfg2.SyncPairs[0].To)
	s.Assert().Equal(cfg.SyncPairs[0].Refs, cfg2.SyncPairs[0].Refs)
	s.Assert().Equal(cfg.SyncPairs[1].From, cfg2.SyncPairs[1].From)
	s.Assert().Equal(cfg.SyncPairs[1].To, cfg2.SyncPairs[1].To)
}

func (s *LoaderSuite) TestSyncPairValidation() {
	tests := []struct {
		name        string
		toml        string
		wantErr     bool
		errContains []string
	}{
		{
			name: "valid pair no refs",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"
`,
		},
		{
			name: "valid pair with refs",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"
refs = ["**"]
`,
		},
		{
			name: "missing from",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
to = "personal"
`,
			wantErr:     true,
			errContains: []string{"missing required", "from"},
		},
		{
			name: "missing to",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
`,
			wantErr:     true,
			errContains: []string{"missing required", "to"},
		},
		{
			name: "duplicate pair",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"

[[sync_pair]]
from = "origin"
to   = "personal"
`,
			wantErr:     true,
			errContains: []string{"duplicate", "origin", "personal"},
		},
		{
			name: "same from different to ok",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"

[[sync_pair]]
from = "origin"
to   = "contractor"
`,
		},
		{
			name: "same to different from ok",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"

[[sync_pair]]
from = "personal"
to   = "contractor"
`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			_, err := config.Load(cfgPath)
			if tt.wantErr {
				s.Require().Error(err)
				for _, want := range tt.errContains {
					s.Assert().Contains(err.Error(), want)
				}
				return
			}
			s.Require().NoError(err)
		})
	}
}

func (s *LoaderSuite) TestSyncCycleDetection() {
	tests := []struct {
		name        string
		toml        string
		wantErr     bool
		errContains []string
	}{
		{
			name: "no pairs no cycle",
			toml: "[metarepo]\nname = \"ws\"\n",
		},
		{
			name: "linear chain no cycle",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"

[[sync_pair]]
from = "personal"
to   = "contractor"
`,
		},
		{
			name: "two-node cycle",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"

[[sync_pair]]
from = "personal"
to   = "origin"
`,
			wantErr:     true,
			errContains: []string{"sync_pair cycle detected", "origin", "personal"},
		},
		{
			name: "three-node cycle",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"

[[sync_pair]]
from = "personal"
to   = "contractor"

[[sync_pair]]
from = "contractor"
to   = "origin"
`,
			wantErr:     true,
			errContains: []string{"sync_pair cycle detected", "origin", "personal", "contractor", "→ origin"},
		},
		{
			name: "self-loop",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "origin"
`,
			wantErr:     true,
			errContains: []string{"sync_pair cycle detected", "origin"},
		},
		{
			name: "cycle in longer chain",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "a"
to   = "b"

[[sync_pair]]
from = "b"
to   = "c"

[[sync_pair]]
from = "c"
to   = "b"
`,
			wantErr:     true,
			errContains: []string{"sync_pair cycle detected", "b", "c"},
		},
		{
			name: "diamond no cycle",
			toml: `[metarepo]
name = "ws"

[[sync_pair]]
from = "origin"
to   = "personal"

[[sync_pair]]
from = "origin"
to   = "contractor"

[[sync_pair]]
from = "personal"
to   = "mirror"

[[sync_pair]]
from = "contractor"
to   = "mirror"
`,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			_, err := config.Load(cfgPath)
			if tt.wantErr {
				s.Require().Error(err)
				for _, want := range tt.errContains {
					s.Assert().Contains(err.Error(), want)
				}
				return
			}
			s.Require().NoError(err)
		})
	}
}

func (s *LoaderSuite) TestRemoteValidation() {
	tests := []struct {
		name        string
		toml        string
		cfgSuffix   string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid remote no branch rules",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"
kind = "github"
`,
			cfgSuffix: ".gitw",
		},
		{
			name: "valid remote with branch rules",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"
kind = "gitea"

[[remote.branch_rule]]
pattern = "main"
action  = "allow"
`,
			cfgSuffix: ".gitw",
		},
		{
			name: "missing name",
			toml: `[metarepo]
name = "ws"

[[remote]]
kind = "github"
`,
			cfgSuffix:   ".gitw",
			wantErr:     true,
			errContains: "missing required name",
		},
		{
			name: "duplicate name",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"
kind = "github"

[[remote]]
name = "origin"
kind = "gitea"
`,
			cfgSuffix:   ".gitw",
			wantErr:     true,
			errContains: "duplicate",
		},
		{
			name: "invalid kind",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"
kind = "bitbucket"
`,
			cfgSuffix:   ".gitw",
			wantErr:     true,
			errContains: "bitbucket",
		},
		{
			name: "invalid action",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"
kind = "github"

[[remote.branch_rule]]
pattern = "main"
action  = "skip"
`,
			cfgSuffix:   ".gitw",
			wantErr:     true,
			errContains: "skip",
		},
		{
			name: "private in public file",
			toml: `[metarepo]
name = "ws"

[[remote]]
name    = "personal"
kind    = "gitea"
private = true
`,
			cfgSuffix:   ".gitw",
			wantErr:     true,
			errContains: ".git/.gitw",
		},
		{
			name: "private in private file ok",
			toml: `[metarepo]
name = "ws"

[[remote]]
name    = "personal"
kind    = "gitea"
private = true
`,
			cfgSuffix: ".git/.gitw",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			dir := s.T().TempDir()

			var cfgPath string
			if tt.cfgSuffix == ".git/.gitw" {
				gitDir := filepath.Join(dir, ".git")
				s.Require().NoError(os.MkdirAll(gitDir, 0o755))
				cfgPath = filepath.Join(gitDir, ".gitw")
			} else {
				cfgPath = filepath.Join(dir, tt.cfgSuffix)
			}

			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			_, err := config.Load(cfgPath)
			if tt.wantErr {
				s.Require().Error(err)
				if tt.errContains != "" {
					s.Assert().Contains(err.Error(), tt.errContains)
				}
				return
			}

			s.Require().NoError(err)
		})
	}
}

func (s *LoaderSuite) TestWorkstreamBlocksParse() {
	tests := []struct {
		name string
		toml string
		want []config.WorkstreamConfig
	}{
		{
			name: "no workstream blocks",
			toml: `[metarepo]
name = "ws"
`,
			want: []config.WorkstreamConfig{},
		},
		{
			name: "single workstream with remotes",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"

[[workstream]]
name = "alpha"
remotes = ["origin"]
`,
			want: []config.WorkstreamConfig{
				{Name: "alpha", Remotes: []string{"origin"}},
			},
		},
		{
			name: "single workstream with empty remotes",
			toml: `[metarepo]
name = "ws"

[[workstream]]
name = "alpha"
remotes = []
`,
			want: []config.WorkstreamConfig{
				{Name: "alpha", Remotes: []string{}},
			},
		},
		{
			name: "multiple workstream blocks",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"

[[remote]]
name = "personal"

[[workstream]]
name = "beta"
remotes = ["personal"]

[[workstream]]
name = "alpha"
remotes = ["origin"]
`,
			want: []config.WorkstreamConfig{
				{Name: "alpha", Remotes: []string{"origin"}},
				{Name: "beta", Remotes: []string{"personal"}},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			cfg, err := config.Load(cfgPath)
			s.Require().NoError(err)
			if len(tt.want) == 0 {
				s.Assert().Empty(cfg.Workstreams)
				return
			}

			s.Assert().Equal(tt.want, cfg.Workstreams)
		})
	}
}

func (s *LoaderSuite) TestWorkstreamValidation() {
	tests := []struct {
		name        string
		toml        string
		errContains []string
	}{
		{
			name: "missing name",
			toml: `[metarepo]
name = "ws"

[[workstream]]
remotes = []
`,
			errContains: []string{"[[workstream]]", "missing required name field"},
		},
		{
			name: "missing remotes key",
			toml: `[metarepo]
name = "ws"

[[workstream]]
name = "alpha"
`,
			errContains: []string{"[[workstream]]", "missing required remotes key"},
		},
		{
			name: "duplicate workstream name",
			toml: `[metarepo]
name = "ws"

[[workstream]]
name = "alpha"
remotes = []

[[workstream]]
name = "alpha"
remotes = []
`,
			errContains: []string{"duplicate [[workstream]] name", "alpha"},
		},
		{
			name: "unknown key",
			toml: `[metarepo]
name = "ws"

[[workstream]]
name = "alpha"
remotes = []
policy = "x"
`,
			errContains: []string{"[[workstream]]", "unknown key", "policy"},
		},
		{
			name: "unknown remote reference",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"

[[workstream]]
name = "alpha"
remotes = ["does-not-exist"]
`,
			errContains: []string{"workstream", "unknown remote", "does-not-exist"},
		},
		{
			name: "duplicate remote in remotes list",
			toml: `[metarepo]
name = "ws"

[[remote]]
name = "origin"

[[workstream]]
name = "alpha"
remotes = ["origin", "origin"]
`,
			errContains: []string{"workstream", "duplicate remote", "origin"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfgPath := filepath.Join(s.T().TempDir(), ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tt.toml), 0o644))

			_, err := config.Load(cfgPath)
			s.Require().Error(err)
			for _, fragment := range tt.errContains {
				s.Assert().Contains(err.Error(), fragment)
			}
		})
	}
}

func (s *LoaderSuite) TestWorkstreamPlacementAllowedInPublicConfig() {
	content := `[metarepo]
name = "ws"

[[remote]]
name = "origin"

[[workstream]]
name = "alpha"
remotes = ["origin"]
`

	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)
	s.Require().Len(cfg.Workstreams, 1)

	for _, warning := range cfg.Warnings {
		s.Assert().NotContains(strings.ToLower(warning), "workstream")
		s.Assert().NotContains(strings.ToLower(warning), "misconfiguration")
	}
}

func (s *LoaderSuite) TestWorkstreamNormalizationOrder() {
	content := `[metarepo]
name = "ws"

[[remote]]
name = "origin"

[[remote]]
name = "backup"

[[remote]]
name = "personal"

[[workstream]]
name = "zeta"
remotes = ["personal", "backup"]

[[workstream]]
name = "alpha"
remotes = ["origin", "backup"]

[[workstream]]
name = "beta"
remotes = ["personal", "origin", "backup"]
`

	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Require().Len(cfg.Workstreams, 3)
	s.Assert().Equal("alpha", cfg.Workstreams[0].Name)
	s.Assert().Equal("beta", cfg.Workstreams[1].Name)
	s.Assert().Equal("zeta", cfg.Workstreams[2].Name)

	s.Assert().Equal([]string{"backup", "origin"}, cfg.Workstreams[0].Remotes)
	s.Assert().Equal([]string{"backup", "origin", "personal"}, cfg.Workstreams[1].Remotes)
	s.Assert().Equal([]string{"backup", "personal"}, cfg.Workstreams[2].Remotes)
}

// privatePath returns the expected .git/.gitw path for the suite's cfgPath.
func (s *LoaderSuite) privatePath() string {
	return filepath.Join(filepath.Dir(s.cfgPath), ".git", ".gitw")
}

// writePrivate creates the .git/ directory and writes content to .git/.gitw.
func (s *LoaderSuite) writePrivate(content string) {
	dir := filepath.Join(filepath.Dir(s.cfgPath), ".git")
	s.Require().NoError(os.MkdirAll(dir, 0o755))
	s.Require().NoError(os.WriteFile(s.privatePath(), []byte(content), 0o644))
}

func (s *LoaderSuite) TestPrivateConfigAbsent() {
	content := `
[[repo]]
name = "api"
path = "repos/api"

[[remote]]
name = "origin"
kind = "github"
`
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(content), 0o644))

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().Len(cfg.Repos, 1)
	s.Assert().Contains(cfg.Repos, "api")
	s.Assert().Len(cfg.Remotes, 1)
	s.Assert().Equal("origin", cfg.Remotes[0].Name)
}

func (s *LoaderSuite) TestPrivateConfigRemoteOverride() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[remote]]
name = "origin"
kind = "github"
url = "https://github.com"
`), 0o644))

	s.writePrivate(`
[[remote]]
name = "origin"
token_env = "GITHUB_TOKEN"
private = true
`)

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Require().Len(cfg.Remotes, 1)
	r := cfg.Remotes[0]
	s.Assert().Equal("origin", r.Name)
	s.Assert().Equal("github", r.Kind)
	s.Assert().Equal("https://github.com", r.URL)
	s.Assert().Equal("GITHUB_TOKEN", r.TokenEnv)
	s.Assert().True(r.Private)
}

func (s *LoaderSuite) TestPrivateConfigNewRemote() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[remote]]
name = "origin"
kind = "github"
`), 0o644))

	s.writePrivate(`
[[remote]]
name = "personal"
kind = "gitea"
url = "https://gitea.example.com"
private = true
`)

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Require().Len(cfg.Remotes, 2)
	names := []string{cfg.Remotes[0].Name, cfg.Remotes[1].Name}
	s.Assert().Contains(names, "origin")
	s.Assert().Contains(names, "personal")
}

func (s *LoaderSuite) TestPrivateConfigRepoOverride() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[repo]]
name = "api-service"
path = "repos/api-service"
`), 0o644))

	s.writePrivate(`
[[repo]]
name = "api-service"
clone_url = "https://github.com/work-org/api-service"
`)

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	r := cfg.Repos["api-service"]
	s.Assert().Equal("repos/api-service", r.Path)
	s.Assert().Equal("https://github.com/work-org/api-service", r.CloneURL)
}

func (s *LoaderSuite) TestPrivateConfigUnknownRepo() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[repo]]
name = "api-service"
path = "repos/api-service"
`), 0o644))

	s.writePrivate(`
[[repo]]
name = "nonexistent"
path = "repos/nonexistent"
`)

	_, err := config.Load(s.cfgPath)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "nonexistent")
	s.Assert().Contains(err.Error(), "not declared in .gitw")
}

func (s *LoaderSuite) TestPrivateConfigMetarepoOverride() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[metarepo]
default_remotes = ["origin"]
`), 0o644))

	s.writePrivate(`
[metarepo]
default_remotes = ["origin", "personal"]
`)

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Assert().Equal([]string{"origin", "personal"}, cfg.Metarepo.DefaultRemotes)
}

func (s *LoaderSuite) TestPrivateConfigWorkstreamOverride() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[remote]]
name = "origin"

[[workstream]]
name = "TICKET-123"
remotes = ["origin"]
`), 0o644))

	s.writePrivate(`
[[remote]]
name = "personal"
private = true

[[workstream]]
name = "TICKET-123"
remotes = ["personal"]
`)

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	var ws *config.WorkstreamConfig
	for i := range cfg.Workstreams {
		if cfg.Workstreams[i].Name == "TICKET-123" {
			ws = &cfg.Workstreams[i]
			break
		}
	}
	s.Require().NotNil(ws)
	s.Assert().Equal([]string{"personal"}, ws.Remotes)
}

func (s *LoaderSuite) TestPrivateConfigNewWorkstream() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[remote]]
name = "origin"
`), 0o644))

	s.writePrivate(`
[[remote]]
name = "personal"
private = true

[[workstream]]
name = "LOCAL-WS"
remotes = ["personal"]
`)

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	found := false
	for _, w := range cfg.Workstreams {
		if w.Name == "LOCAL-WS" {
			found = true
			break
		}
	}
	s.Assert().True(found, "expected workstream LOCAL-WS in merged config")
}

func (s *LoaderSuite) TestPrivateConfigSyncPairOverride() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[remote]]
name = "origin"

[[remote]]
name = "personal"

[[sync_pair]]
from = "origin"
to = "personal"
refs = ["main"]
`), 0o644))

	s.writePrivate(`
[[sync_pair]]
from = "origin"
to = "personal"
refs = ["**"]
`)

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Require().Len(cfg.SyncPairs, 1)
	s.Assert().Equal([]string{"**"}, cfg.SyncPairs[0].Refs)
}

func (s *LoaderSuite) TestPrivateConfigNewSyncPair() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[remote]]
name = "origin"

[[remote]]
name = "personal"
`), 0o644))

	s.writePrivate(`
[[sync_pair]]
from = "origin"
to = "personal"
`)

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	s.Require().Len(cfg.SyncPairs, 1)
	s.Assert().Equal("origin", cfg.SyncPairs[0].From)
	s.Assert().Equal("personal", cfg.SyncPairs[0].To)
}

func (s *LoaderSuite) TestPrivateConfigWorkspaceOverride() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[repo]]
name = "api-service"
path = "repos/api-service"

[[workspace]]
name = "platform"
repos = ["api-service"]
`), 0o644))

	s.writePrivate(`
[[workspace]]
name = "platform"
description = "local override"
`)

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	var ws *config.WorkspaceBlock
	for i := range cfg.Workspaces {
		if cfg.Workspaces[i].Name == "platform" {
			ws = &cfg.Workspaces[i]
			break
		}
	}
	s.Require().NotNil(ws)
	s.Assert().Equal([]string{"api-service"}, ws.Repos)
	s.Assert().Equal("local override", ws.Description)
}

func (s *LoaderSuite) TestPrivateConfigNewWorkspace() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[repo]]
name = "api-service"
path = "repos/api-service"
`), 0o644))

	s.writePrivate(`
[[workspace]]
name = "local-ws"
repos = ["api-service"]
`)

	cfg, err := config.Load(s.cfgPath)
	s.Require().NoError(err)

	found := false
	for _, w := range cfg.Workspaces {
		if w.Name == "local-ws" {
			found = true
			break
		}
	}
	s.Assert().True(found, "expected workspace local-ws in merged config")
}

func (s *LoaderSuite) TestPrivateEnforcementInSharedFile() {
	s.Require().NoError(os.WriteFile(s.cfgPath, []byte(`
[[remote]]
name = "secret"
private = true
`), 0o644))

	_, err := config.Load(s.cfgPath)
	s.Require().Error(err)
	s.Assert().Contains(err.Error(), "secret")
	s.Assert().Contains(err.Error(), ".git/.gitw")
}
