package content

import "github.com/spf13/cobra"

func getListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all installed templates.",
		Long:  "Lists all installed templates.",
		RunE:  listRunE,
	}

	return cmd
}

func listRunE(cmd *cobra.Command, args []string) error {
	return nil
}
