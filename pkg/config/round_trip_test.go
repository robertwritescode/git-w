package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type RoundTripSuite struct {
	suite.Suite
}

func TestRoundTripSuite(t *testing.T) {
	testutil.RunSuite(t, new(RoundTripSuite))
}

type roundTripCase struct {
	name    string
	initial string
	setup   func(*config.WorkspaceConfig)
	comment string
	anchor  string
}

func (s *RoundTripSuite) TestRoundTrip_PerBlockType() {
	cases := []roundTripCase{
		{
			name: "metarepo",
			initial: `# workspace identity
[metarepo]
name = "myws"
`,
			setup: func(cfg *config.WorkspaceConfig) {
				cfg.Metarepo.DefaultBranch = "main"
			},
			comment: "# workspace identity",
			anchor:  "name",
		},
		{
			name: "workspace block",
			initial: `[metarepo]
name = "myws"

# dev workspace
[[workspace]]
name = "dev"
repos = ["svc-a"]
`,
			setup: func(cfg *config.WorkspaceConfig) {
				cfg.Workspaces = append(cfg.Workspaces, config.WorkspaceBlock{
					Name:  "prod",
					Repos: []string{"svc-a"},
				})
			},
			comment: "# dev workspace",
			// go-toml marshals string values with single quotes in workspace blocks
			anchor: "[[workspace]]",
		},
		{
			name: "repo",
			initial: `[metarepo]
name = "myws"

# primary repo
[[repo]]
name = "svc-a"
path = "services/svc-a"
`,
			setup: func(cfg *config.WorkspaceConfig) {
				cfg.Repos["svc-b"] = config.RepoConfig{Path: "services/svc-b"}
			},
			comment: "# primary repo",
			anchor:  `[[repo]]`,
		},
		{
			name: "repo path mutation",
			initial: `[metarepo]
name = "myws"

# main service
[[repo]]
name = "svc-a"
path = "services/svc-a"
`,
			setup: func(cfg *config.WorkspaceConfig) {
				rc := cfg.Repos["svc-a"]
				rc.Path = "apps/svc-a"
				cfg.Repos["svc-a"] = rc
			},
			comment: "# main service",
			anchor:  `[[repo]]`,
		},
		{
			name: "remote",
			initial: `[metarepo]
name = "myws"

# github remote
[[remote]]
name = "origin"
url = "https://github.com/org/repo"
`,
			setup: func(cfg *config.WorkspaceConfig) {
				cfg.Remotes = append(cfg.Remotes, config.RemoteConfig{
					Name: "mirror",
					URL:  "https://github.com/org/mirror",
				})
			},
			comment: "# github remote",
			anchor:  `[[remote]]`,
		},
		{
			name: "remote with branch_rule",
			initial: `[metarepo]
name = "myws"

[[remote]]
name = "origin"
url = "https://github.com/org/repo"

# block main
[[remote.branch_rule]]
pattern = "main"
action = "block"
`,
			setup: func(cfg *config.WorkspaceConfig) {
				for i, r := range cfg.Remotes {
					if r.Name == "origin" {
						cfg.Remotes[i].URL = "https://github.com/org/updated"
						break
					}
				}
			},
			comment: "# block main",
			anchor:  `[[remote.branch_rule]]`,
		},
		{
			name: "sync_pair",
			initial: `[metarepo]
name = "myws"

[[remote]]
name = "svc-a"

[[remote]]
name = "svc-b"

[[remote]]
name = "svc-c"

[[remote]]
name = "svc-d"

# mirror pair
[[sync_pair]]
from = "svc-a"
to = "svc-b"
`,
			setup: func(cfg *config.WorkspaceConfig) {
				cfg.SyncPairs = append(cfg.SyncPairs, config.SyncPairConfig{
					From: "svc-c",
					To:   "svc-d",
				})
			},
			comment: "# mirror pair",
			anchor:  `[[sync_pair]]`,
		},
		{
			name: "workstream",
			initial: `[metarepo]
name = "myws"

[[remote]]
name = "origin"
url = "https://github.com/org/repo"

# feature stream
[[workstream]]
name = "feature-x"
remotes = ["origin"]
`,
			setup: func(cfg *config.WorkspaceConfig) {
				cfg.Workstreams = append(cfg.Workstreams, config.WorkstreamConfig{
					Name:    "feature-y",
					Remotes: []string{"origin"},
				})
			},
			comment: "# feature stream",
			anchor:  `[[workstream]]`,
		},
		{
			name: "groups",
			initial: `[metarepo]
name = "myws"

[[repo]]
name = "svc-a"
path = "services/svc-a"

[[repo]]
name = "svc-b"
path = "services/svc-b"

# frontend group
[groups.web]
repos = ["svc-a", "svc-b"]
`,
			setup: func(cfg *config.WorkspaceConfig) {
				cfg.Groups["backend"] = config.GroupConfig{Repos: []string{"svc-a"}}
			},
			comment: "# frontend group",
			anchor:  `[groups.web]`,
		},
		{
			name: "worktrees",
			initial: `[metarepo]
name = "myws"

# infra worktree
[worktrees.infra]
url = "https://github.com/org/infra"
bare_path = "infra/.bare"

[worktrees.infra.branches]
main = "infra/main"
`,
			setup: func(cfg *config.WorkspaceConfig) {
				cfg.Worktrees["platform"] = config.WorktreeConfig{
					URL:      "https://github.com/org/platform",
					BarePath: "platform/.bare",
					Branches: map[string]string{"main": "platform/main"},
				}
			},
			comment: "# infra worktree",
			anchor:  "infra",
		},
	}

	for _, tc := range cases {
		tc := tc
		s.Run(tc.name, func() {
			dir := s.T().TempDir()
			cfgPath := filepath.Join(dir, ".gitw")
			s.Require().NoError(os.WriteFile(cfgPath, []byte(tc.initial), 0o644))

			cfg, err := config.Load(cfgPath)
			s.Require().NoError(err)

			tc.setup(cfg)

			s.Require().NoError(config.Save(cfgPath, cfg))

			data, err := os.ReadFile(cfgPath)
			s.Require().NoError(err)
			out := string(data)

			s.Assert().Contains(out, tc.comment)
			commentIdx := strings.Index(out, tc.comment)
			anchorIdx := strings.Index(out, tc.anchor)
			s.Assert().GreaterOrEqual(anchorIdx, 0, "anchor %q not found in output", tc.anchor)
			s.Assert().Less(commentIdx, anchorIdx, "comment %q should precede anchor %q", tc.comment, tc.anchor)
		})
	}
}

