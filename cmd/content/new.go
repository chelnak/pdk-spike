package content

import "github.com/spf13/cobra"

func getNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Creates a Puppet project or other artifact based on a template.",
		Long:  "Creates a Puppet project or other artifact based on a template.",
		RunE:  newRunE,
	}

	return cmd
}

func newRunE(cmd *cobra.Command, args []string) error {
	return nil
}
