// Package content contains commands for managing Puppet content.
package content

import "github.com/spf13/cobra"

// GetContentCmd returns a cobra.Command that implements functionality for working with
// puppet content templates.
func GetContentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Commands for working with puppet content templates.",
		Long:  "Commands for working with puppet content templates.",
	}

	cmd.AddCommand(getNewCmd())
	cmd.AddCommand(getListCmd())

	return cmd
}
