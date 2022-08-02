// Package exec contains commands for executing tools against some Puppet Content
// using the configured backend.
package exec

import "github.com/spf13/cobra"

// GetExecCmd returns a cobra.Command that implements functionality fpr executing a
// tool against some Puppet content.
func GetExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Executes a given tool against some Puppet Content.",
		Long:  "Executes a given tool against some Puppet Content.",
		RunE:  execRunE,
	}

	return cmd
}

func execRunE(cmd *cobra.Command, args []string) error {
	return nil
}
