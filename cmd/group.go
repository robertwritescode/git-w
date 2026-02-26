package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/spf13/cobra"
)

var (
	groupName      string
	groupAddPath   string
	groupEditPath  string
	groupClearPath bool
)

var (
	// groupCmd is the parent command for all group operations.
	groupCmd        = &cobra.Command{Use: "group", Aliases: []string{"g"}, Short: "Manage repo groups"}
	groupAddCmd     = &cobra.Command{Use: "add <repos...>", Short: "Create a group or add repos to an existing group", Args: cobra.MinimumNArgs(1), RunE: runGroupAdd}
	groupRmCmd      = &cobra.Command{Use: "rm <name>", Short: "Delete a group", Args: cobra.ExactArgs(1), RunE: runGroupRm}
	groupRenameCmd  = &cobra.Command{Use: "rename <old> <new>", Short: "Rename a group", Args: cobra.ExactArgs(2), RunE: runGroupRename}
	groupRmrepoCmd  = &cobra.Command{Use: "rmrepo <repos...>", Short: "Remove repos from a group", Args: cobra.MinimumNArgs(1), RunE: runGroupRmrepo}
	groupListCmd    = &cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List all group names", Args: cobra.NoArgs, RunE: runGroupList}
	groupInfoCmd    = &cobra.Command{Use: "info [name]", Aliases: []string{"ll"}, Short: "Show repos in a group (or all groups)", Args: cobra.MaximumNArgs(1), RunE: runGroupInfo}
	groupSetPathCmd = &cobra.Command{Use: "set-path <name> <path>", Short: "Set the filesystem path for a group (used by context auto)", Args: cobra.ExactArgs(2), RunE: runGroupSetPath}
	groupEditCmd    = &cobra.Command{Use: "edit <name>", Short: "Edit group attributes", Args: cobra.ExactArgs(1), RunE: runGroupEdit}
)

func init() {
	rootCmd.AddCommand(groupCmd)
	groupCmd.AddCommand(groupAddCmd, groupRmCmd, groupRenameCmd, groupRmrepoCmd, groupListCmd, groupInfoCmd, groupSetPathCmd, groupEditCmd)

	groupAddCmd.Flags().StringVarP(&groupName, "name", "n", "", "group name")
	groupRmrepoCmd.Flags().StringVarP(&groupName, "name", "n", "", "group name")

	_ = groupAddCmd.MarkFlagRequired("name")
	_ = groupRmrepoCmd.MarkFlagRequired("name")

	groupAddCmd.Flags().StringVar(&groupAddPath, "path", "", "filesystem path for auto-context detection")
	groupEditCmd.Flags().StringVar(&groupEditPath, "path", "", "set group path")
	groupEditCmd.Flags().BoolVar(&groupClearPath, "clear-path", false, "clear group path")
}

func runGroupAdd(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	if err := validateRegisteredRepos(cfg, args); err != nil {
		return err
	}

	g := cfg.Groups[groupName]
	g.Repos = appendUnique(g.Repos, args)
	if groupAddPath != "" {
		g.Path = groupAddPath
	}
	cfg.Groups[groupName] = g

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Group %q updated\n", groupName)
	return nil
}

func runGroupRm(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	name := args[0]
	if _, ok := cfg.Groups[name]; !ok {
		return fmt.Errorf("group %q not found", name)
	}

	delete(cfg.Groups, name)

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Group %q removed\n", name)
	return nil
}

func runGroupRename(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	oldName, newName := args[0], args[1]

	if _, ok := cfg.Groups[oldName]; !ok {
		return fmt.Errorf("group %q not found", oldName)
	}

	if _, ok := cfg.Groups[newName]; ok {
		return fmt.Errorf("group %q already exists", newName)
	}

	cfg.Groups[newName] = cfg.Groups[oldName]
	delete(cfg.Groups, oldName)

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Renamed group %q to %q\n", oldName, newName)
	return nil
}

func runGroupRmrepo(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	if _, ok := cfg.Groups[groupName]; !ok {
		return fmt.Errorf("group %q not found", groupName)
	}

	g := cfg.Groups[groupName]
	g.Repos = removeItems(g.Repos, args)
	cfg.Groups[groupName] = g

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Updated group %q\n", groupName)
	return nil
}

func runGroupList(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	for _, name := range sortedKeys(cfg.Groups) {
		fmt.Fprintln(cmd.OutOrStdout(), name)
	}
	return nil
}

func runGroupInfo(cmd *cobra.Command, args []string) error {
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	if len(args) == 1 {
		return printGroupInfo(cmd.OutOrStdout(), cfg, args[0])
	}

	for _, name := range sortedKeys(cfg.Groups) {
		if err := printGroupInfo(cmd.OutOrStdout(), cfg, name); err != nil {
			return err
		}
	}
	return nil
}

func appendUnique(existing, items []string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, r := range existing {
		seen[r] = struct{}{}
	}

	result := existing
	for _, item := range items {
		if _, ok := seen[item]; !ok {
			result = append(result, item)
			seen[item] = struct{}{}
		}
	}

	return result
}

func removeItems(slice, drop []string) []string {
	dropSet := make(map[string]struct{}, len(drop))
	for _, d := range drop {
		dropSet[d] = struct{}{}
	}

	var result []string
	for _, s := range slice {
		if _, ok := dropSet[s]; !ok {
			result = append(result, s)
		}
	}

	return result
}

func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func printGroupInfo(w io.Writer, cfg *config.WorkspaceConfig, name string) error {
	g, ok := cfg.Groups[name]
	if !ok {
		return fmt.Errorf("group %q not found", name)
	}

	fmt.Fprintf(w, "%s: %s\n", name, strings.Join(g.Repos, ", "))
	return nil
}

func validateRegisteredRepos(cfg *config.WorkspaceConfig, names []string) error {
	for _, name := range names {
		if _, ok := cfg.Repos[name]; !ok {
			return fmt.Errorf("repo %q is not registered", name)
		}
	}
	return nil
}

func runGroupSetPath(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	name, path := args[0], args[1]
	g, ok := cfg.Groups[name]
	if !ok {
		return fmt.Errorf("group %q not found", name)
	}

	g.Path = path
	cfg.Groups[name] = g

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Group %q path set to %q\n", name, path)
	return nil
}

func runGroupEdit(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		return err
	}

	name := args[0]
	g, ok := cfg.Groups[name]
	if !ok {
		return fmt.Errorf("group %q not found", name)
	}

	if groupEditPath == "" && !groupClearPath {
		return fmt.Errorf("at least one of --path or --clear-path must be provided")
	}
	if groupEditPath != "" && groupClearPath {
		return fmt.Errorf("--path and --clear-path are mutually exclusive")
	}

	if groupClearPath {
		g.Path = ""
	} else {
		g.Path = groupEditPath
	}

	cfg.Groups[name] = g

	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Group %q updated\n", name)
	return nil
}
