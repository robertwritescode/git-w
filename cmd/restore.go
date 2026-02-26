package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Materialize all repos: clone missing, pull existing",
	Args:  cobra.NoArgs,
	RunE:  runRestore,
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}

func runRestore(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	return restoreAll(cmd, cfg, cfgPath)
}

func restoreAll(cmd *cobra.Command, cfg *config.WorkspaceConfig, cfgPath string) error {
	cfgDir := config.ConfigDir(cfgPath)
	gitignore := autoGitignoreEnabled(cfg)

	ctx := context.Background()
	var mu sync.Mutex
	var wg sync.WaitGroup
	var failures []string

	for name, rc := range cfg.Repos {
		wg.Go(func() {
			absPath := filepath.Join(cfgDir, rc.Path)
			msg, err := restoreRepo(ctx, rc, absPath)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				failures = append(failures, fmt.Sprintf("  [%s]: %v", name, err))
				fmt.Fprintf(cmd.ErrOrStderr(), "[%s] error: %v\n", name, err)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s\n", name, msg)
			}

			if err == nil && gitignore {
				if giErr := ensureGitignore(cfgDir, rc.Path); giErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "[%s] warning: .gitignore: %v\n", name, giErr)
				}
			}
		})
	}

	wg.Wait()

	if len(failures) > 0 {
		return fmt.Errorf("%d of %d repos failed:\n%s",
			len(failures), len(cfg.Repos), strings.Join(failures, "\n"))
	}

	return nil
}

func restoreRepo(ctx context.Context, rc config.RepoConfig, absPath string) (string, error) {
	if isGitRepo(absPath) {
		return restorePull(ctx, absPath)
	}

	if rc.URL == "" {
		return "skipped: no URL configured", nil
	}

	if err := cloneRepo(rc.URL, absPath); err != nil {
		return "", err
	}

	return "cloned", nil
}

func restorePull(ctx context.Context, absPath string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", absPath, "pull").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git pull: %w\n%s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func cloneRepo(url, destPath string) error {
	out, err := exec.Command("git", "clone", url, destPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %w\n%s", err, out)
	}
	return nil
}
