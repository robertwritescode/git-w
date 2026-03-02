package worktree

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/spf13/cobra"
)

type dropOperation struct {
	setName string
	wt      workspace.WorktreeConfig
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
	cfg, cfgPath, err := workspace.LoadConfig(cmd)
	if err != nil {
		return err
	}

	op, err := prepareDropOperation(cmd, cfg, cfgPath, args[0])
	if err != nil {
		return err
	}

	if err := executeDropOperation(cfgPath, op); err != nil {
		return err
	}

	if err := finalizeDropOperation(cfg, cfgPath, op); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Dropped worktree set %q\n", op.setName)
	return nil
}

func finalizeDropOperation(cfg *workspace.WorkspaceConfig, cfgPath string, op dropOperation) error {
	if err := os.RemoveAll(op.bareAbs); err != nil {
		return fmt.Errorf("removing bare repo: %w", err)
	}

	for _, branch := range workspace.SortedWorktreeBranchNames(op.wt.Branches) {
		cfg.RemoveRepoFromManualGroups(workspace.WorktreeRepoName(op.setName, branch))
	}

	delete(cfg.Worktrees, op.setName)
	return workspace.Save(cfgPath, cfg)
}

func prepareDropOperation(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath, setName string) (dropOperation, error) {
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

func executeDropOperation(cfgPath string, op dropOperation) error {
	if err := validateDropSafety(op.force, cfgPath, op.setName, op.wt); err != nil {
		return err
	}

	return removeDropWorktrees(cfgPath, op.bareAbs, op.wt, op.force)
}

func lookupDropSet(cfg *workspace.WorkspaceConfig, setName string) (workspace.WorktreeConfig, error) {
	wt, exists := cfg.Worktrees[setName]
	if !exists {
		return workspace.WorktreeConfig{}, fmt.Errorf("worktree set %q not found", setName)
	}

	return wt, nil
}

func validateDropSafety(force bool, cfgPath, setName string, wt workspace.WorktreeConfig) error {
	if force {
		return nil
	}

	violations, err := collectDropViolations(cfgPath, setName, wt)
	if err != nil {
		return err
	}

	if len(violations) == 0 {
		return nil
	}

	return fmt.Errorf("refusing to drop %q:\n%s", setName, strings.Join(violations, "\n"))
}

func removeDropWorktrees(cfgPath, bareAbs string, wt workspace.WorktreeConfig, force bool) error {
	for _, branch := range workspace.SortedWorktreeBranchNames(wt.Branches) {
		absPath, err := workspace.ResolveRepoPath(cfgPath, wt.Branches[branch])
		if err != nil {
			return err
		}

		if _, statErr := os.Stat(absPath); errors.Is(statErr, os.ErrNotExist) {
			continue // already removed by a prior attempt
		}

		if err := removeOneWorktree(bareAbs, absPath, force); err != nil {
			return err
		}
	}

	return nil
}

func collectDropViolations(cfgPath, setName string, wt workspace.WorktreeConfig) ([]string, error) {
	var violations []string
	for _, branch := range workspace.SortedWorktreeBranchNames(wt.Branches) {
		absPath, err := workspace.ResolveRepoPath(cfgPath, wt.Branches[branch])
		if err != nil {
			return nil, err
		}

		r := repo.Repo{Name: workspace.WorktreeRepoName(setName, branch), AbsPath: absPath}
		msgs, err := safetyViolations(r)
		if err != nil {
			return nil, err
		}

		for _, msg := range msgs {
			violations = append(violations, fmt.Sprintf("- %s: %s", r.Name, msg))
		}
	}

	return violations, nil
}
