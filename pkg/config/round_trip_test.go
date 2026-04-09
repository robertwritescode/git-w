package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRoundTripPreservesMetarepoComment(t *testing.T) {
	out := runRoundTripMutation(t, `# workspace identity
[metarepo]
name = "myws"
`, func(cfg *config.WorkspaceConfig) {
		cfg.Metarepo.DefaultBranch = "main"
	})

	assertCommentBeforeAnchor(t, out, "# workspace identity", "name")
}

func TestRoundTripPreservesWorkspaceComment(t *testing.T) {
	out := runRoundTripMutation(t, `[metarepo]
name = "myws"

# dev workspace
[[workspace]]
name = "dev"
repos = ["svc-a"]
`, func(cfg *config.WorkspaceConfig) {
		cfg.Workspaces = append(cfg.Workspaces, config.WorkspaceBlock{
			Name:  "prod",
			Repos: []string{"svc-a"},
		})
	})

	assertCommentBeforeAnchor(t, out, "# dev workspace", "[[workspace]]")
}

func TestRoundTripPreservesRepoComment(t *testing.T) {
	out := runRoundTripMutation(t, `[metarepo]
name = "myws"

# primary repo
[[repo]]
name = "svc-a"
path = "services/svc-a"
`, func(cfg *config.WorkspaceConfig) {
		cfg.Repos["svc-b"] = config.RepoConfig{Path: "services/svc-b"}
	})

	assertCommentBeforeAnchor(t, out, "# primary repo", "[[repo]]")
}

func TestRoundTripPreservesRepoCommentAfterPathMutation(t *testing.T) {
	out := runRoundTripMutation(t, `[metarepo]
name = "myws"

# main service
[[repo]]
name = "svc-a"
path = "services/svc-a"
`, func(cfg *config.WorkspaceConfig) {
		repo := cfg.Repos["svc-a"]
		repo.Path = "apps/svc-a"
		cfg.Repos["svc-a"] = repo
	})

	assertCommentBeforeAnchor(t, out, "# main service", "[[repo]]")
}

func TestRoundTripPreservesRemoteComment(t *testing.T) {
	out := runRoundTripMutation(t, `[metarepo]
name = "myws"

# github remote
[[remote]]
name = "origin"
url = "https://github.com/org/repo"
`, func(cfg *config.WorkspaceConfig) {
		cfg.Remotes = append(cfg.Remotes, config.RemoteConfig{
			Name: "mirror",
			URL:  "https://github.com/org/mirror",
		})
	})

	assertCommentBeforeAnchor(t, out, "# github remote", "[[remote]]")
}

func TestRoundTripPreservesRemoteBranchRuleComment(t *testing.T) {
	out := runRoundTripMutation(t, `[metarepo]
name = "myws"

[[remote]]
name = "origin"
url = "https://github.com/org/repo"

# block main
[[remote.branch_rule]]
pattern = "main"
action = "block"
`, func(cfg *config.WorkspaceConfig) {
		for i, remote := range cfg.Remotes {
			if remote.Name == "origin" {
				cfg.Remotes[i].URL = "https://github.com/org/updated"
				break
			}
		}
	})

	assertCommentBeforeAnchor(t, out, "# block main", "[[remote.branch_rule]]")
}

func TestRoundTripPreservesSyncPairComment(t *testing.T) {
	out := runRoundTripMutation(t, `[metarepo]
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
`, func(cfg *config.WorkspaceConfig) {
		cfg.SyncPairs = append(cfg.SyncPairs, config.SyncPairConfig{
			From: "svc-c",
			To:   "svc-d",
		})
	})

	assertCommentBeforeAnchor(t, out, "# mirror pair", "[[sync_pair]]")
}

func TestRoundTripPreservesWorkstreamComment(t *testing.T) {
	out := runRoundTripMutation(t, `[metarepo]
name = "myws"

[[remote]]
name = "origin"
url = "https://github.com/org/repo"

# feature stream
[[workstream]]
name = "feature-x"
remotes = ["origin"]
`, func(cfg *config.WorkspaceConfig) {
		cfg.Workstreams = append(cfg.Workstreams, config.WorkstreamConfig{
			Name:    "feature-y",
			Remotes: []string{"origin"},
		})
	})

	assertCommentBeforeAnchor(t, out, "# feature stream", "[[workstream]]")
}

func TestRoundTripPreservesGroupComment(t *testing.T) {
	out := runRoundTripMutation(t, `[metarepo]
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
`, func(cfg *config.WorkspaceConfig) {
		cfg.Groups["backend"] = config.GroupConfig{Repos: []string{"svc-a"}}
	})

	assertCommentBeforeAnchor(t, out, "# frontend group", "[groups.web]")
}

func TestRoundTripPreservesWorktreeComment(t *testing.T) {
	out := runRoundTripMutation(t, `[metarepo]
name = "myws"

# infra worktree
[worktrees.infra]
url = "https://github.com/org/infra"
bare_path = "infra/.bare"

[worktrees.infra.branches]
main = "infra/main"
`, func(cfg *config.WorkspaceConfig) {
		cfg.Worktrees["platform"] = config.WorktreeConfig{
			URL:      "https://github.com/org/platform",
			BarePath: "platform/.bare",
			Branches: map[string]string{"main": "platform/main"},
		}
	})

	assertCommentBeforeAnchor(t, out, "# infra worktree", "[worktrees.infra]")
}

func TestRoundTripPreservesAllBlockComments(t *testing.T) {
	out := runRoundTripMutation(t, `# all-blocks workspace
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
`, func(cfg *config.WorkspaceConfig) {})

	assertCommentBeforeAnchor(t, out, "# all-blocks workspace", "[metarepo]")
	assertCommentBeforeAnchor(t, out, "# all-blocks ws", "[[workspace]]")
	assertCommentBeforeAnchor(t, out, "# all-blocks repo", "[[repo]]")
	assertCommentBeforeAnchor(t, out, "# all-blocks remote", "[[remote]]")
	assertCommentBeforeAnchor(t, out, "# all-blocks sync", "[[sync_pair]]")
	assertCommentBeforeAnchor(t, out, "# all-blocks stream", "[[workstream]]")
	assertCommentBeforeAnchor(t, out, "# all-blocks group", "[groups.all]")
	assertCommentBeforeAnchor(t, out, "# all-blocks wt", "[worktrees.set1]")
}

func runRoundTripMutation(t *testing.T, initial string, mutate func(*config.WorkspaceConfig)) string {
	t.Helper()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".gitw")
	require.NoError(t, os.WriteFile(cfgPath, []byte(initial), 0o644))

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)

	mutate(cfg)
	require.NoError(t, config.Save(cfgPath, cfg))

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)

	return string(data)
}

func assertCommentBeforeAnchor(t *testing.T, out, comment, anchor string) {
	t.Helper()

	require.Contains(t, out, comment)
	commentIdx := strings.Index(out, comment)
	anchorIdx := strings.Index(out, anchor)
	require.GreaterOrEqual(t, anchorIdx, 0, "anchor %q not found in output", anchor)
	require.Less(t, commentIdx, anchorIdx, "comment %q should precede anchor %q", comment, anchor)
}
