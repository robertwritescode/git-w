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
func Output(repoPath string, args ...string) ([]byte, error) {
	out, err := exec.Command("git", append([]string{"-C", repoPath}, args...)...).Output()
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

// CloneBare runs `git clone --bare <url> <dest>` with context support for cancellation.
func CloneBare(ctx context.Context, url, dest string) error {
	out, err := exec.CommandContext(ctx, "git", "clone", "--bare", url, dest).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone --bare: %w\n%s", err, out)
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

// RemoveWorktree runs `git -C <barePath> worktree remove <treePath>`.
func RemoveWorktree(barePath, treePath string) error {
	out, err := exec.Command("git", "-C", barePath, "worktree", "remove", treePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove: %w\n%s", err, out)
	}

	return nil
}

// RemoveWorktreeForce runs `git -C <barePath> worktree remove --force <treePath>`.
func RemoveWorktreeForce(barePath, treePath string) error {
	out, err := exec.Command("git", "-C", barePath, "worktree", "remove", "--force", treePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove --force: %w\n%s", err, out)
	}

	return nil
}

// FetchBare runs `git -C <barePath> fetch`.
func FetchBare(barePath string) error {
	out, err := exec.Command("git", "-C", barePath, "fetch").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch (bare): %w\n%s", err, out)
	}

	return nil
}

// RemoteURL returns the origin remote URL of the repo at repoPath, or the
// empty string if no origin remote is configured.
func RemoteURL(repoPath string) string {
	out, err := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin").Output()
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
