package worktree

import (
	"path/filepath"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/spf13/cobra"
)

type branchTarget struct {
	SetName  string
	Branch   string
	RelPath  string
	RepoName string
}

// findByRepoName locates the worktree set and branch for a synthesized repo
// name (e.g. "infra-dev"). O(sets × branches) — acceptable for realistic
// workspace sizes (single-digit sets with 2-5 branches each).
func findByRepoName(cfg *config.WorkspaceConfig, name string) (branchTarget, bool) {
	for setName, wt := range cfg.Worktrees {
		for branch, relPath := range wt.Branches {
			if config.WorktreeRepoName(setName, branch) == name {
				return branchTarget{SetName: setName, Branch: branch, RelPath: relPath, RepoName: name}, true
			}
		}
	}

	return branchTarget{}, false
}

func bareAbsPath(cfgPath string, wt config.WorktreeConfig) (string, error) {
	return config.ResolveRepoPath(cfgPath, wt.BarePath)
}

func defaultBranchAbsPath(cfgPath string, wt config.WorktreeConfig, branch string) (string, error) {
	bareAbs, err := bareAbsPath(cfgPath, wt)
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(bareAbs), branch), nil
}

func writeGitignoreWarning(cmd *cobra.Command, cfgPath, relPath string, gitignore bool) {
	if !gitignore {
		return
	}

	if err := gitutil.EnsureGitignore(config.ConfigDir(cfgPath), relPath); err != nil {
		output.Writef(cmd.ErrOrStderr(), "warning: .gitignore: %v\n", err)
	}
}
