package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ResolveBoolFlag resolves a pair of mutually exclusive boolean flags
// (e.g. --push / --no-push) against a config default.
func ResolveBoolFlag(cmd *cobra.Command, onFlag, offFlag string, dflt bool) (bool, error) {
	on, _ := cmd.Flags().GetBool(onFlag)
	off, _ := cmd.Flags().GetBool(offFlag)

	if on && off {
		return false, fmt.Errorf("--%s and --%s cannot be used together", onFlag, offFlag)
	}

	if on {
		return true, nil
	}

	if off {
		return false, nil
	}

	return dflt, nil
}
