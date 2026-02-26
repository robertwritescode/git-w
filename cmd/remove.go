package cmd

import (
	"fmt"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "rm <name> [name...]",
	Aliases: []string{"remove"},
	Short:   "Unregister repos from the workspace",
	Long:    `Removes one or more repos from the .gitworkspace config. Also removes them from any groups.`,
	Args:    cobra.MinimumNArgs(1),
	RunE:    runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	for _, name := range args {
		if err := removeRepo(cfg, name); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Removed repo %q\n", name)
	}

	return config.Save(cfgPath, cfg)
}

func removeRepo(cfg *config.WorkspaceConfig, name string) error {
	if _, exists := cfg.Repos[name]; !exists {
		return fmt.Errorf("repo %q not found", name)
	}
	delete(cfg.Repos, name)
	removeRepoFromGroups(cfg, name)
	return nil
}

func removeRepoFromGroups(cfg *config.WorkspaceConfig, name string) {
	for gName, g := range cfg.Groups {
		found := false
		for _, r := range g.Repos {
			if r == name {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		var filtered []string
		for _, r := range g.Repos {
			if r != name {
				filtered = append(filtered, r)
			}
		}
		g.Repos = filtered
		cfg.Groups[gName] = g
	}
}
