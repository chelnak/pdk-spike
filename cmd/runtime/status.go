package runtime

import "github.com/spf13/cobra"

func getStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Shows the status of the runtime.",
		Long:  "Shows the status of the runtime.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return cmd
}
