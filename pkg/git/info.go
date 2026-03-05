package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/display"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
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

	entries := collectStatuses(cmd.Context(), repos)
	sets := buildWorktreeSets(cfg)
	display.RenderGroupedTable(cmd.OutOrStdout(), entries, sets)

	sections := collectWorkgroupSections(cmd.Context(), cfg, cfgPath)
	if len(sections) > 0 {
		output.Writef(cmd.OutOrStdout(), "\n")
		display.RenderWorkgroupTable(cmd.OutOrStdout(), sections)
	}

	return nil
}

func buildWorktreeSets(cfg *config.WorkspaceConfig) []display.WorktreeSet {
	sets := make([]display.WorktreeSet, 0, len(cfg.Worktrees))

	for _, name := range config.SortedStringKeys(cfg.Worktrees) {
		branches := config.SortedWorktreeBranchNames(cfg.Worktrees[name].Branches)
		sets = append(sets, display.WorktreeSet{SetName: name, Branches: branches})
	}

	return sets
}

func collectStatuses(ctx context.Context, repos []repo.Repo) []display.TableEntry {
	workers := parallel.MaxWorkers(runtime.NumCPU(), len(repos))
	return parallel.RunFanOut(repos, workers, func(r repo.Repo) display.TableEntry {
		status, err := repo.GetStatus(ctx, r)
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

func collectWorkgroupSections(ctx context.Context, cfg *config.WorkspaceConfig, cfgPath string) []display.WorkgroupSection {
	sections := make([]display.WorkgroupSection, 0)

	for _, name := range config.SortedStringKeys(cfg.Workgroups) {
		wg := cfg.Workgroups[name]
		wgRepos := workgroupRepos(cfgPath, name, wg)
		if len(wgRepos) == 0 {
			continue
		}

		entries := collectStatuses(ctx, wgRepos)
		sections = append(sections, display.WorkgroupSection{Name: name, Entries: entries})
	}

	return sections
}

func workgroupRepos(cfgPath, wgName string, wg config.WorkgroupConfig) []repo.Repo {
	repos := make([]repo.Repo, 0)

	for _, repoName := range wg.Repos {
		path := workgroupWorktreePath(cfgPath, wgName, repoName)
		if _, err := os.Stat(path); err != nil && errors.Is(err, os.ErrNotExist) {
			continue
		}

		repos = append(repos, repo.Repo{Name: repoName, AbsPath: path})
	}

	return repos
}

func workgroupWorktreePath(cfgPath, wgName, repoName string) string {
	return filepath.Join(config.ConfigDir(cfgPath), ".workgroup", wgName, repoName)
}
