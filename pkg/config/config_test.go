package config_test

import (
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

type branchAccessorCase struct {
	name     string
	meta     config.MetarepoConfig
	wantSync bool
	wantUp   bool
	wantPush bool
}

type resolveDefaultCase struct {
	name        string
	repoDefault string
	workspace   string
	want        string
}

type worktreeBranchCase struct {
	name  string
	repo  string
	want  string
	found bool
}

type worktreeIndexCase struct {
	name      string
	worktrees map[string]config.WorktreeConfig
	want      map[string]string
}

type syncPushEnabledCase struct {
	name string
	meta config.MetarepoConfig
	want bool
}

var syncPushEnabledCases = func() []syncPushEnabledCase {
	trueValue := true
	falseValue := false
	return []syncPushEnabledCase{
		{name: "nil defaults true", meta: config.MetarepoConfig{}, want: true},
		{name: "explicit true", meta: config.MetarepoConfig{SyncPush: &trueValue}, want: true},
		{name: "explicit false", meta: config.MetarepoConfig{SyncPush: &falseValue}, want: false},
	}
}()

func TestConfigSuite(t *testing.T) {
	testutil.RunSuite(t, new(ConfigSuite))
}

func (s *ConfigSuite) TestSyncPushEnabled() {
	for _, tt := range syncPushEnabledCases {
		s.Run(tt.name, func() {
			cfg := config.WorkspaceConfig{Metarepo: tt.meta}
			s.Equal(tt.want, cfg.SyncPushEnabled())
		})
	}
}

func (s *ConfigSuite) TestBranchAccessors() {
	for _, tt := range branchAccessorCases() {
		s.Run(tt.name, func() {
			cfg := config.WorkspaceConfig{Metarepo: tt.meta}
			s.Equal(tt.wantSync, cfg.BranchSyncSourceEnabled())
			s.Equal(tt.wantUp, cfg.BranchSetUpstreamEnabled())
			s.Equal(tt.wantPush, cfg.BranchPushEnabled())
		})
	}
}

func (s *ConfigSuite) TestResolveDefaultBranch() {
	for _, tt := range resolveDefaultCases() {
		s.Run(tt.name, func() {
			cfg := config.WorkspaceConfig{
				Metarepo: config.MetarepoConfig{DefaultBranch: tt.workspace},
				Repos:    map[string]config.RepoConfig{"frontend": {DefaultBranch: tt.repoDefault}},
			}
			s.Equal(tt.want, cfg.ResolveDefaultBranch("frontend"))
		})
	}
}

func (s *ConfigSuite) TestResolveDefaultBranchWorktreeRepo() {
	cfg := worktreeBranchConfig()
	cfg.Metarepo.DefaultBranch = "main"

	// Worktree repos must return their own branch, not the workspace default.
	s.Equal("dev", cfg.ResolveDefaultBranch("infra-dev"))
	s.Equal("prod", cfg.ResolveDefaultBranch("infra-prod"))
	// Plain repo names fall through to workspace default.
	s.Equal("main", cfg.ResolveDefaultBranch("backend"))
}

func (s *ConfigSuite) TestWorktreeBranchForRepo() {
	cfg := worktreeBranchConfig()

	for _, tt := range worktreeBranchCases() {
		s.Run(tt.name, func() {
			branch, ok := cfg.WorktreeBranchForRepo(tt.repo)
			s.Equal(tt.found, ok)
			s.Equal(tt.want, branch)
		})
	}
}

func (s *ConfigSuite) TestWorktreeRepoToSetIndex() {
	for _, tt := range worktreeIndexCases() {
		s.Run(tt.name, func() {
			cfg := config.WorkspaceConfig{Worktrees: tt.worktrees}
			s.Equal(tt.want, config.WorktreeRepoToSetIndex(&cfg))
		})
	}
}

func (s *ConfigSuite) TestWorkspaceBlockFields() {
	wb := config.WorkspaceBlock{
		Name:        "payments",
		Description: "Payment processing",
		Repos:       []string{"api", "gateway"},
	}
	s.Equal("payments", wb.Name)
	s.Equal("Payment processing", wb.Description)
	s.Equal([]string{"api", "gateway"}, wb.Repos)
}

func branchAccessorCases() []branchAccessorCase {
	trueValue := true
	falseValue := false

	return []branchAccessorCase{
		{name: "nil defaults true", meta: config.MetarepoConfig{}, wantSync: true, wantUp: true, wantPush: true},
		{name: "explicit false", meta: config.MetarepoConfig{BranchSyncSource: &falseValue, BranchSetUpstream: &falseValue, BranchPush: &falseValue}, wantSync: false, wantUp: false, wantPush: false},
		{name: "explicit true", meta: config.MetarepoConfig{BranchSyncSource: &trueValue, BranchSetUpstream: &trueValue, BranchPush: &trueValue}, wantSync: true, wantUp: true, wantPush: true},
	}
}

func resolveDefaultCases() []resolveDefaultCase {
	return []resolveDefaultCase{
		{name: "per-repo override wins", repoDefault: "develop", workspace: "staging", want: "develop"},
		{name: "workspace fallback", repoDefault: "", workspace: "trunk", want: "trunk"},
		{name: "hardcoded fallback", repoDefault: "", workspace: "", want: "main"},
		{name: "empty repo falls through", repoDefault: "", workspace: "trunk", want: "trunk"},
	}
}

func worktreeBranchConfig() config.WorkspaceConfig {
	return config.WorkspaceConfig{
		Worktrees: map[string]config.WorktreeConfig{
			"infra": {Branches: map[string]string{"dev": "infra/dev", "prod": "infra/prod"}},
		},
	}
}

func worktreeBranchCases() []worktreeBranchCase {
	return []worktreeBranchCase{
		{name: "found dev", repo: "infra-dev", want: "dev", found: true},
		{name: "found prod", repo: "infra-prod", want: "prod", found: true},
		{name: "not a worktree repo", repo: "backend", want: "", found: false},
		{name: "unknown name", repo: "xyz", want: "", found: false},
	}
}

func worktreeIndexCases() []worktreeIndexCase {
	return []worktreeIndexCase{
		{name: "empty", worktrees: map[string]config.WorktreeConfig{}, want: map[string]string{}},
		{
			name: "one set two branches",
			worktrees: map[string]config.WorktreeConfig{
				"infra": {Branches: map[string]string{"dev": "infra/dev", "test": "infra/test"}},
			},
			want: map[string]string{"infra-dev": "infra", "infra-test": "infra"},
		},
		{
			name: "two sets",
			worktrees: map[string]config.WorktreeConfig{
				"infra": {Branches: map[string]string{"dev": "infra/dev", "test": "infra/test"}},
				"ops":   {Branches: map[string]string{"dev": "ops/dev", "test": "ops/test"}},
			},
			want: map[string]string{"infra-dev": "infra", "infra-test": "infra", "ops-dev": "ops", "ops-test": "ops"},
		},
	}
}

type isAliasCase struct {
	name        string
	trackBranch string
	want        bool
}

func (s *ConfigSuite) TestRepoConfigIsAlias() {
	cases := []isAliasCase{
		{name: "empty track_branch", trackBranch: "", want: false},
		{name: "non-empty track_branch", trackBranch: "dev", want: true},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			rc := config.RepoConfig{TrackBranch: tc.trackBranch}
			s.Assert().Equal(tc.want, rc.IsAlias())
		})
	}
}

