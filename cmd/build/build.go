// Package build contains commands for building content templates.
package build

import "github.com/spf13/cobra"

// GetBuildCmd returns a cobra.Command that implements functionality
// for building a package from a content template.
func GetBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Builds a package from a template.",
		Long:  "Builds a package from a template.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}
