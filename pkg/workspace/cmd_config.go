package workspace

import (
	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/spf13/cobra"
)

func withConfig(cmd *cobra.Command, fn func(cfg *config.WorkspaceConfig, cfgPath string) error) error {
	cfg, cfgPath, err := config.LoadConfig(cmd)
	if err != nil {
		return err
	}

	return fn(cfg, cfgPath)
}

func withMutableConfig(cmd *cobra.Command, fn func(cfg *config.WorkspaceConfig) error) error {
	return withConfig(cmd, func(cfg *config.WorkspaceConfig, cfgPath string) error {
		if err := fn(cfg); err != nil {
			return err
		}

		return config.Save(cfgPath, cfg)
	})
}

func withConfigReadOnly(cmd *cobra.Command, fn func(cfg *config.WorkspaceConfig) error) error {
	return withConfig(cmd, func(cfg *config.WorkspaceConfig, _ string) error {
		return fn(cfg)
	})
}
