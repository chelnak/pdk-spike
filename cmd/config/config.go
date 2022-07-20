// Package config contains commands for managing pdk config.
package config

import "github.com/spf13/cobra"

func GetConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Commands for working with pdk configuration.",
		Long:  "Commands for working with pdk configuration.",
	}

	cmd.AddCommand(getShowCmd())

	return cmd
}
