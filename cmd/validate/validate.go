// Package validate contains commands for validating puppet Content
// with the configured backend.
package validate

import "github.com/spf13/cobra"

// GetValidateCmd returns a cobra.Command that implements functionality
// for validating a puppet content. It will use installed tools that support
// validation.
func GetValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validates Puppet Content with a given tool.",
		Long:  "Validates Puppet Content with a given tool.",
		RunE:  validateRunE,
	}

	return cmd
}

func validateRunE(cmd *cobra.Command, args []string) error {
	return nil
}
