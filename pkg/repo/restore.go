package repo

import (
	"context"
	"fmt"
	"maps"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/spf13/cobra"
)

// restoreInput pairs a repo name with its config for fan-out processing.
type restoreInput struct {
	Name string
	RC   workspace.RepoConfig
}

// restoreResult captures the outcome of restoring a single repo.
type restoreResult struct {
	Name    string
	RelPath string
	Msg     string
	Err     error
}

func registerRestore(root *cobra.Command) {
	restoreCmd := &cobra.Command{
		Use:   "restore",
		Short: "Materialize all repos: clone missing, pull existing",
		Args:  cobra.NoArgs,
		RunE:  runRestore,
	}
	restoreCmd.Flags().IntP("jobs", "j", 0, "maximum number of concurrent restore operations (default: number of CPUs)")
	restoreCmd.Flags().Duration("timeout", 0, "overall timeout for restore (e.g. 30s, 2m); 0 disables timeout")
	root.AddCommand(restoreCmd)
}

func runRestore(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := workspace.LoadConfig(cmd)
	if err != nil {
		return err
	}

	jobs, _ := cmd.Flags().GetInt("jobs")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	return restoreAll(cmd, cfg, cfgPath, jobs, timeout)
}

func restoreAll(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath string, jobs int, timeout time.Duration) error {
	cfgDir := workspace.ConfigDir(cfgPath)
	gitignore := cfg.AutoGitignoreEnabled()

	// Cancel all in-flight clones/pulls on SIGINT or SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	inputs := buildRestoreInputs(cfg)
	workers := parallel.MaxWorkers(jobs, len(inputs))

	results := parallel.RunFanOut(inputs, workers, func(in restoreInput) restoreResult {
		msg, err := processRestore(ctx, cfgPath, in.RC)
		return restoreResult{Name: in.Name, RelPath: in.RC.Path, Msg: msg, Err: err}
	})

	return reportRestoreResults(cmd, results, cfgDir, gitignore)
}

func buildRestoreInputs(cfg *workspace.WorkspaceConfig) []restoreInput {
	inputs := make([]restoreInput, 0, len(cfg.Repos))

	for _, name := range slices.Sorted(maps.Keys(cfg.Repos)) {
		inputs = append(inputs, restoreInput{Name: name, RC: cfg.Repos[name]})
	}

	return inputs
}

func reportRestoreResults(cmd *cobra.Command, results []restoreResult, cfgDir string, gitignore bool) error {
	var failures []string
	for _, r := range results {
		if r.Err != nil {
			failures = append(failures, fmt.Sprintf("  [%s]: %v", r.Name, r.Err))
			writef(cmd.ErrOrStderr(), "[%s] error: %v\n", r.Name, r.Err)
			continue
		}

		writef(cmd.OutOrStdout(), "[%s] %s\n", r.Name, r.Msg)

		if gitignore {
			if giErr := gitutil.EnsureGitignore(cfgDir, r.RelPath); giErr != nil {
				writef(cmd.ErrOrStderr(), "[%s] warning: .gitignore: %v\n", r.Name, giErr)
			}
		}
	}

	return parallel.FormatFailureError(failures, len(results))
}

func processRestore(ctx context.Context, cfgPath string, rc workspace.RepoConfig) (string, error) {
	absPath, err := workspace.ResolveRepoPath(cfgPath, rc.Path)
	if err != nil {
		return "", err
	}

	return restoreRepo(ctx, rc, absPath)
}

func restoreRepo(ctx context.Context, rc workspace.RepoConfig, absPath string) (string, error) {
	if IsGitRepo(absPath) {
		return gitutil.Pull(ctx, absPath)
	}

	if rc.URL == "" {
		return "skipped: no URL configured", nil
	}

	if err := gitutil.Clone(ctx, rc.URL, absPath); err != nil {
		return "", err
	}

	return "cloned", nil
}
