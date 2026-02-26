package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/spf13/cobra"
)

// contextCmd gets or sets the active repo group context.
var contextCmd = &cobra.Command{
	Use:   "context [group|auto|none]",
	Short: "Get or set the active repo group context",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runContext,
}

func init() { rootCmd.AddCommand(contextCmd) }

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
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	if cfg.Context.Active == "" {
		fmt.Fprint(cmd.OutOrStdout(), "(none)\n")
		return nil
	}

	fmt.Fprint(cmd.OutOrStdout(), cfg.Context.Active+"\n")
	return nil
}

func runContextSet(cmd *cobra.Command, group string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	if _, ok := cfg.Groups[group]; !ok {
		return fmt.Errorf("group %q not found in workspace config", group)
	}

	if err := config.SaveLocal(cfgPath, config.ContextConfig{Active: group}); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Active context: %q\n", group)
	return nil
}

func runContextClear(cmd *cobra.Command) error {
	_, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	if err := config.SaveLocal(cfgPath, config.ContextConfig{}); err != nil {
		return err
	}

	fmt.Fprint(cmd.OutOrStdout(), "Context cleared\n")
	return nil
}

func runContextAuto(cmd *cobra.Command) error {
	cfg, cfgPath, err := loadConfig()
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

	fmt.Fprintf(cmd.OutOrStdout(), "Active context: %q\n", group)
	return nil
}

// detectContextFromCWD finds the deepest group whose Path contains the current
// working directory. Groups with no Path are skipped.
func detectContextFromCWD(cfg *config.WorkspaceConfig, cfgRoot string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	bestGroup := ""
	bestDepth := -1

	for name, g := range cfg.Groups {
		if g.Path == "" {
			continue
		}

		absPath := filepath.Join(cfgRoot, g.Path)
		rel, err := filepath.Rel(absPath, cwd)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}

		depth := strings.Count(filepath.Clean(absPath), string(os.PathSeparator))
		if depth > bestDepth {
			bestDepth = depth
			bestGroup = name
		}
	}

	if bestGroup == "" {
		return "", fmt.Errorf("no group with a matching path found for current directory")
	}

	return bestGroup, nil
}
