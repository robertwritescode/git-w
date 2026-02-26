package workspace

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

func registerGroup(root *cobra.Command) {
	groupCmd := &cobra.Command{
		Use:     "group",
		Aliases: []string{"g"},
		Short:   "Manage repo groups",
	}

	groupAddCmd := &cobra.Command{
		Use:   "add <repos...>",
		Short: "Create a group or add repos to an existing group",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runGroupAdd,
	}
	groupAddCmd.Flags().StringP("name", "n", "", "group name")
	_ = groupAddCmd.MarkFlagRequired("name")
	groupAddCmd.Flags().String("path", "", "filesystem path for auto-context detection")

	groupRmrepoCmd := &cobra.Command{
		Use:     "remove-repo <repos...>",
		Aliases: []string{"rmrepo"},
		Short:   "Remove repos from a group",
		Args:    cobra.MinimumNArgs(1),
		RunE:    runGroupRmrepo,
	}
	groupRmrepoCmd.Flags().StringP("name", "n", "", "group name")
	_ = groupRmrepoCmd.MarkFlagRequired("name")

	groupEditCmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit group attributes",
		Args:  cobra.ExactArgs(1),
		RunE:  runGroupEdit,
	}
	groupEditCmd.Flags().String("path", "", "set group path")
	groupEditCmd.Flags().Bool("clear-path", false, "clear group path")

	groupCmd.AddCommand(
		groupAddCmd,
		&cobra.Command{
			Use:     "remove <name>",
			Aliases: []string{"rm"},
			Short:   "Delete a group",
			Args:    cobra.ExactArgs(1),
			RunE:    runGroupRm,
		},
		&cobra.Command{
			Use:     "rename <old> <new>",
			Aliases: []string{"mv"},
			Short:   "Rename a group",
			Args:    cobra.ExactArgs(2),
			RunE:    runGroupRename,
		},
		groupRmrepoCmd,
		&cobra.Command{
			Use:     "list",
			Aliases: []string{"ls"},
			Short:   "List all group names",
			Args:    cobra.NoArgs,
			RunE:    runGroupList,
		},
		&cobra.Command{
			Use:     "info [name]",
			Aliases: []string{"ll"},
			Short:   "Show repos in a group (or all groups)",
			Args:    cobra.MaximumNArgs(1),
			RunE:    runGroupInfo,
		},
		groupEditCmd,
	)

	root.AddCommand(groupCmd)
}

func runGroupAdd(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := LoadConfig(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	path, _ := cmd.Flags().GetString("path")

	if err := validateRegisteredRepos(cfg, args); err != nil {
		return err
	}

	g := cfg.Groups[name]
	g.Repos = appendUnique(g.Repos, args)
	if path != "" {
		g.Path = path
	}
	cfg.Groups[name] = g

	if err := Save(cfgPath, cfg); err != nil {
		return err
	}

	writef(cmd.OutOrStdout(), "Group %q updated\n", name)
	return nil
}

func runGroupRm(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := LoadConfig(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	if _, ok := cfg.Groups[name]; !ok {
		return fmt.Errorf("group %q not found", name)
	}

	delete(cfg.Groups, name)

	if err := Save(cfgPath, cfg); err != nil {
		return err
	}

	writef(cmd.OutOrStdout(), "Group %q removed\n", name)
	return nil
}

func runGroupRename(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := LoadConfig(cmd)
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

	if err := Save(cfgPath, cfg); err != nil {
		return err
	}

	writef(cmd.OutOrStdout(), "Renamed group %q to %q\n", oldName, newName)
	return nil
}

func runGroupRmrepo(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := LoadConfig(cmd)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")

	if _, ok := cfg.Groups[name]; !ok {
		return fmt.Errorf("group %q not found", name)
	}

	g := cfg.Groups[name]
	g.Repos = removeItems(g.Repos, args)
	cfg.Groups[name] = g

	if err := Save(cfgPath, cfg); err != nil {
		return err
	}

	writef(cmd.OutOrStdout(), "Updated group %q\n", name)
	return nil
}

func runGroupList(cmd *cobra.Command, args []string) error {
	cfg, _, err := LoadConfig(cmd)
	if err != nil {
		return err
	}

	for _, name := range slices.Sorted(maps.Keys(cfg.Groups)) {
		writef(cmd.OutOrStdout(), "%s\n", name)
	}
	return nil
}

func runGroupInfo(cmd *cobra.Command, args []string) error {
	cfg, _, err := LoadConfig(cmd)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		return printGroupInfo(cmd.OutOrStdout(), cfg, args[0])
	}

	for _, name := range slices.Sorted(maps.Keys(cfg.Groups)) {
		if err := printGroupInfo(cmd.OutOrStdout(), cfg, name); err != nil {
			return err
		}
	}
	return nil
}

func runGroupEdit(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := LoadConfig(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	g, ok := cfg.Groups[name]
	if !ok {
		return fmt.Errorf("group %q not found", name)
	}

	editPath, _ := cmd.Flags().GetString("path")
	clearPath, _ := cmd.Flags().GetBool("clear-path")

	newPath, err := resolveGroupPath(editPath, clearPath)
	if err != nil {
		return err
	}

	g.Path = newPath
	cfg.Groups[name] = g

	if err := Save(cfgPath, cfg); err != nil {
		return err
	}

	writef(cmd.OutOrStdout(), "Group %q updated\n", name)
	return nil
}

func resolveGroupPath(editPath string, clearPath bool) (string, error) {
	if err := validateEditFlags(editPath, clearPath); err != nil {
		return "", err
	}

	if clearPath {
		return "", nil
	}

	return editPath, nil
}

func validateEditFlags(editPath string, clearPath bool) error {
	if editPath == "" && !clearPath {
		return fmt.Errorf("at least one of --path or --clear-path must be provided")
	}

	if editPath != "" && clearPath {
		return fmt.Errorf("--path and --clear-path are mutually exclusive")
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

	return slices.DeleteFunc(slice, func(s string) bool {
		_, ok := dropSet[s]
		return ok
	})
}

func printGroupInfo(w io.Writer, cfg *WorkspaceConfig, name string) error {
	g, ok := cfg.Groups[name]
	if !ok {
		return fmt.Errorf("group %q not found", name)
	}

	writef(w, "%s: %s\n", name, strings.Join(g.Repos, ", "))
	return nil
}

func validateRegisteredRepos(cfg *WorkspaceConfig, names []string) error {
	for _, name := range names {
		if _, ok := cfg.Repos[name]; !ok {
			return fmt.Errorf("repo %q is not registered", name)
		}
	}

	return nil
}
