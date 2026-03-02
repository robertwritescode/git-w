package cmd

import (
	"fmt"
	"os"

	gitpkg "github.com/robertwritescode/git-w/pkg/git"
	"github.com/robertwritescode/git-w/pkg/repo"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/robertwritescode/git-w/pkg/worktree"
	"github.com/spf13/cobra"
)

// newRootCmd builds and returns a fully-wired cobra root command.
func newRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "git-w",
		Short: "A Git plugin for managing meta-repo workspaces",
		Long: `git-w makes it easy to set up, share, manage, and run operations across a meta-repo.
Invoke as 'git w <cmd>' via git's plugin system (git-w must be in $PATH).`,
	}

	root.PersistentFlags().String("config", "", "path to .gitw config (default: nearest .gitw found by walking up from CWD)")

	workspace.Register(root)
	repo.Register(root)
	worktree.Register(root)
	gitpkg.Register(root)
	registerCompletion(root)

	if version != "" {
		root.Version = version
	}

	return root
}

// Execute builds the command tree and runs it.
func Execute(version string) {
	if err := newRootCmd(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
