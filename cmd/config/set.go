package config

import (
	"github.com/chelnak/pdk/internal/config"
	"github.com/spf13/cobra"
)

var (
	key   string
	value string
)

func getSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Sets a configuration property to the specified value value.",
		Long:  "Sets a configuration property to the specified value value.",
		RunE:  setRunE,
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "The configuration property to set.")
	_ = cmd.MarkFlagRequired("key")

	cmd.Flags().StringVarP(&value, "value", "v", "", "The value to set the configuration property to.")
	_ = cmd.MarkFlagRequired("value")

	return cmd
}

func setRunE(cmd *cobra.Command, args []string) error {
	return config.Set(key, value)
}
