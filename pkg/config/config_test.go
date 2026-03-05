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
	meta     config.WorkspaceMeta
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
	meta config.WorkspaceMeta
	want bool
}

var syncPushEnabledCases = func() []syncPushEnabledCase {
	trueValue := true
	falseValue := false
	return []syncPushEnabledCase{
		{name: "nil defaults true", meta: config.WorkspaceMeta{}, want: true},
		{name: "explicit true", meta: config.WorkspaceMeta{SyncPush: &trueValue}, want: true},
		{name: "explicit false", meta: config.WorkspaceMeta{SyncPush: &falseValue}, want: false},
	}
}()

func TestConfigSuite(t *testing.T) {
	testutil.RunSuite(t, new(ConfigSuite))
}

func (s *ConfigSuite) TestSyncPushEnabled() {
	for _, tt := range syncPushEnabledCases {
		s.Run(tt.name, func() {
			cfg := config.WorkspaceConfig{Workspace: tt.meta}
			s.Equal(tt.want, cfg.SyncPushEnabled())
		})
	}
}

func (s *ConfigSuite) TestBranchAccessors() {
	for _, tt := range branchAccessorCases() {
		s.Run(tt.name, func() {
			cfg := config.WorkspaceConfig{Workspace: tt.meta}
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
				Workspace: config.WorkspaceMeta{DefaultBranch: tt.workspace},
				Repos:     map[string]config.RepoConfig{"frontend": {DefaultBranch: tt.repoDefault}},
			}
			s.Equal(tt.want, cfg.ResolveDefaultBranch("frontend"))
		})
	}
}

func (s *ConfigSuite) TestResolveDefaultBranchWorktreeRepo() {
	cfg := worktreeBranchConfig()
	cfg.Workspace.DefaultBranch = "main"

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

func branchAccessorCases() []branchAccessorCase {
	trueValue := true
	falseValue := false

	return []branchAccessorCase{
		{name: "nil defaults true", meta: config.WorkspaceMeta{}, wantSync: true, wantUp: true, wantPush: true},
		{name: "explicit false", meta: config.WorkspaceMeta{BranchSyncSource: &falseValue, BranchSetUpstream: &falseValue, BranchPush: &falseValue}, wantSync: false, wantUp: false, wantPush: false},
		{name: "explicit true", meta: config.WorkspaceMeta{BranchSyncSource: &trueValue, BranchSetUpstream: &trueValue, BranchPush: &trueValue}, wantSync: true, wantUp: true, wantPush: true},
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
