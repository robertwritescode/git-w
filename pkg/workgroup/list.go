package workgroup

import (
	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/output"
	"github.com/spf13/cobra"
)

func registerList(workCmd *cobra.Command) {
	workCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List active workgroups",
		RunE:  runList,
	})
}

func runList(cmd *cobra.Command, _ []string) error {
	cfg, _, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	if len(cfg.Workgroups) == 0 {
		output.Writef(cmd.OutOrStdout(), "no workgroups\n")
		return nil
	}

	for _, name := range config.SortedStringKeys(cfg.Workgroups) {
		wg := cfg.Workgroups[name]
		output.Writef(cmd.OutOrStdout(), "%s  branch=%s  repos=%d\n", name, wg.Branch, len(wg.Repos))
	}

	return nil
}
