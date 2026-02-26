package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/robertwritescode/git-workspace/internal/executor"
	"github.com/robertwritescode/git-workspace/internal/repo"
	"github.com/spf13/cobra"
)

var execCommand = &cobra.Command{
	Use:   "exec [repos...] -- <git-args>",
	Short: "Execute an arbitrary git command across repos",
	Long: `Runs any git command in each registered repo concurrently.
Repo names before '--' filter targets; everything after '--' is passed to git.`,
	// DisableFlagParsing preserves "--" in args so splitExecArgs can split on it.
	DisableFlagParsing: true,
	RunE:               runExec,
}

func init() {
	rootCmd.AddCommand(execCommand)
}

func runExec(cmd *cobra.Command, args []string) error {
	repoNames, gitArgs := splitExecArgs(args)

	if len(gitArgs) == 0 {
		return fmt.Errorf("no git command specified; use: exec [repos...] -- <git-args>")
	}

	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	repos, err := filterRepos(cfg, cfgPath, repoNames)
	if err != nil {
		return err
	}

	opts := executor.ExecOptions{Async: true}
	results := executor.RunParallel(repos, gitArgs, opts)
	writeResults(cmd.OutOrStdout(), results)

	return execErrors(results)
}

func splitExecArgs(args []string) (repoNames, gitArgs []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return nil, args
}

// filterRepos resolves names as repo and/or group names.
// With no names: falls back to active context, then all repos.
// With names: expands group names to their repos, deduplicates by repo name.
func filterRepos(cfg *config.WorkspaceConfig, cfgPath string, names []string) ([]repo.Repo, error) {
	if len(names) == 0 {
		return reposForContext(cfg, cfgPath)
	}
	return resolveTargets(cfg, cfgPath, names)
}

// reposForContext returns the active context's repos, or all repos if no context is set.
func reposForContext(cfg *config.WorkspaceConfig, cfgPath string) ([]repo.Repo, error) {
	if cfg.Context.Active == "" {
		return repo.FromConfig(cfg, cfgPath), nil
	}

	g, ok := cfg.Groups[cfg.Context.Active]
	if !ok {
		return nil, fmt.Errorf("active context group %q not found", cfg.Context.Active)
	}

	return groupRepos(cfg, cfgPath, g), nil
}

// resolveTargets expands each name as a repo name or group name, deduplicating.
func resolveTargets(cfg *config.WorkspaceConfig, cfgPath string, names []string) ([]repo.Repo, error) {
	all := repo.FromConfig(cfg, cfgPath)
	byRepo := repoIndex(all)
	seen := make(map[string]bool)

	var result []repo.Repo
	for _, name := range names {
		if r, ok := byRepo[name]; ok {
			if !seen[r.Name] {
				seen[r.Name] = true
				result = append(result, r)
			}
			continue
		}

		if g, ok := cfg.Groups[name]; ok {
			for _, r := range groupRepos(cfg, cfgPath, g) {
				if !seen[r.Name] {
					seen[r.Name] = true
					result = append(result, r)
				}
			}
			continue
		}

		return nil, fmt.Errorf("%q is not a registered repo or group", name)
	}

	return result, nil
}

// groupRepos builds the Repo slice for repos listed in a GroupConfig.
func groupRepos(cfg *config.WorkspaceConfig, cfgPath string, g config.GroupConfig) []repo.Repo {
	sub := &config.WorkspaceConfig{
		Repos:  make(map[string]config.RepoConfig, len(g.Repos)),
		Groups: make(map[string]config.GroupConfig),
	}

	for _, name := range g.Repos {
		if rc, ok := cfg.Repos[name]; ok {
			sub.Repos[name] = rc
		}
	}

	return repo.FromConfig(sub, cfgPath)
}

// repoIndex builds a name → Repo lookup map.
func repoIndex(repos []repo.Repo) map[string]repo.Repo {
	m := make(map[string]repo.Repo, len(repos))
	for _, r := range repos {
		m[r.Name] = r
	}
	return m
}

func writeResults(w io.Writer, results []executor.ExecResult) {
	for _, r := range results {
		w.Write(r.Stdout) //nolint:errcheck
		w.Write(r.Stderr) //nolint:errcheck
	}
}

func execErrors(results []executor.ExecResult) error {
	var failures []string
	for _, r := range results {
		if r.ExitCode != 0 || r.Err != nil {
			failures = append(failures, "  ["+r.RepoName+"]: "+failureMessage(r))
		}
	}
	if len(failures) == 0 {
		return nil
	}
	return fmt.Errorf("%d of %d repos failed:\n%s",
		len(failures), len(results), strings.Join(failures, "\n"))
}

// failureMessage returns a human-readable reason for a failed ExecResult.
func failureMessage(r executor.ExecResult) string {
	if r.Err != nil {
		return r.Err.Error()
	}
	return fmt.Sprintf("exit %d", r.ExitCode)
}
