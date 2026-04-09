package config_test

import (
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/stretchr/testify/assert"
)

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

func TestSyncPushEnabled(t *testing.T) {
	for _, tt := range syncPushEnabledCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WorkspaceConfig{Metarepo: tt.meta}
			assert.Equal(t, tt.want, cfg.SyncPushEnabled())
		})
	}
}

func TestBranchAccessors(t *testing.T) {
	for _, tt := range branchAccessorCases() {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WorkspaceConfig{Metarepo: tt.meta}
			assert.Equal(t, tt.wantSync, cfg.BranchSyncSourceEnabled())
			assert.Equal(t, tt.wantUp, cfg.BranchSetUpstreamEnabled())
			assert.Equal(t, tt.wantPush, cfg.BranchPushEnabled())
		})
	}
}

func TestResolveDefaultBranch(t *testing.T) {
	for _, tt := range resolveDefaultCases() {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WorkspaceConfig{
				Metarepo: config.MetarepoConfig{DefaultBranch: tt.workspace},
				Repos:    map[string]config.RepoConfig{"frontend": {DefaultBranch: tt.repoDefault}},
			}

			assert.Equal(t, tt.want, cfg.ResolveDefaultBranch("frontend"))
		})
	}
}

func TestResolveDefaultBranchForWorktreeRepo(t *testing.T) {
	cfg := worktreeBranchConfig()
	cfg.Metarepo.DefaultBranch = "main"

	assert.Equal(t, "dev", cfg.ResolveDefaultBranch("infra-dev"))
	assert.Equal(t, "prod", cfg.ResolveDefaultBranch("infra-prod"))
	assert.Equal(t, "main", cfg.ResolveDefaultBranch("backend"))
}

func TestWorktreeBranchForRepo(t *testing.T) {
	cfg := worktreeBranchConfig()

	for _, tt := range worktreeBranchCases() {
		t.Run(tt.name, func(t *testing.T) {
			branch, ok := cfg.WorktreeBranchForRepo(tt.repo)
			assert.Equal(t, tt.found, ok)
			assert.Equal(t, tt.want, branch)
		})
	}
}

func TestWorktreeRepoToSetIndex(t *testing.T) {
	for _, tt := range worktreeIndexCases() {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WorkspaceConfig{Worktrees: tt.worktrees}
			assert.Equal(t, tt.want, config.WorktreeRepoToSetIndex(&cfg))
		})
	}
}

func TestWorkspaceBlockFields(t *testing.T) {
	wb := config.WorkspaceBlock{
		Name:        "payments",
		Description: "Payment processing",
		Repos:       []string{"api", "gateway"},
	}

	assert.Equal(t, "payments", wb.Name)
	assert.Equal(t, "Payment processing", wb.Description)
	assert.Equal(t, []string{"api", "gateway"}, wb.Repos)
}

func TestRepoConfigIsAlias(t *testing.T) {
	cases := []struct {
		name        string
		trackBranch string
		want        bool
	}{
		{name: "empty track_branch", trackBranch: "", want: false},
		{name: "non-empty track_branch", trackBranch: "dev", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rc := config.RepoConfig{TrackBranch: tc.trackBranch}
			assert.Equal(t, tc.want, rc.IsAlias())
		})
	}
}

func TestBranchActionConstants(t *testing.T) {
	assert.Equal(t, config.BranchAction("allow"), config.ActionAllow)
	assert.Equal(t, config.BranchAction("block"), config.ActionBlock)
	assert.Equal(t, config.BranchAction("warn"), config.ActionWarn)
	assert.Equal(t, config.BranchAction("require-flag"), config.ActionRequireFlag)
}