func (s *ConfigSuite) TestBranchActionConstants() {
	s.Assert().Equal(config.BranchAction("allow"), config.ActionAllow)
	s.Assert().Equal(config.BranchAction("block"), config.ActionBlock)
	s.Assert().Equal(config.BranchAction("warn"), config.ActionWarn)
	s.Assert().Equal(config.BranchAction("require-flag"), config.ActionRequireFlag)
}

func (s *ConfigSuite) TestMergeRemote() {
	trueVal := true

	cases := []struct {
		name     string
		base     config.RemoteConfig
		override config.RemoteConfig
		want     config.RemoteConfig
	}{
		{
			name:     "empty override returns base unchanged",
			base:     config.RemoteConfig{Name: "origin", Kind: "github", URL: "https://github.com"},
			override: config.RemoteConfig{},
			want:     config.RemoteConfig{Name: "origin", Kind: "github", URL: "https://github.com"},
		},
		{
			name:     "non-zero name in override wins",
			base:     config.RemoteConfig{Name: "origin"},
			override: config.RemoteConfig{Name: "personal"},
			want:     config.RemoteConfig{Name: "personal"},
		},
		{
			name:     "zero URL keeps base URL",
			base:     config.RemoteConfig{Name: "origin", URL: "https://github.com"},
			override: config.RemoteConfig{Name: "origin"},
			want:     config.RemoteConfig{Name: "origin", URL: "https://github.com"},
		},
		{
			name:     "non-zero URL in override wins",
			base:     config.RemoteConfig{Name: "origin", URL: "https://github.com"},
			override: config.RemoteConfig{URL: "https://gitea.example.com"},
			want:     config.RemoteConfig{Name: "origin", URL: "https://gitea.example.com"},
		},
		{
			name:     "non-zero kind wins; empty kind keeps base kind",
			base:     config.RemoteConfig{Kind: "github"},
			override: config.RemoteConfig{Kind: "gitea"},
			want:     config.RemoteConfig{Kind: "gitea"},
		},
		{
			name:     "Critical true in override wins",
			base:     config.RemoteConfig{Critical: false},
			override: config.RemoteConfig{Critical: true},
			want:     config.RemoteConfig{Critical: true},
		},
		{
			name:     "Private true in override wins",
			base:     config.RemoteConfig{Private: false},
			override: config.RemoteConfig{Private: true},
			want:     config.RemoteConfig{Private: true},
		},
		{
			name: "nil BranchRules override keeps base BranchRules",
			base: config.RemoteConfig{
				BranchRules: []config.BranchRuleConfig{{Pattern: "main", Action: config.ActionAllow}},
			},
			override: config.RemoteConfig{},
			want: config.RemoteConfig{
				BranchRules: []config.BranchRuleConfig{{Pattern: "main", Action: config.ActionAllow}},
			},
		},
		{
			name: "non-nil BranchRules in override replaces base",
			base: config.RemoteConfig{
				BranchRules: []config.BranchRuleConfig{{Pattern: "old/*", Action: config.ActionBlock}},
			},
			override: config.RemoteConfig{
				BranchRules: []config.BranchRuleConfig{{Pattern: "new/*", Action: config.ActionWarn}},
			},
			want: config.RemoteConfig{
				BranchRules: []config.BranchRuleConfig{{Pattern: "new/*", Action: config.ActionWarn}},
			},
		},
		{
			name: "BranchRuleConfig *bool fields preserved",
			base: config.RemoteConfig{
				BranchRules: []config.BranchRuleConfig{{Untracked: &trueVal}},
			},
			override: config.RemoteConfig{},
			want: config.RemoteConfig{
				BranchRules: []config.BranchRuleConfig{{Untracked: &trueVal}},
			},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := config.MergeRemote(tc.base, tc.override)
			s.Assert().Equal(tc.want, got)
		})
	}
}

