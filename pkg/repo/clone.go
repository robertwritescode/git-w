package repo

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/spf13/cobra"
)

func registerClone(root *cobra.Command) {
	cloneCmd := &cobra.Command{
		Use:   "clone <url> [<path>]",
		Short: "Clone a remote repo and register it in the workspace",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runClone,
	}
	cloneCmd.Flags().StringP("group", "g", "", "add cloned repo to this group")
	root.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := workspace.LoadConfig(cmd)
	if err != nil {
		return err
	}

	group, _ := cmd.Flags().GetString("group")

	destPath, name, err := resolveCloneTarget(cfg, args, cfgPath)
	if err != nil {
		return err
	}

	if err := gitutil.Clone(context.Background(), args[0], destPath); err != nil {
		return err
	}

	relPath, err := registerClonedRepo(cmd, cfg, cfgPath, destPath, name, args[0], group)
	if err != nil {
		_ = os.RemoveAll(destPath)
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Cloned %q (%s)\n", name, relPath)
	return nil
}

func resolveCloneTarget(cfg *workspace.WorkspaceConfig, args []string, cfgPath string) (string, string, error) {
	destPath, err := resolveCloneDest(args, cfgPath)
	if err != nil {
		return "", "", err
	}

	name, err := cfg.RepoName(destPath)
	if err != nil {
		return "", "", err
	}

	return destPath, name, nil
}

func registerClonedRepo(cmd *cobra.Command, cfg *workspace.WorkspaceConfig, cfgPath, destPath, name, url, group string) (string, error) {
	relPath, err := workspace.RelPath(cfgPath, destPath)
	if err != nil {
		return "", err
	}

	cfg.Repos[name] = workspace.RepoConfig{Path: relPath, URL: url}

	applyMeta(cmd, cfg, cfgPath, relPath, name, group)

	if err := workspace.Save(cfgPath, cfg); err != nil {
		return "", err
	}

	return relPath, nil
}

func resolveCloneDest(args []string, cfgPath string) (string, error) {
	if len(args) >= 2 {
		return filepath.Abs(args[1])
	}

	return filepath.Join(workspace.ConfigDir(cfgPath), deriveClonePath(args[0])), nil
}

func deriveClonePath(rawURL string) string {
	// path.Base (not filepath.Base) intentionally: URLs always use forward slashes regardless of OS.
	base := path.Base(rawURL)
	return strings.TrimSuffix(base, ".git")
}