func TestMergeRemote(t *testing.T) {
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
			name: "BranchRuleConfig bool pointer fields preserved",
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
		t.Run(tc.name, func(t *testing.T) {
			got := config.MergeRemote(tc.base, tc.override)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMergeSyncPair(t *testing.T) {
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := config.MergeSyncPair(tc.base, tc.override)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMergeWorkstream(t *testing.T) {
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
			name:     "empty slice remotes override replaces base explicit no remotes",
			base:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{"origin"}},
			override: config.WorkstreamConfig{Remotes: []string{}},
			want:     config.WorkstreamConfig{Name: "alpha", Remotes: []string{}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := config.MergeWorkstream(tc.base, tc.override)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWorkstreamByName(t *testing.T) {
	cfg := config.WorkspaceConfig{
		Workstreams: []config.WorkstreamConfig{
			{Name: "alpha", Remotes: []string{"origin"}},
			{Name: "beta", Remotes: []string{"mirror"}},
			{Name: "alpha", Remotes: []string{"backup"}},
		},
	}

	t.Run("found first entry", func(t *testing.T) {
		workstream, ok := cfg.WorkstreamByName("alpha")
		assert.True(t, ok)
		assert.Equal(t, []string{"origin"}, workstream.Remotes)
	})

	t.Run("found second entry", func(t *testing.T) {
		workstream, ok := cfg.WorkstreamByName("beta")
		assert.True(t, ok)
		assert.Equal(t, []string{"mirror"}, workstream.Remotes)
	})

	t.Run("missing returns false", func(t *testing.T) {
		_, ok := cfg.WorkstreamByName("missing")
		assert.False(t, ok)
	})

	t.Run("duplicate names return first match", func(t *testing.T) {
		workstream, ok := cfg.WorkstreamByName("alpha")
		assert.True(t, ok)
		assert.Equal(t, []string{"origin"}, workstream.Remotes)
	})
}

func TestMergeRepo(t *testing.T) {
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
			name:     "non-zero Path in override wins",
			base:     config.RepoConfig{Path: "repos/old"},
			override: config.RepoConfig{Path: "repos/new"},
			want:     config.RepoConfig{Path: "repos/new"},
		},
		{
			name:     "non-zero CloneURL in override wins",
			base:     config.RepoConfig{CloneURL: "https://github.com/org/old"},
			override: config.RepoConfig{CloneURL: "https://gitea.example.com/org/new"},
			want:     config.RepoConfig{CloneURL: "https://gitea.example.com/org/new"},
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
			name:     "non-zero TrackBranch in override wins",
			base:     config.RepoConfig{TrackBranch: "dev"},
			override: config.RepoConfig{TrackBranch: "test"},
			want:     config.RepoConfig{TrackBranch: "test"},
		},
		{
			name:     "non-zero Upstream in override wins",
			base:     config.RepoConfig{Upstream: "origin"},
			override: config.RepoConfig{Upstream: "personal"},
			want:     config.RepoConfig{Upstream: "personal"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := config.MergeRepo(tc.base, tc.override)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMergeWorkspace(t *testing.T) {
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
			name:     "non-zero Description in override wins",
			base:     config.WorkspaceBlock{Description: "base desc"},
			override: config.WorkspaceBlock{Description: "override desc"},
			want:     config.WorkspaceBlock{Description: "override desc"},
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
		t.Run(tc.name, func(t *testing.T) {
			got := config.MergeWorkspace(tc.base, tc.override)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRemoteByName(t *testing.T) {
	cfg := config.WorkspaceConfig{
		Remotes: []config.RemoteConfig{
			{Name: "origin", Kind: "github"},
			{Name: "personal", Kind: "gitea"},
		},
	}

	t.Run("found by name", func(t *testing.T) {
		r, ok := cfg.RemoteByName("origin")
		assert.True(t, ok)
		assert.Equal(t, "github", r.Kind)
	})

	t.Run("second entry found", func(t *testing.T) {
		r, ok := cfg.RemoteByName("personal")
		assert.True(t, ok)
		assert.Equal(t, "gitea", r.Kind)
	})

	t.Run("not found returns false", func(t *testing.T) {
		_, ok := cfg.RemoteByName("missing")
		assert.False(t, ok)
	})

	t.Run("first matching name returned", func(t *testing.T) {
		cfgDup := config.WorkspaceConfig{
			Remotes: []config.RemoteConfig{
				{Name: "origin", Kind: "github"},
				{Name: "origin", Kind: "gitea"},
			},
		}

		r, ok := cfgDup.RemoteByName("origin")
		assert.True(t, ok)
		assert.Equal(t, "github", r.Kind)
	})
}

func TestResolveWorkstreamRemotes(t *testing.T) {
	cases := []struct {
		name              string
		repoRemotes       []string
		workstreamName    string
		workstreamRemotes []string
		metaRemotes       []string
		wantRemotes       []string
		wantSource        string
	}{
		{
			name:              "repo remotes set returns repo remotes",
			repoRemotes:       []string{"origin", "personal"},
			workstreamName:    "ws1",
			workstreamRemotes: []string{"mirror"},
			metaRemotes:       []string{"default"},
			wantRemotes:       []string{"origin", "personal"},
			wantSource:        "repo",
		},
		{
			name:              "repo remotes explicit empty stops cascade",
			repoRemotes:       []string{},
			workstreamName:    "ws1",
			workstreamRemotes: []string{"mirror"},
			metaRemotes:       []string{"default"},
			wantRemotes:       []string{},
			wantSource:        "repo",
		},
		{
			name:              "workstream remotes used when repo remotes nil",
			repoRemotes:       nil,
			workstreamName:    "ws1",
			workstreamRemotes: []string{"personal"},
			metaRemotes:       []string{"default"},
			wantRemotes:       []string{"personal"},
			wantSource:        "workstream",
		},
		{
			name:              "workstream explicit empty stops cascade",
			repoRemotes:       nil,
			workstreamName:    "ws1",
			workstreamRemotes: []string{},
			metaRemotes:       []string{"default"},
			wantRemotes:       []string{},
			wantSource:        "workstream",
		},
		{
			name:              "metarepo remotes used when repo and workstream nil",
			repoRemotes:       nil,
			workstreamName:    "ws1",
			workstreamRemotes: nil,
			metaRemotes:       []string{"origin"},
			wantRemotes:       []string{"origin"},
			wantSource:        "metarepo",
		},
		{
			name:              "all nil returns none",
			repoRemotes:       nil,
			workstreamName:    "ws1",
			workstreamRemotes: nil,
			metaRemotes:       nil,
			wantRemotes:       nil,
			wantSource:        "none",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.WorkspaceConfig{
				Metarepo: config.MetarepoConfig{DefaultRemotes: tc.metaRemotes},
				Repos: map[string]config.RepoConfig{
					"svc-a": {Name: "svc-a", Remotes: tc.repoRemotes},
				},
			}

			if tc.workstreamName == "ws1" {
				cfg.Workstreams = []config.WorkstreamConfig{{Name: "ws1", Remotes: tc.workstreamRemotes}}
			}

			gotRemotes, gotSource := cfg.ResolveWorkstreamRemotes("svc-a", tc.workstreamName)
			assert.Equal(t, tc.wantRemotes, gotRemotes)
			assert.Equal(t, tc.wantSource, gotSource)
		})
	}
}

func TestResolveRepoRemotes(t *testing.T) {
	cases := []struct {
		name        string
		repoRemotes []string
		metaRemotes []string
		wantRemotes []string
		wantSource  string
	}{
		{
			name:        "repo remotes set returns repo remotes",
			repoRemotes: []string{"origin", "personal"},
			metaRemotes: []string{"default"},
			wantRemotes: []string{"origin", "personal"},
			wantSource:  "repo",
		},
		{
			name:        "repo explicit empty stops cascade",
			repoRemotes: []string{},
			metaRemotes: []string{"default"},
			wantRemotes: []string{},
			wantSource:  "repo",
		},
		{
			name:        "metarepo remotes used when repo nil",
			repoRemotes: nil,
			metaRemotes: []string{"origin"},
			wantRemotes: []string{"origin"},
			wantSource:  "metarepo",
		},
		{
			name:        "repo nil and metarepo nil returns none",
			repoRemotes: nil,
			metaRemotes: nil,
			wantRemotes: nil,
			wantSource:  "none",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.WorkspaceConfig{
				Metarepo: config.MetarepoConfig{DefaultRemotes: tc.metaRemotes},
				Repos: map[string]config.RepoConfig{
					"svc-a": {Name: "svc-a", Remotes: tc.repoRemotes},
				},
			}

			gotRemotes, gotSource := cfg.ResolveRepoRemotes("svc-a")
			assert.Equal(t, tc.wantRemotes, gotRemotes)
			assert.Equal(t, tc.wantSource, gotSource)
		})
	}
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
