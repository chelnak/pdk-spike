package content

import "github.com/spf13/cobra"

func getListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all installed templates.",
		Long:  "Lists all installed templates.",
		Run:   nil,
	}

	return cmd
}
