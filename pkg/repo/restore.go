package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/spf13/cobra"
)

// restoreInput pairs a repo name with its config for fan-out processing.
type restoreInput struct {
	Name     string
	Repo     config.RepoConfig
	Worktree config.WorktreeConfig
	IsRepo   bool
	RelPaths []string
}

// restoreResult captures the outcome of restoring a single repo.
type restoreResult struct {
	Name     string
	RelPaths []string
	Msg      string
	Err      error
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
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	jobs, _ := cmd.Flags().GetInt("jobs")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	return restoreAll(cmd, cfg, cfgPath, jobs, timeout)
}

func restoreAll(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath string, jobs int, timeout time.Duration) error {
	cfgDir := config.ConfigDir(cfgPath)
	gitignore := cfg.AutoGitignoreEnabled()

	ctx, stop := newRestoreContext(timeout)
	defer stop()

	inputs := buildRestoreInputs(cfg)
	workers := parallel.MaxWorkers(jobs, len(inputs))

	results := parallel.RunFanOut(inputs, workers, func(in restoreInput) restoreResult {
		msg, err := processRestore(ctx, cfgPath, in)
		return restoreResult{Name: in.Name, RelPaths: in.RelPaths, Msg: msg, Err: err}
	})

	return reportRestoreResults(cmd, results, cfgDir, gitignore)
}

func newRestoreContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	if timeout <= 0 {
		return ctx, stop
	}

	ctxWithTimeout, cancelTimeout := context.WithTimeout(ctx, timeout)
	return ctxWithTimeout, func() {
		cancelTimeout()
		stop()
	}
}

func buildRestoreInputs(cfg *config.WorkspaceConfig) []restoreInput {
	inputs := make([]restoreInput, 0, len(cfg.Repos)+len(cfg.Worktrees))

	synthNames := make(map[string]struct{})
	for setName, wt := range cfg.Worktrees {
		for branch := range wt.Branches {
			synthNames[config.WorktreeRepoName(setName, branch)] = struct{}{}
		}
	}
	for _, name := range config.SortedStringKeys(cfg.Repos) {
		if _, isSynth := synthNames[name]; isSynth {
			continue
		}
		rc := cfg.Repos[name]
		inputs = append(inputs, restoreInput{Name: name, Repo: rc, IsRepo: true, RelPaths: []string{rc.Path}})
	}

	for _, setName := range config.SortedStringKeys(cfg.Worktrees) {
		wt := cfg.Worktrees[setName]
		relPaths := make([]string, 0, len(wt.Branches))
		for _, branch := range config.SortedStringKeys(wt.Branches) {
			relPaths = append(relPaths, wt.Branches[branch])
		}

		inputs = append(inputs, restoreInput{
			Name:     setName,
			Worktree: wt,
			RelPaths: relPaths,
		})
	}

	return inputs
}

func reportRestoreResults(cmd *cobra.Command, results []restoreResult, cfgDir string, gitignore bool) error {
	var failures []string
	for _, r := range results {
		if r.Err != nil {
			reportRestoreError(cmd, r)
			failures = append(failures, fmt.Sprintf("  [%s]: %v", r.Name, r.Err))
			continue
		}

		reportRestoreSuccess(cmd, r)
		applyRestoreGitignore(cmd, cfgDir, gitignore, r)
	}

	return parallel.FormatFailureError(failures, len(results))
}

func reportRestoreError(cmd *cobra.Command, r restoreResult) {
	output.Writef(cmd.ErrOrStderr(), "[%s] error: %v\n", r.Name, r.Err)
}

func reportRestoreSuccess(cmd *cobra.Command, r restoreResult) {
	output.Writef(cmd.OutOrStdout(), "[%s] %s\n", r.Name, r.Msg)
}