func (s *RoundTripSuite) TestRoundTrip_AllBlocks() {
	initial := `# all-blocks workspace
[metarepo]
name = "all-blocks"

# all-blocks ws
[[workspace]]
name = "default"
repos = ["svc-a"]

# all-blocks repo
[[repo]]
name = "svc-a"
path = "services/svc-a"

# all-blocks remote
[[remote]]
name = "origin"
url = "https://github.com/org/svc-a"

[[remote]]
name = "svc-a"

[[remote]]
name = "svc-b"

# all-blocks sync
[[sync_pair]]
from = "svc-a"
to = "svc-b"

# all-blocks stream
[[workstream]]
name = "feature-x"
remotes = ["origin"]

[[repo]]
name = "svc-b"
path = "services/svc-b"

# all-blocks group
[groups.all]
repos = ["svc-a", "svc-b"]

# all-blocks wt
[worktrees.set1]
url = "https://github.com/org/infra"
bare_path = "infra/.bare"

[worktrees.set1.branches]
main = "infra/main"
`
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, ".gitw")
	s.Require().NoError(os.WriteFile(cfgPath, []byte(initial), 0o644))

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)

	// identity round-trip — no mutations
	s.Require().NoError(config.Save(cfgPath, cfg))

	data, err := os.ReadFile(cfgPath)
	s.Require().NoError(err)
	out := string(data)

	type commentAnchorPair struct {
		comment string
		anchor  string
	}

	pairs := []commentAnchorPair{
		{"# all-blocks workspace", "[metarepo]"},
		{"# all-blocks ws", "[[workspace]]"},
		{"# all-blocks repo", "[[repo]]"},
		{"# all-blocks remote", "[[remote]]"},
		{"# all-blocks sync", "[[sync_pair]]"},
		{"# all-blocks stream", "[[workstream]]"},
		{"# all-blocks group", "[groups.all]"},
		{"# all-blocks wt", "set1"},
	}

	for _, p := range pairs {
		s.Assert().Contains(out, p.comment, "comment %q not found", p.comment)
		commentIdx := strings.Index(out, p.comment)
		anchorIdx := strings.Index(out, p.anchor)
		s.Assert().GreaterOrEqual(anchorIdx, 0, "anchor %q not found in output", p.anchor)
		s.Assert().Less(commentIdx, anchorIdx, "comment %q should precede anchor %q", p.comment, p.anchor)
	}
}
