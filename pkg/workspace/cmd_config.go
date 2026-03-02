package workspace

import "github.com/spf13/cobra"

func withConfig(cmd *cobra.Command, fn func(cfg *WorkspaceConfig, cfgPath string) error) error {
	cfg, cfgPath, err := LoadConfig(cmd)
	if err != nil {
		return err
	}

	return fn(cfg, cfgPath)
}

func withMutableConfig(cmd *cobra.Command, fn func(cfg *WorkspaceConfig) error) error {
	return withConfig(cmd, func(cfg *WorkspaceConfig, cfgPath string) error {
		if err := fn(cfg); err != nil {
			return err
		}

		return Save(cfgPath, cfg)
	})
}

func withConfigReadOnly(cmd *cobra.Command, fn func(cfg *WorkspaceConfig) error) error {
	return withConfig(cmd, func(cfg *WorkspaceConfig, _ string) error {
		return fn(cfg)
	})
}