func applyRestoreGitignore(cmd *cobra.Command, cfgDir string, gitignore bool, r restoreResult) {
	if !gitignore {
		return
	}

	for _, relPath := range r.RelPaths {
		if giErr := gitutil.EnsureGitignore(cfgDir, relPath); giErr != nil {
			output.Writef(cmd.ErrOrStderr(), "[%s] warning: .gitignore (%s): %v\n", r.Name, relPath, giErr)
		}
	}
}

func processRestore(ctx context.Context, cfgPath string, in restoreInput) (string, error) {
	if in.IsRepo {
		return processRepoRestore(ctx, cfgPath, in.Repo)
	}

	if in.Worktree.BarePath != "" {
		return processWorktreeRestore(ctx, cfgPath, in.Worktree)
	}

	return "", fmt.Errorf("invalid restore input")
}

func processRepoRestore(ctx context.Context, cfgPath string, rc config.RepoConfig) (string, error) {
	absPath, err := config.ResolveRepoPath(cfgPath, rc.Path)
	if err != nil {
		return "", err
	}

	return restoreRepo(ctx, rc, absPath)
}

func restoreRepo(ctx context.Context, rc config.RepoConfig, absPath string) (string, error) {
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

func processWorktreeRestore(ctx context.Context, cfgPath string, wt config.WorktreeConfig) (string, error) {
	bareAbsPath, err := config.ResolveRepoPath(cfgPath, wt.BarePath)
	if err != nil {
		return "", err
	}

	skipped, err := ensureBareForWorktree(ctx, wt, bareAbsPath)
	if err != nil {
		return "", err
	}

	if skipped {
		return "skipped: no URL configured", nil
	}

	if err := gitutil.ConfigureBareOriginTracking(ctx, bareAbsPath); err != nil {
		return "", err
	}

	added, pulled, err := restoreWorktreeBranches(ctx, cfgPath, bareAbsPath, wt)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("worktrees: added %d, pulled %d", added, pulled), nil
}

// restoreWorktreeBranches restores branches sequentially within a single
// worktree set. Inter-set parallelism is handled by the RunFanOut caller.
// Intra-set parallelism is not needed for typical 2-5 branches per set.
func restoreWorktreeBranches(ctx context.Context, cfgPath, bareAbsPath string, wt config.WorktreeConfig) (int, int, error) {
	added := 0
	pulled := 0

	for _, branch := range config.SortedStringKeys(wt.Branches) {
		addedOne, pulledOne, err := restoreWorktreeBranch(ctx, cfgPath, bareAbsPath, branch, wt.Branches[branch])
		if err != nil {
			return 0, 0, err
		}

		if addedOne {
			added++
		}

		if pulledOne {
			pulled++
		}
	}

	return added, pulled, nil
}

func ensureBareForWorktree(ctx context.Context, wt config.WorktreeConfig, bareAbsPath string) (bool, error) {
	if _, err := os.Stat(bareAbsPath); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	if wt.URL == "" {
		return true, nil
	}

	if err := gitutil.CloneBare(ctx, wt.URL, bareAbsPath); err != nil {
		return false, err
	}

	return false, nil
}

func restoreWorktreeBranch(ctx context.Context, cfgPath, bareAbsPath, branch, branchPath string) (bool, bool, error) {
	absPath, err := config.ResolveRepoPath(cfgPath, branchPath)
	if err != nil {
		return false, false, err
	}

	if IsGitRepo(absPath) {
		return restoreExistingWorktree(ctx, absPath, branch)
	}

	if err := gitutil.AddWorktree(ctx, bareAbsPath, absPath, branch); err != nil {
		return false, false, err
	}

	if err := gitutil.SetBranchTrackingToOrigin(ctx, absPath, branch); err != nil {
		return false, false, err
	}

	return true, false, nil
}

func restoreExistingWorktree(ctx context.Context, absPath, branch string) (bool, bool, error) {
	if err := gitutil.SetBranchTrackingToOrigin(ctx, absPath, branch); err != nil {
		return false, false, err
	}

	if _, err := gitutil.Pull(ctx, absPath); err != nil {
		return false, false, err
	}

	return false, true, nil
}
