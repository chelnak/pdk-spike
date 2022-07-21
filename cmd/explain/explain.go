// Package explain contains commands for presenting documentation about
// commands provided by PDK.
package explain

import "github.com/spf13/cobra"

// GetExplainCmd returns a cobra.Command that implements functionality
// for explaining functionality of the cli. Think of it as advanced help.
func GetExplainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Present documentation about topics.",
		Long:  "Present documentation about topics.",
		RunE:  nil,
	}

	return cmd
}
