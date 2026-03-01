package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/robertwritescode/git-w/pkg/parallel"
	"github.com/robertwritescode/git-w/pkg/repo"
)

// ExecOptions configures a RunParallel invocation.
type ExecOptions struct {
	MaxConcurrency int           // 0 → runtime.NumCPU()
	Timeout        time.Duration // 0 → no timeout
	Async          bool          // false → serial run with stdin passthrough
}

// RunParallel executes git args in each repo.
// Async=false: stdin passes through; output not prefixed (one repo at a time).
// Async=true: stdin suppressed; output captured and prefixed "[name]".
func RunParallel(repos []repo.Repo, args []string, opts ExecOptions) []ExecResult {
	workers := parallel.MaxWorkers(optMaxConcurrency(opts), len(repos))
	ctx, cancel := buildContext(opts)
	defer cancel()

	if !opts.Async {
		results := make([]ExecResult, len(repos))
		for i, r := range repos {
			results[i] = runSerial(ctx, r, args)
		}
		return results
	}

	return parallel.RunFanOut(repos, workers, func(r repo.Repo) ExecResult {
		return runAsync(ctx, r, args)
	})
}

// buildCmd constructs a git command for the given repo and args,
// setting the working directory. The caller configures Stdin/Stdout/Stderr.
func buildCmd(ctx context.Context, r repo.Repo, args []string) *exec.Cmd {
	// Explicit copy avoids mutating r.Flags's backing array when appending args.
	gitArgs := make([]string, len(r.Flags)+len(args))
	copy(gitArgs, r.Flags)
	copy(gitArgs[len(r.Flags):], args)

	cmd := exec.CommandContext(ctx, "git", gitArgs...)
	cmd.Dir = r.AbsPath
	return cmd
}

// exitCode returns cmd's exit code after it has run, or -1 if unavailable.
func exitCode(cmd *exec.Cmd) int {
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	return -1
}

func optMaxConcurrency(opts ExecOptions) int {
	if opts.MaxConcurrency > 0 {
		return opts.MaxConcurrency
	}
	return runtime.NumCPU()
}

func buildContext(opts ExecOptions) (context.Context, context.CancelFunc) {
	if opts.Timeout > 0 {
		return context.WithTimeout(context.Background(), opts.Timeout)
	}
	return context.WithCancel(context.Background())
}

func runSerial(ctx context.Context, r repo.Repo, args []string) ExecResult {
	if ctx.Err() != nil {
		return ExecResult{RepoName: r.Name, Err: ctx.Err()}
	}
	cmd := buildCmd(ctx, r, args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return ExecResult{
		RepoName: r.Name,
		ExitCode: exitCode(cmd),
		Err:      err,
	}
}

// runAsync runs a command and prefixes its output with the repo name.
func runAsync(ctx context.Context, r repo.Repo, args []string) ExecResult {
	result := runOne(ctx, r, args)
	result.Stdout = prefixLines(r.Name, result.Stdout)
	result.Stderr = prefixLines(r.Name, result.Stderr)
	return result
}

func runOne(ctx context.Context, r repo.Repo, args []string) ExecResult {
	var stdout, stderr bytes.Buffer
	cmd := buildCmd(ctx, r, args)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return ExecResult{
		RepoName: r.Name,
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: exitCode(cmd),
		Err:      err,
	}
}