func (s *ConfigSuite) TestMergeSyncPair() {
	cases := []struct {
		name     string
		base     config.SyncPairConfig
		override config.SyncPairConfig
		want     config.SyncPairConfig
	}{
		{
			name:     "empty override returns base unchanged",
			base:     config.SyncPairConfig{From: "origin", To: "personal", Refs: []string{"**"}},
			override: config.SyncPairConfig{},
			want:     config.SyncPairConfig{From: "origin", To: "personal", Refs: []string{"**"}},
		},
		{
			name:     "non-zero From in override wins",
			base:     config.SyncPairConfig{From: "origin"},
			override: config.SyncPairConfig{From: "personal"},
			want:     config.SyncPairConfig{From: "personal"},
		},
		{
			name:     "non-zero To in override wins",
			base:     config.SyncPairConfig{To: "personal"},
			override: config.SyncPairConfig{To: "contractor"},
			want:     config.SyncPairConfig{To: "contractor"},
		},
		{
			name:     "non-empty Refs in override wins",
			base:     config.SyncPairConfig{Refs: []string{"main"}},
			override: config.SyncPairConfig{Refs: []string{"release/**"}},
			want:     config.SyncPairConfig{Refs: []string{"release/**"}},
		},
		{
			name:     "nil override Refs keeps base Refs",
			base:     config.SyncPairConfig{Refs: []string{"main", "develop"}},
			override: config.SyncPairConfig{Refs: nil},
			want:     config.SyncPairConfig{Refs: []string{"main", "develop"}},
		},
		{
			name:     "empty slice override Refs keeps base Refs",
			base:     config.SyncPairConfig{Refs: []string{"main"}},
			override: config.SyncPairConfig{Refs: []string{}},
			want:     config.SyncPairConfig{Refs: []string{"main"}},
		},
		{
			name:     "zero From keeps base From",
			base:     config.SyncPairConfig{From: "origin"},
			override: config.SyncPairConfig{From: ""},
			want:     config.SyncPairConfig{From: "origin"},
		},
		{
			name:     "zero To keeps base To",
			base:     config.SyncPairConfig{To: "personal"},
			override: config.SyncPairConfig{To: ""},
			want:     config.SyncPairConfig{To: "personal"},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := config.MergeSyncPair(tc.base, tc.override)
			s.Assert().Equal(tc.want, got)
		})
	}
}

