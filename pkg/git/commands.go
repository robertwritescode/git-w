package git

import (
	"fmt"
	"slices"
	"strings"

	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/spf13/cobra"
)

// registerGit adds all git execution commands to root.
func registerGit(root *cobra.Command) {
	root.AddCommand(
		&cobra.Command{
			Use:     "fetch [repos...]",
			Aliases: []string{"f"},
			Short:   "Run git fetch in repos",
			RunE:    runFetch,
		},
		&cobra.Command{
			Use:     "pull [repos...]",
			Aliases: []string{"pl"},
			Short:   "Run git pull in repos",
			RunE:    runPull,
		},
		&cobra.Command{
			Use:     "push [repos...]",
			Aliases: []string{"ps"},
			Short:   "Run git push in repos",
			RunE:    runPush,
		},
		&cobra.Command{
			Use:     "status [repos...]",
			Aliases: []string{"st"},
			Short:   "Run git status -sb in repos",
			RunE:    runStatus,
		},
	)
}

func runFetch(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, repos, err := loadFetchInputs(cmd, args)
	if err != nil {
		return err
	}

	if len(repos) <= 1 {
		return runGitCmd(cmd, args, "fetch")
	}

	if setName, ok := singleWorktreeSet(cfg, repos); ok {
		return fetchSetBare(cmd, cfgPath, setName, cfg.Worktrees[setName])
	}

	if len(args) == 0 {
		return fetchWithAllRepoDedup(cmd, cfg, cfgPath, repos)
	}

	return runGitCmd(cmd, args, "fetch")
}

func loadFetchInputs(cmd *cobra.Command, args []string) (*workspace.WorkspaceConfig, string, []repo.Repo, error) {
	cfg, cfgPath, err := workspace.LoadConfig(cmd)
	if err != nil {
		return nil, "", nil, err
	}

	repos, err := repo.Filter(cfg, cfgPath, args)
	if err != nil {
		return nil, "", nil, err
	}

	return cfg, cfgPath, repos, nil
}

func runPull(cmd *cobra.Command, args []string) error {
	return runGitCmd(cmd, args, "pull")
}

func runPush(cmd *cobra.Command, args []string) error {
	return runGitCmd(cmd, args, "push")
}

func runStatus(cmd *cobra.Command, args []string) error {
	return runGitCmd(cmd, args, "status", "-sb")
}

func singleWorktreeSet(cfg *workspace.WorkspaceConfig, repos []repo.Repo) (string, bool) {
	byRepo := worktreeRepoToSet(cfg)
	setName := ""

	for _, r := range repos {
		currentSet, isWorktree := byRepo[r.Name]
		if !isWorktree {
			return "", false
		}

		if setName == "" {
			setName = currentSet
			continue
		}

		if setName != currentSet {
			return "", false
		}
	}

	return setName, setName != ""
}

func fetchSetBare(cmd *cobra.Command, cfgPath, setName string, wt workspace.WorktreeConfig) error {
	bareAbsPath, err := workspace.ResolveRepoPath(cfgPath, wt.BarePath)
	if err != nil {
		return err
	}

	if err := gitutil.FetchBare(bareAbsPath); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] fetch\n", setName)
	return nil
}

func fetchWithAllRepoDedup(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath string, repos []repo.Repo) error {
	nonWorktree, setToBare, err := collectFetchTargets(cfg, cfgPath, repos)
	if err != nil {
		return err
	}

	var failures []string
	failures = append(failures, fetchWorktreeBareTargets(cmd, setToBare)...)
	failures = append(failures, fetchNonWorktreeTargets(cmd, nonWorktree)...)

	if len(failures) == 0 {
		return nil
	}

	return fmt.Errorf("%s", strings.Join(failures, "\n"))
}

func collectFetchTargets(cfg *workspace.WorkspaceConfig, cfgPath string, repos []repo.Repo) ([]repo.Repo, map[string][]string, error) {
	byRepo := worktreeRepoToSet(cfg)
	nonWorktree := make([]repo.Repo, 0, len(repos))
	setToBare := make(map[string][]string)

	for _, r := range repos {
		setName, isWorktree := byRepo[r.Name]
		if !isWorktree {
			nonWorktree = append(nonWorktree, r)
			continue
		}

		wt := cfg.Worktrees[setName]
		bareAbsPath, err := workspace.ResolveRepoPath(cfgPath, wt.BarePath)
		if err != nil {
			return nil, nil, err
		}

		setToBare[setName] = append(setToBare[setName], bareAbsPath)
	}

	return nonWorktree, dedupeSetsByBare(setToBare), nil
}

func dedupeSetsByBare(setToBare map[string][]string) map[string][]string {
	bareToSets := make(map[string][]string)
	for setName, barePaths := range setToBare {
		for _, barePath := range barePaths {
			bareToSets[barePath] = append(bareToSets[barePath], setName)
		}
	}

	for barePath := range bareToSets {
		setNames := bareToSets[barePath]
		slices.Sort(setNames)
		bareToSets[barePath] = slices.Compact(setNames)
	}

	return bareToSets
}

func fetchWorktreeBareTargets(cmd *cobra.Command, bareToSets map[string][]string) []string {
	bars := workspace.SortedStringKeys(bareToSets)
	failures := make([]string, 0)

	for _, bareAbsPath := range bars {
		setNames := bareToSets[bareAbsPath]
		label := strings.Join(setNames, ",")
		if err := fetchBarePath(cmd, bareAbsPath, label); err != nil {
			failures = append(failures, fmt.Sprintf("  [%s]: %v", label, err))
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "[%s] error: %v\n", label, err)
		}
	}

	return failures
}

func fetchBarePath(cmd *cobra.Command, bareAbsPath, label string) error {
	if err := gitutil.FetchBare(bareAbsPath); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] fetch\n", label)
	return nil
}

func fetchNonWorktreeTargets(cmd *cobra.Command, repos []repo.Repo) []string {
	if len(repos) == 0 {
		return nil
	}

	results := RunParallel(repos, []string{"fetch"}, ExecOptions{Async: len(repos) > 1})
	WriteResults(cmd.OutOrStdout(), results)

	if err := ExecErrors(results); err != nil {
		return []string{strings.TrimSpace(err.Error())}
	}

	return nil
}

func worktreeRepoToSet(cfg *workspace.WorkspaceConfig) map[string]string {
	result := make(map[string]string)
	for setName, wt := range cfg.Worktrees {
		for _, branch := range workspace.SortedStringKeys(wt.Branches) {
			result[workspace.WorktreeRepoName(setName, branch)] = setName
		}
	}

	return result
}
