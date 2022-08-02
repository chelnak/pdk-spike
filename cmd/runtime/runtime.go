// Package runtime contains commands for managing the Puppet runtime
// used by PDK.
package runtime

import "github.com/spf13/cobra"

// GetRuntimeCmd returns a cobra.Command that implements functionality
// for working with the current runtime. In this context a runtime is the
// underlying platform that will drive the exec and validate commands.
func GetRuntimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runtime",
		Short: "Manage the runtime used by PDK.",
		Long:  "Manage the runtime used by PDK.",
	}

	cmd.AddCommand(getStatusCmd())

	return cmd
}
