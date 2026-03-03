package repo

import (
	"fmt"
	"slices"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/spf13/cobra"
)

func registerUnlink(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:   "unlink <name> [name...]",
		Short: "Unregister repos from the workspace",
		Long:  `Removes one or more repos from the .gitw config. Also removes them from any groups. Does not delete the actual repo directories.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runRemove,
	})
}

func runRemove(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	for _, name := range args {
		if err := removeRepo(cfg, name); err != nil {
			return err
		}

		output.Writef(cmd.OutOrStdout(), "Removed repo %q\n", name)
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
		before := len(g.Repos)
		g.Repos = slices.DeleteFunc(g.Repos, func(r string) bool {
			return r == name
		})

		if len(g.Repos) != before {
			cfg.Groups[gName] = g
		}
	}
}
