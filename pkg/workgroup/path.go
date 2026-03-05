package workgroup

import (
	"fmt"
	"path/filepath"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/spf13/cobra"
)

func registerPath(workCmd *cobra.Command) {
	workCmd.AddCommand(&cobra.Command{
		Use:   "path <name> [repo]",
		Short: "Print the path to a workgroup or specific repo worktree",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runPath,
	})
}

func runPath(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	wgName := args[0]
	if _, ok := cfg.Workgroups[wgName]; !ok {
		return fmt.Errorf("workgroup %q not found", wgName)
	}

	if len(args) == 1 {
		output.Writef(cmd.OutOrStdout(), "%s\n", workgroupRootPath(cfgPath, wgName))
		return nil
	}

	repoName := args[1]
	output.Writef(cmd.OutOrStdout(), "%s\n", config.WorkgroupWorktreePath(cfgPath, wgName, repoName))

	return nil
}

func workgroupRootPath(cfgPath, wgName string) string {
	return filepath.Join(config.ConfigDir(cfgPath), ".workgroup", wgName)
}
