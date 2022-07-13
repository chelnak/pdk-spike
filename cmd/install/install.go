// Package install contains commands for installing templates and tools.
package install

import "github.com/spf13/cobra"

// GetInstallCmd returns a cobra.Command that implements functionality
// for installing a template package.
func GetInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Installs a template package (in tar.gz format)",
		Long:  "Installs a template package (in tar.gz format)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}
