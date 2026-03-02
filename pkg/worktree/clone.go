package worktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/spf13/cobra"
)

type cloneOperation struct {
	url     string
	setName string
	baseAbs string
	bareAbs string
	wt      workspace.WorktreeConfig
}

func registerClone(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:   "clone <url> <base-path> <branch> [branch...]",
		Short: "Create a worktree set from a remote repo",
		Args:  cobra.MinimumNArgs(3),
		RunE:  runClone,
	})
}

func runClone(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := workspace.LoadConfig(cmd)
	if err != nil {
		return err
	}

	op, err := prepareCloneOperation(cfg, cfgPath, args)
	if err != nil {
		return err
	}

	if err := executeCloneOperation(cmd, cfgPath, cfg.AutoGitignoreEnabled(), &op, args[2:]); err != nil {
		return err
	}

	cfg.Worktrees[op.setName] = op.wt
	if err := workspace.Save(cfgPath, cfg); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Created worktree set %q (%d branches)\n", op.setName, len(args)-2)
	return nil
}

func prepareCloneOperation(cfg *workspace.WorkspaceConfig, cfgPath string, args []string) (cloneOperation, error) {
	url, baseAbs, setName, err := parseCloneArgs(args)
	if err != nil {
		return cloneOperation{}, err
	}

	if err := validateCloneBasePath(cfgPath, baseAbs); err != nil {
		return cloneOperation{}, err
	}

	if err := ensureCloneTarget(cfg, setName, baseAbs); err != nil {
		return cloneOperation{}, err
	}

	bareAbs := filepath.Join(baseAbs, ".bare")
	wt, err := initWorktreeConfig(cfgPath, url, bareAbs)
	if err != nil {
		return cloneOperation{}, err
	}

	return cloneOperation{url: url, setName: setName, baseAbs: baseAbs, bareAbs: bareAbs, wt: wt}, nil
}

func executeCloneOperation(cmd *cobra.Command, cfgPath string, gitignore bool, op *cloneOperation, branches []string) error {
	if err := createWorktreeSet(op.url, op.baseAbs, op.bareAbs); err != nil {
		return err
	}

	if err := addCloneBranches(cmd, cfgPath, gitignore, &op.wt, op.bareAbs, op.baseAbs, branches); err != nil {
		cleanupFailedClone(op)
		return err
	}

	return nil
}

// cleanupFailedClone removes the bare repo and any branch worktrees that were
// already created before the clone operation failed. Branch absolute paths are
// reconstructed from relPath because the .gitw config stores relative paths;
// addCloneBranches always places branches at <baseAbs>/<branch>, so
// filepath.Base(relPath) == branch name.
func cleanupFailedClone(op *cloneOperation) {
	_ = os.RemoveAll(op.bareAbs)
	for _, relPath := range op.wt.Branches {
		branchAbs := filepath.Join(op.baseAbs, filepath.Base(relPath))
		_ = os.RemoveAll(branchAbs)
	}
}

func parseCloneArgs(args []string) (string, string, string, error) {
	url := args[0]
	baseAbs, err := filepath.Abs(args[1])
	if err != nil {
		return "", "", "", fmt.Errorf("resolving base path: %w", err)
	}

	return url, baseAbs, filepath.Base(baseAbs), nil
}

func validateCloneBasePath(cfgPath, baseAbs string) error {
	baseRel, err := workspace.RelPath(cfgPath, baseAbs)
	if err != nil {
		return err
	}

	if _, err := workspace.ResolveRepoPath(cfgPath, baseRel); err != nil {
		return fmt.Errorf("base path must be inside workspace root")
	}

	return nil
}

func ensureCloneTarget(cfg *workspace.WorkspaceConfig, setName, baseAbs string) error {
	if _, exists := cfg.Worktrees[setName]; exists {
		return fmt.Errorf("worktree set %q already exists", setName)
	}

	bareAbs := filepath.Join(baseAbs, ".bare")
	if _, err := os.Stat(bareAbs); err == nil {
		return fmt.Errorf("bare path already exists: %s", bareAbs)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking bare path: %w", err)
	}

	return nil
}

func createWorktreeSet(url, baseAbs, bareAbs string) error {
	if err := os.MkdirAll(baseAbs, 0o755); err != nil {
		return fmt.Errorf("creating base path: %w", err)
	}

	if err := gitutil.CloneBare(context.Background(), url, bareAbs); err != nil {
		return err
	}

	return nil
}

func initWorktreeConfig(cfgPath, url, bareAbs string) (workspace.WorktreeConfig, error) {
	bareRel, err := workspace.RelPath(cfgPath, bareAbs)
	if err != nil {
		return workspace.WorktreeConfig{}, err
	}

	return workspace.WorktreeConfig{
		URL:      url,
		BarePath: bareRel,
		Branches: make(map[string]string),
	}, nil
}

func addCloneBranches(cmd *cobra.Command, cfgPath string, gitignore bool, wt *workspace.WorktreeConfig, bareAbs, baseAbs string, branches []string) error {
	for _, branch := range branches {
		relPath, err := addBranchWorktree(cfgPath, bareAbs, filepath.Join(baseAbs, branch), branch)
		if err != nil {
			return err
		}

		wt.Branches[branch] = relPath
		writeGitignoreWarning(cmd, cfgPath, relPath, gitignore)
	}

	return nil
}

func addBranchWorktree(cfgPath, bareAbs, branchAbs, branch string) (string, error) {
	if err := gitutil.AddWorktree(context.Background(), bareAbs, branchAbs, branch); err != nil {
		return "", err
	}

	return workspace.RelPath(cfgPath, branchAbs)
}