func (s *ConfigSuite) TestMergeWorkstream() {
	cases := []struct {
		name     string
		base     config.WorkstreamConfig
		override config.WorkstreamConfig
		want     config.WorkstreamConfig
	}{
		{
			name:     "empty override keeps base",
			base:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{"origin", "mirror"}},
			override: config.WorkstreamConfig{},
			want:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{"origin", "mirror"}},
		},
		{
			name:     "name override replaces base name",
			base:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{"origin"}},
			override: config.WorkstreamConfig{Name: "beta"},
			want:     config.WorkstreamConfig{Name: "beta", Remotes: []string{"origin"}},
		},
		{
			name:     "non-empty remotes override replaces base remotes",
			base:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{"origin"}},
			override: config.WorkstreamConfig{Remotes: []string{"mirror", "backup"}},
			want:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{"mirror", "backup"}},
		},
		{
			name:     "nil remotes override keeps base remotes",
			base:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{"origin"}},
			override: config.WorkstreamConfig{Remotes: nil},
			want:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{"origin"}},
		},
		{
			name:     "empty remotes override keeps base remotes",
			base:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{"origin"}},
			override: config.WorkstreamConfig{Remotes: []string{}},
			want:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{"origin"}},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := config.MergeWorkstream(tc.base, tc.override)
			s.Assert().Equal(tc.want, got)
		})
	}
}

func (s *ConfigSuite) TestWorkstreamByName() {
	cfg := config.WorkspaceConfig{
		Workstreams: []config.WorkstreamConfig{
			{Name: "alpha", Remotes: []string{"origin"}},
			{Name: "beta", Remotes: []string{"mirror"}},
			{Name: "alpha", Remotes: []string{"backup"}},
		},
	}

	s.Run("found first entry", func() {
		workstream, ok := cfg.WorkstreamByName("alpha")
		s.Assert().True(ok)
		s.Assert().Equal([]string{"origin"}, workstream.Remotes)
	})

	s.Run("found second entry", func() {
		workstream, ok := cfg.WorkstreamByName("beta")
		s.Assert().True(ok)
		s.Assert().Equal([]string{"mirror"}, workstream.Remotes)
	})

	s.Run("missing returns false", func() {
		_, ok := cfg.WorkstreamByName("missing")
		s.Assert().False(ok)
	})

	s.Run("duplicate names return first match", func() {
		workstream, ok := cfg.WorkstreamByName("alpha")
		s.Assert().True(ok)
		s.Assert().Equal([]string{"origin"}, workstream.Remotes)
	})
}

