package git

import (
	"fmt"
	"runtime"

	"github.com/robertwritescode/git-w/pkg/display"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/spf13/cobra"
)

func registerInfo(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:     "info [group]",
		Aliases: []string{"ll"},
		Short:   "Show status table for all repos (or a group)",
		Args:    cobra.MaximumNArgs(1),
		RunE:    runInfo,
	})
}

func runInfo(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	var repos []repo.Repo
	if len(args) == 0 {
		repos, err = repo.ForContext(cfg, cfgPath)
	} else {
		repos, err = repo.ForGroup(cfg, cfgPath, args[0])
	}
	if err != nil {
		return err
	}

	entries := collectStatuses(repos)
	display.RenderTable(cmd.OutOrStdout(), entries)

	return nil
}

func collectStatuses(repos []repo.Repo) []display.TableEntry {
	workers := parallel.MaxWorkers(runtime.NumCPU(), len(repos))
	return parallel.RunFanOut(repos, workers, func(r repo.Repo) display.TableEntry {
		status, err := repo.GetStatus(r)
		if err != nil {
			status = repo.RepoStatus{
				Branch:     "?",
				LastCommit: fmt.Sprintf("(error: %v)", err),
			}
		}

		return display.TableEntry{
			Name:        r.Name,
			Branch:      status.Branch,
			RemoteState: status.RemoteState,
			Dirty:       status.Dirty,
			Staged:      status.Staged,
			Untracked:   status.Untracked,
			Stashed:     status.Stashed,
			LastCommit:  status.LastCommit,
		}
	})
}
