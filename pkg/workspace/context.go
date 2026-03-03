package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/spf13/cobra"
)

func registerContext(root *cobra.Command) {
	root.AddCommand(&cobra.Command{
		Use:     "context [group|auto|none]",
		Aliases: []string{"ctx"},
		Short:   "Get or set the active repo group context",
		Args:    cobra.MaximumNArgs(1),
		RunE:    runContext,
	})
}

func runContext(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runContextShow(cmd)
	}

	switch args[0] {
	case "none":
		return runContextClear(cmd)
	case "auto":
		return runContextAuto(cmd)
	default:
		return runContextSet(cmd, args[0])
	}
}

func runContextShow(cmd *cobra.Command) error {
	cfg, _, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	if cfg.Context.Active == "" {
		output.Writef(cmd.OutOrStdout(), "(none)\n")
		return nil
	}

	output.Writef(cmd.OutOrStdout(), "%s\n", cfg.Context.Active)
	return nil
}

func runContextSet(cmd *cobra.Command, group string) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	if _, ok := cfg.Groups[group]; !ok {
		return fmt.Errorf("group %q not found in workspace config", group)
	}

	if err := config.SaveLocal(cfgPath, config.ContextConfig{Active: group}); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Active context: %q\n", group)
	return nil
}

func runContextClear(cmd *cobra.Command) error {
	_, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	if err := config.SaveLocal(cfgPath, config.ContextConfig{}); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Context cleared\n")
	return nil
}

func runContextAuto(cmd *cobra.Command) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	group, err := detectContextFromCWD(cfg, config.ConfigDir(cfgPath))
	if err != nil {
		return err
	}

	if err := config.SaveLocal(cfgPath, config.ContextConfig{Active: group}); err != nil {
		return err
	}

	output.Writef(cmd.OutOrStdout(), "Active context: %q\n", group)
	return nil
}

// detectContextFromCWD finds the deepest group whose Path contains the current
// working directory. Groups with no Path are skipped.
func detectContextFromCWD(cfg *config.WorkspaceConfig, cfgRoot string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	bestGroup, found := findDeepestMatchingGroup(cfg, cfgRoot, cwd)
	if !found {
		return "", fmt.Errorf("no group with a matching path found for current directory")
	}

	return bestGroup, nil
}

func findDeepestMatchingGroup(cfg *config.WorkspaceConfig, cfgRoot, cwd string) (string, bool) {
	bestGroup := ""
	bestDepth := -1

	for name, g := range cfg.Groups {
		depth, ok := groupMatchDepth(g, cfgRoot, cwd)
		if !ok {
			continue
		}
		// When depths are equal, alphabetically earlier name wins for determinism.
		if isBetterGroupMatch(depth, name, bestDepth, bestGroup) {
			bestDepth = depth
			bestGroup = name
		}
	}

	return bestGroup, bestGroup != ""
}

func groupMatchDepth(g config.GroupConfig, cfgRoot, cwd string) (int, bool) {
	if g.Path == "" {
		return 0, false
	}

	absPath := filepath.Join(cfgRoot, g.Path)
	rel, err := filepath.Rel(absPath, cwd)
	if err != nil || strings.HasPrefix(rel, "..") {
		return 0, false
	}

	return strings.Count(filepath.Clean(g.Path), string(os.PathSeparator)), true
}

func isBetterGroupMatch(depth int, name string, bestDepth int, bestGroup string) bool {
	return depth > bestDepth || (depth == bestDepth && name < bestGroup)
}
