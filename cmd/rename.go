package cmd

import (
	"fmt"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a tracked repo",
	Long:  `Renames a repo key in .gitworkspace and updates all group references.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runRename,
}

func init() {
	rootCmd.AddCommand(renameCmd)
}

func runRename(cmd *cobra.Command, args []string) error {
	oldName, newName := args[0], args[1]

	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	if err := renameRepo(cfg, oldName, newName); err != nil {
		return err
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Renamed repo %q → %q\n", oldName, newName)
	fmt.Fprintf(cmd.OutOrStdout(), "NOTE: Only the workspace key was renamed. The repo directory on disk has not moved.\n")
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

		for i, repo := range g.Repos {
			if repo == oldName {
				g.Repos[i] = newName
				changed = true
			}
		}

		if changed {
			cfg.Groups[gName] = g
		}
	}
}