func (s *ConfigSuite) TestMergeRepo() {
	cases := []struct {
		name     string
		base     config.RepoConfig
		override config.RepoConfig
		want     config.RepoConfig
	}{
		{
			name: "empty override returns base unchanged",
			base: config.RepoConfig{
				Name: "svc-a", Path: "repos/svc-a", CloneURL: "https://github.com/org/svc-a",
				DefaultBranch: "main", TrackBranch: "dev", Upstream: "origin",
				Flags: []string{"--no-push"}, Remotes: []string{"origin"},
			},
			override: config.RepoConfig{},
			want: config.RepoConfig{
				Name: "svc-a", Path: "repos/svc-a", CloneURL: "https://github.com/org/svc-a",
				DefaultBranch: "main", TrackBranch: "dev", Upstream: "origin",
				Flags: []string{"--no-push"}, Remotes: []string{"origin"},
			},
		},
		{
			name:     "non-zero Name in override wins",
			base:     config.RepoConfig{Name: "old-name"},
			override: config.RepoConfig{Name: "new-name"},
			want:     config.RepoConfig{Name: "new-name"},
		},
		{
			name:     "zero Name keeps base Name",
			base:     config.RepoConfig{Name: "svc-a"},
			override: config.RepoConfig{Name: ""},
			want:     config.RepoConfig{Name: "svc-a"},
		},
		{
			name:     "non-zero Path in override wins",
			base:     config.RepoConfig{Path: "repos/old"},
			override: config.RepoConfig{Path: "repos/new"},
			want:     config.RepoConfig{Path: "repos/new"},
		},
		{
			name:     "zero Path keeps base Path",
			base:     config.RepoConfig{Path: "repos/svc-a"},
			override: config.RepoConfig{Path: ""},
			want:     config.RepoConfig{Path: "repos/svc-a"},
		},
		{
			name:     "non-zero CloneURL in override wins",
			base:     config.RepoConfig{CloneURL: "https://github.com/org/old"},
			override: config.RepoConfig{CloneURL: "https://gitea.example.com/org/new"},
			want:     config.RepoConfig{CloneURL: "https://gitea.example.com/org/new"},
		},
		{
			name:     "zero CloneURL keeps base CloneURL",
			base:     config.RepoConfig{CloneURL: "https://github.com/org/svc-a"},
			override: config.RepoConfig{CloneURL: ""},
			want:     config.RepoConfig{CloneURL: "https://github.com/org/svc-a"},
		},
		{
			name:     "non-nil Flags in override replaces base Flags",
			base:     config.RepoConfig{Flags: []string{"--no-push"}},
			override: config.RepoConfig{Flags: []string{"--push", "--verbose"}},
			want:     config.RepoConfig{Flags: []string{"--push", "--verbose"}},
		},
		{
			name:     "nil Flags in override keeps base Flags",
			base:     config.RepoConfig{Flags: []string{"--no-push"}},
			override: config.RepoConfig{Flags: nil},
			want:     config.RepoConfig{Flags: []string{"--no-push"}},
		},
		{
			name:     "non-nil Remotes in override replaces base Remotes",
			base:     config.RepoConfig{Remotes: []string{"origin"}},
			override: config.RepoConfig{Remotes: []string{"origin", "mirror"}},
			want:     config.RepoConfig{Remotes: []string{"origin", "mirror"}},
		},
		{
			name:     "nil Remotes in override keeps base Remotes",
			base:     config.RepoConfig{Remotes: []string{"origin"}},
			override: config.RepoConfig{Remotes: nil},
			want:     config.RepoConfig{Remotes: []string{"origin"}},
		},
		{
			name:     "non-zero DefaultBranch in override wins",
			base:     config.RepoConfig{DefaultBranch: "main"},
			override: config.RepoConfig{DefaultBranch: "develop"},
			want:     config.RepoConfig{DefaultBranch: "develop"},
		},
		{
			name:     "zero DefaultBranch keeps base DefaultBranch",
			base:     config.RepoConfig{DefaultBranch: "main"},
			override: config.RepoConfig{DefaultBranch: ""},
			want:     config.RepoConfig{DefaultBranch: "main"},
		},
		{
			name:     "non-zero TrackBranch in override wins",
			base:     config.RepoConfig{TrackBranch: "dev"},
			override: config.RepoConfig{TrackBranch: "test"},
			want:     config.RepoConfig{TrackBranch: "test"},
		},
		{
			name:     "zero TrackBranch keeps base TrackBranch",
			base:     config.RepoConfig{TrackBranch: "dev"},
			override: config.RepoConfig{TrackBranch: ""},
			want:     config.RepoConfig{TrackBranch: "dev"},
		},
		{
			name:     "non-zero Upstream in override wins",
			base:     config.RepoConfig{Upstream: "origin"},
			override: config.RepoConfig{Upstream: "personal"},
			want:     config.RepoConfig{Upstream: "personal"},
		},
		{
			name:     "zero Upstream keeps base Upstream",
			base:     config.RepoConfig{Upstream: "origin"},
			override: config.RepoConfig{Upstream: ""},
			want:     config.RepoConfig{Upstream: "origin"},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := config.MergeRepo(tc.base, tc.override)
			s.Assert().Equal(tc.want, got)
		})
	}
}

