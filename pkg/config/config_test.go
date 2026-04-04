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
