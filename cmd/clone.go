package cmd

import (
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/spf13/cobra"
)

var cloneGroup string

var cloneCmd = &cobra.Command{
	Use:   "clone <url> [<path>]",
	Short: "Clone a remote repo and register it in the workspace",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runClone,
}

func init() {
	rootCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().StringVarP(&cloneGroup, "group", "g", "", "add cloned repo to this group")
}

func runClone(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	destPath, err := resolveCloneDest(args, cfgPath)
	if err != nil {
		return err
	}

	name, err := resolveRepoName(cfg, destPath)
	if err != nil {
		return err
	}

	if err := gitClone(args[0], destPath); err != nil {
		return err
	}

	relPath, err := computeRelPath(cfgPath, destPath)
	if err != nil {
		return err
	}

	cfg.Repos[name] = config.RepoConfig{Path: relPath, URL: args[0]}

	if autoGitignoreEnabled(cfg) {
		if err := ensureGitignore(config.ConfigDir(cfgPath), relPath); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not update .gitignore: %v\n", err)
		}
	}

	if cloneGroup != "" {
		addRepoToGroup(cfg, cloneGroup, name)
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Cloned %q (%s)\n", name, relPath)
	return nil
}

func resolveCloneDest(args []string, cfgPath string) (string, error) {
	if len(args) >= 2 {
		return filepath.Abs(args[1])
	}
	return filepath.Join(config.ConfigDir(cfgPath), deriveClonePath(args[0])), nil
}

func deriveClonePath(rawURL string) string {
	base := path.Base(rawURL)
	return strings.TrimSuffix(base, ".git")
}

func gitClone(url, destPath string) error {
	out, err := exec.Command("git", "clone", url, destPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %w\n%s", err, out)
	}
	return nil
}