func (s *ConfigSuite) TestMergeWorkspace() {
	cases := []struct {
		name     string
		base     config.WorkspaceBlock
		override config.WorkspaceBlock
		want     config.WorkspaceBlock
	}{
		{
			name:     "empty override returns base unchanged",
			base:     config.WorkspaceBlock{Name: "default", Description: "main workspace", Repos: []string{"svc-a", "svc-b"}},
			override: config.WorkspaceBlock{},
			want:     config.WorkspaceBlock{Name: "default", Description: "main workspace", Repos: []string{"svc-a", "svc-b"}},
		},
		{
			name:     "non-zero Name in override wins",
			base:     config.WorkspaceBlock{Name: "old-name"},
			override: config.WorkspaceBlock{Name: "new-name"},
			want:     config.WorkspaceBlock{Name: "new-name"},
		},
		{
			name:     "zero Name keeps base Name",
			base:     config.WorkspaceBlock{Name: "default"},
			override: config.WorkspaceBlock{Name: ""},
			want:     config.WorkspaceBlock{Name: "default"},
		},
		{
			name:     "non-zero Description in override wins",
			base:     config.WorkspaceBlock{Description: "base desc"},
			override: config.WorkspaceBlock{Description: "override desc"},
			want:     config.WorkspaceBlock{Description: "override desc"},
		},
		{
			name:     "zero Description keeps base Description",
			base:     config.WorkspaceBlock{Description: "base desc"},
			override: config.WorkspaceBlock{Description: ""},
			want:     config.WorkspaceBlock{Description: "base desc"},
		},
		{
			name:     "non-nil Repos in override replaces base Repos",
			base:     config.WorkspaceBlock{Repos: []string{"svc-a"}},
			override: config.WorkspaceBlock{Repos: []string{"svc-b", "svc-c"}},
			want:     config.WorkspaceBlock{Repos: []string{"svc-b", "svc-c"}},
		},
		{
			name:     "nil Repos in override keeps base Repos",
			base:     config.WorkspaceBlock{Repos: []string{"svc-a", "svc-b"}},
			override: config.WorkspaceBlock{Repos: nil},
			want:     config.WorkspaceBlock{Repos: []string{"svc-a", "svc-b"}},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got := config.MergeWorkspace(tc.base, tc.override)
			s.Assert().Equal(tc.want, got)
		})
	}
}

func (s *ConfigSuite) TestRemoteByName() {
	cfg := config.WorkspaceConfig{
		Remotes: []config.RemoteConfig{
			{Name: "origin", Kind: "github"},
			{Name: "personal", Kind: "gitea"},
		},
	}

	s.Run("found by name", func() {
		r, ok := cfg.RemoteByName("origin")
		s.Assert().True(ok)
		s.Assert().Equal("github", r.Kind)
	})

	s.Run("second entry found", func() {
		r, ok := cfg.RemoteByName("personal")
		s.Assert().True(ok)
		s.Assert().Equal("gitea", r.Kind)
	})

	s.Run("not found returns false", func() {
		_, ok := cfg.RemoteByName("missing")
		s.Assert().False(ok)
	})

	s.Run("first matching name returned", func() {
		cfgDup := config.WorkspaceConfig{
			Remotes: []config.RemoteConfig{
				{Name: "origin", Kind: "github"},
				{Name: "origin", Kind: "gitea"},
			},
		}
		r, ok := cfgDup.RemoteByName("origin")
		s.Assert().True(ok)
		s.Assert().Equal("github", r.Kind)
	})
}
