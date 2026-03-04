package gitutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var gitignoreMu sync.Mutex

// Output runs a git command in repoPath and returns its stdout.
// On failure it returns stderr (if available) alongside the error.
func Output(ctx context.Context, repoPath string, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, "git", append([]string{"-C", repoPath}, args...)...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.Stderr, err
		}
		return nil, err
	}

	return out, nil
}

// Clone runs `git clone <url> <destPath>` with context support for cancellation.
func Clone(ctx context.Context, url, destPath string) error {
	out, err := exec.CommandContext(ctx, "git", "clone", url, destPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %w\n%s", err, out)
	}
	return nil
}

// Pull runs `git pull` in repoPath with context support for cancellation.
func Pull(ctx context.Context, repoPath string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "pull").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git pull: %w\n%s", err, out)
	}

	return strings.TrimSpace(string(out)), nil
}

// CheckoutBranch runs `git checkout <branch>` in repoPath.
func CheckoutBranch(ctx context.Context, repoPath, branch string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout: %w\n%s", err, out)
	}

	return nil
}

// FetchOrigin runs `git fetch origin` in repoPath.
func FetchOrigin(ctx context.Context, repoPath string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "fetch", "origin").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch origin: %w\n%s", err, out)
	}

	return nil
}

// PullBranch runs `git pull origin <branch>` in repoPath.
func PullBranch(ctx context.Context, repoPath, branch string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "pull", "origin", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull origin %s: %w\n%s", branch, err, out)
	}

	return nil
}

// BranchExists reports whether branchName exists locally in repoPath.
func BranchExists(ctx context.Context, repoPath, branchName string) (bool, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "--list", branchName).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git branch --list: %w\n%s", err, out)
	}

	return strings.TrimSpace(string(out)) != "", nil
}

// CurrentBranch returns the current branch name in repoPath.
func CurrentBranch(ctx context.Context, repoPath string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --abbrev-ref HEAD: %w\n%s", err, out)
	}

	return strings.TrimSpace(string(out)), nil
}

// CreateBranch runs `git branch <branchName> <sourceBranch>` in repoPath.
func CreateBranch(ctx context.Context, repoPath, branchName, sourceBranch string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", branchName, sourceBranch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch %s %s: %w\n%s", branchName, sourceBranch, err, out)
	}

	return nil
}

// PushBranchUpstream runs `git push -u <remote> <branchName>`.
func PushBranchUpstream(ctx context.Context, repoPath, remote, branchName string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "push", "-u", remote, branchName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push -u %s %s: %w\n%s", remote, branchName, err, out)
	}

	return nil
}

// SetBranchUpstream runs `git branch --set-upstream-to=<remote>/<branchName>`.
func SetBranchUpstream(ctx context.Context, repoPath, branchName, remote string) error {
	upstream := remote + "/" + branchName
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "--set-upstream-to="+upstream, branchName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch --set-upstream-to: %w\n%s", err, out)
	}

	return nil
}

// CloneBare runs `git clone --bare <url> <dest>` with context support for cancellation.
func CloneBare(ctx context.Context, url, dest string) error {
	out, err := exec.CommandContext(ctx, "git", "clone", "--bare", url, dest).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone --bare: %w\n%s", err, out)
	}

	return nil
}

// ConfigureBareOriginTracking ensures origin fetches branch heads into
// refs/remotes/origin/* and fetches the latest refs.
func ConfigureBareOriginTracking(ctx context.Context, barePath string) error {
	if out, err := exec.CommandContext(ctx, "git", "-C", barePath, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*").CombinedOutput(); err != nil {
		return fmt.Errorf("git config remote.origin.fetch: %w\n%s", err, out)
	}

	if out, err := exec.CommandContext(ctx, "git", "-C", barePath, "fetch", "origin").CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch origin: %w\n%s", err, out)
	}

	return nil
}

// AddWorktree runs `git -C <barePath> worktree add <treePath> <branch>` with context support for cancellation.
func AddWorktree(ctx context.Context, barePath, treePath, branch string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", barePath, "worktree", "add", treePath, branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add: %w\n%s", err, out)
	}

	return nil
}

// SetBranchTrackingToOrigin sets branch upstream to origin/<branch> in treePath.
func SetBranchTrackingToOrigin(ctx context.Context, treePath, branch string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", treePath, "branch", "--set-upstream-to=origin/"+branch, branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch --set-upstream-to: %w\n%s", err, out)
	}

	return nil
}

// RemoveWorktree runs `git -C <barePath> worktree remove <treePath>`.
func RemoveWorktree(ctx context.Context, barePath, treePath string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", barePath, "worktree", "remove", treePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove: %w\n%s", err, out)
	}

	return nil
}

// RemoveWorktreeForce runs `git -C <barePath> worktree remove --force <treePath>`.
func RemoveWorktreeForce(ctx context.Context, barePath, treePath string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", barePath, "worktree", "remove", "--force", treePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove --force: %w\n%s", err, out)
	}

	return nil
}

// FetchBare runs `git -C <barePath> fetch`.
func FetchBare(ctx context.Context, barePath string) error {
	out, err := exec.CommandContext(ctx, "git", "-C", barePath, "fetch").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch (bare): %w\n%s", err, out)
	}

	return nil
}

// RemoteURL returns the origin remote URL of the repo at repoPath, or the
// empty string if no origin remote is configured.
func RemoteURL(ctx context.Context, repoPath string) string {
	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// EnsureGitignore appends entry to the .gitignore in dir if not already present.
// It is safe to call concurrently from multiple goroutines.
func EnsureGitignore(dir, entry string) error {
	gitignoreMu.Lock()
	defer gitignoreMu.Unlock()
	return ensureGitignore(dir, entry)
}

func ensureGitignore(dir, entry string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")

	existing, err := readGitignore(gitignorePath)
	if err != nil {
		return err
	}

	if gitignoreContains(existing, entry) {
		return nil
	}

	return appendGitignoreEntry(gitignorePath, existing, entry)
}

func readGitignore(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return data, nil
}

func gitignoreContains(content []byte, entry string) bool {
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == entry {
			return true
		}
	}

	return false
}

func appendGitignoreEntry(path string, existing []byte, entry string) (err error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err := ensureTrailingNewline(f, existing); err != nil {
		return err
	}

	_, err = fmt.Fprintln(f, entry)
	return err
}

func ensureTrailingNewline(f *os.File, existing []byte) error {
	if len(existing) == 0 || strings.HasSuffix(string(existing), "\n") {
		return nil
	}

	_, err := f.WriteString("\n")
	return err
}
