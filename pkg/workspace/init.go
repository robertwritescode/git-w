package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/robertwritescode/git-w/pkg/gitutil"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/spf13/cobra"
)

func registerInit(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:   "init [name]",
		Short: "Create a .gitw in the current directory",
		Long: `Creates a .gitw config file in the current directory.
Optionally specify a workspace name; defaults to the directory name.
Also adds .gitw.local to .gitignore (creating .gitignore if absent).`,
		Args: cobra.MaximumNArgs(1),
		RunE: runInit,
	})
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	name := resolveWorkspaceName(cwd, args)

	configPath := filepath.Join(cwd, ".gitw")
	if err := writeInitialConfig(configPath, name); err != nil {
		return err
	}

	if err := gitutil.EnsureGitignore(cwd, ".gitw.local"); err != nil {
		output.Writef(cmd.ErrOrStderr(), "warning: could not update .gitignore: %v\n", err)
	}

	output.Writef(cmd.OutOrStdout(), "Initialized workspace %q in %s\n", name, cwd)
	return nil
}

func resolveWorkspaceName(cwd string, args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	return filepath.Base(cwd)
}

func writeInitialConfig(configPath, name string) error {
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf(".gitw already exists in this directory")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	content := fmt.Sprintf("[workspace]\nname = %q\n", name)
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("creating .gitw: %w", err)
	}

	return nil
}
