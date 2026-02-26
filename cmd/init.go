package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Create a .gitworkspace in the current directory",
	Long: `Creates a .gitworkspace config file in the current directory.
Optionally specify a workspace name; defaults to the directory name.
Also adds .gitworkspace.local to .gitignore (creating .gitignore if absent).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	name := resolveWorkspaceName(cwd, args)

	configPath := filepath.Join(cwd, ".gitworkspace")
	if err := writeInitialConfig(configPath, name); err != nil {
		return err
	}

	if err := ensureGitignore(cwd, ".gitworkspace.local"); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not update .gitignore: %v\n", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Initialized workspace %q in %s\n", name, cwd)
	return nil
}

func resolveWorkspaceName(cwd string, args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return filepath.Base(cwd)
}

func writeInitialConfig(configPath, name string) error {
	// Use Open+Close instead of Stat to avoid a redundant syscall on success.
	if f, err := os.Open(configPath); err == nil {
		f.Close()
		return fmt.Errorf(".gitworkspace already exists in %s", filepath.Dir(configPath))
	}

	content := fmt.Sprintf("[workspace]\nname = %q\n", name)
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("creating .gitworkspace: %w", err)
	}

	return nil
}

func ensureGitignore(dir, entry string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")

	existing, err := os.ReadFile(gitignorePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(f, entry)
	return err
}
