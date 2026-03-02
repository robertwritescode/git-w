package repo

import (
	"fmt"

	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/spf13/cobra"
)

func registerRename(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a tracked repo",
		Long:  `Renames a repo key in .gitw and updates all group references.`,
		Args:  cobra.ExactArgs(2),
		RunE:  runRename,
	})
}

func runRename(cmd *cobra.Command, args []string) error {
	oldName, newName := args[0], args[1]

	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	if err := renameRepo(cfg, oldName, newName); err != nil {
		return err
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Renamed repo %q → %q\nNOTE: Only the workspace key was renamed. The repo directory on disk has not moved.\n", oldName, newName)
	return nil
}

func renameRepo(cfg *config.WorkspaceConfig, oldName, newName string) error {
	repoConfig, exists := cfg.Repos[oldName]
	if !exists {
		return fmt.Errorf("repo %q not found", oldName)
	}

	if _, exists := cfg.Repos[newName]; exists {
		return fmt.Errorf("repo %q already exists", newName)
	}

	delete(cfg.Repos, oldName)
	cfg.Repos[newName] = repoConfig
	renameRepoInGroups(cfg, oldName, newName)

	return nil
}

func renameRepoInGroups(cfg *config.WorkspaceConfig, oldName, newName string) {
	for gName, g := range cfg.Groups {
		changed := false

		for i, r := range g.Repos {
			if r == oldName {
				g.Repos[i] = newName
				changed = true
			}
		}

		if changed {
			cfg.Groups[gName] = g
		}
	}
}
