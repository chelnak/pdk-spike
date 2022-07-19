// Package install contains commands for installing templates and tools.
package install

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	installPath string
	gitURL      string
	force       bool
)

// GetInstallCmd returns a cobra.Command that implements functionality
// for installing a template package.
func GetInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Installs a template package (in tar.gz format)",
		Long:    "Installs a template package (in tar.gz format)",
		PreRunE: preRun,
		RunE:    run,
	}

	cmd.Flags().StringVarP(&installPath, "install-path", "p", "", "The path to the template package.")
	cmd.Flags().StringVarP(&gitURL, "git", "g", "", "The git URL to the template package.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force the installation of the template package.")

	return cmd
}

func preRun(cmd *cobra.Command, args []string) error {
	return nil
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Println("Installing template package...")
	return nil
}
