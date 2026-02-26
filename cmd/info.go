package cmd

import (
	"fmt"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/robertwritescode/git-workspace/internal/display"
	"github.com/robertwritescode/git-workspace/internal/repo"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:     "info [group]",
	Aliases: []string{"ll"},
	Short:   "Show status table for all repos (or a group)",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	repos, err := resolveRepos(cfg, cfgPath, args)
	if err != nil {
		return err
	}

	entries := collectStatuses(repos)
	display.RenderTable(cmd.OutOrStdout(), entries)

	return nil
}

func resolveRepos(cfg *config.WorkspaceConfig, cfgPath string, args []string) ([]repo.Repo, error) {
	if len(args) == 0 {
		return reposForContext(cfg, cfgPath)
	}
	g, ok := cfg.Groups[args[0]]
	if !ok {
		return nil, fmt.Errorf("group %q not found", args[0])
	}
	return groupRepos(cfg, cfgPath, g), nil
}

func collectStatuses(repos []repo.Repo) []display.TableEntry {
	entries := make([]display.TableEntry, len(repos))

	for i, r := range repos {
		status, err := repo.GetStatus(r)
		if err != nil {
			status = repo.RepoStatus{LastCommit: "(error)"}
		}
		entries[i] = display.TableEntry{Name: r.Name, Status: status}
	}

	return entries
}
