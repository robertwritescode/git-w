package worktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/spf13/cobra"
)

type dropOperation struct {
	setName string
	wt      config.WorktreeConfig
	bareAbs string
	force   bool
}

func registerDrop(root *cobra.Command) {
	dropCmd := &cobra.Command{
		Use:   "drop <set-name>",
		Short: "Drop all worktrees and bare repo for a set",
		Args:  cobra.ExactArgs(1),
		RunE:  runDrop,
	}
	dropCmd.Flags().Bool("force", false, "drop even if worktrees are dirty or local-ahead")
	root.AddCommand(dropCmd)
}

func runDrop(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	op, err := prepareDropOperation(cmd, cfg, cfgPath, args[0])
	if err != nil {
		return err
	}

	if err := executeDropOperation(cmd.Context(), cfgPath, op); err != nil {
		return err
	}

	if err := finalizeDropOperation(cfg, cfgPath, op); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Dropped worktree set %q\n", op.setName)
	return nil
}

func finalizeDropOperation(cfg *config.WorkspaceConfig, cfgPath string, op dropOperation) error {
	if err := os.RemoveAll(op.bareAbs); err != nil {
		return fmt.Errorf("removing bare repo: %w", err)
	}

	for _, branch := range config.SortedWorktreeBranchNames(op.wt.Branches) {
		cfg.RemoveRepoFromManualGroups(config.WorktreeRepoName(op.setName, branch))
	}

	delete(cfg.Worktrees, op.setName)
	return config.Save(cfgPath, cfg)
}

func prepareDropOperation(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath, setName string) (dropOperation, error) {
	wt, err := lookupDropSet(cfg, setName)
	if err != nil {
		return dropOperation{}, err
	}

	force, _ := cmd.Flags().GetBool("force")
	bareAbs, err := bareAbsPath(cfgPath, wt)
	if err != nil {
		return dropOperation{}, err
	}

	return dropOperation{setName: setName, wt: wt, bareAbs: bareAbs, force: force}, nil
}

func executeDropOperation(ctx context.Context, cfgPath string, op dropOperation) error {
	if err := validateDropSafety(ctx, op.force, cfgPath, op.setName, op.wt); err != nil {
		return err
	}

	return removeDropWorktrees(ctx, cfgPath, op.bareAbs, op.wt, op.force)
}

func lookupDropSet(cfg *config.WorkspaceConfig, setName string) (config.WorktreeConfig, error) {
	wt, exists := cfg.Worktrees[setName]
	if !exists {
		return config.WorktreeConfig{}, fmt.Errorf("worktree set %q not found", setName)
	}

	return wt, nil
}

func validateDropSafety(ctx context.Context, force bool, cfgPath, setName string, wt config.WorktreeConfig) error {
	if force {
		return nil
	}

	violations, err := collectDropViolations(ctx, cfgPath, setName, wt)
	if err != nil {
		return err
	}

	if len(violations) == 0 {
		return nil
	}

	return fmt.Errorf("refusing to drop %q:\n%s", setName, strings.Join(violations, "\n"))
}

func removeDropWorktrees(ctx context.Context, cfgPath, bareAbs string, wt config.WorktreeConfig, force bool) error {
	for _, branch := range config.SortedWorktreeBranchNames(wt.Branches) {
		absPath, err := config.ResolveRepoPath(cfgPath, wt.Branches[branch])
		if err != nil {
			return err
		}

		if _, statErr := os.Stat(absPath); errors.Is(statErr, os.ErrNotExist) {
			continue // already removed by a prior attempt
		}

		if err := removeOneWorktree(ctx, bareAbs, absPath, force); err != nil {
			return err
		}
	}

	return nil
}

func collectDropViolations(ctx context.Context, cfgPath, setName string, wt config.WorktreeConfig) ([]string, error) {
	var violations []string
	for _, branch := range config.SortedWorktreeBranchNames(wt.Branches) {
		absPath, err := config.ResolveRepoPath(cfgPath, wt.Branches[branch])
		if err != nil {
			return nil, err
		}

		if _, statErr := os.Stat(absPath); errors.Is(statErr, os.ErrNotExist) {
			continue
		} else if statErr != nil {
			return nil, statErr
		}

		r := repo.Repo{Name: config.WorktreeRepoName(setName, branch), AbsPath: absPath}
		msgs, err := safetyViolations(ctx, r)
		if err != nil {
			return nil, err
		}

		for _, msg := range msgs {
			violations = append(violations, fmt.Sprintf("- %s: %s", r.Name, msg))
		}
	}

	return violations, nil
}
