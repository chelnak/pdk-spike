package config

import (
	"errors"
	"os"

	"github.com/chelnak/pdk/internal/config"
	"github.com/chelnak/pdk/internal/utils/terminal"
	"github.com/spf13/cobra"
)

var (
	output  string
	noColor bool
)

func getShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Prints the current configuration to the terminal in either JSON or YAML format. Defaults to YAML.",
		Long:  "Prints the current configuration to the terminal in either JSON or YAML format. Defaults to YAML.",
		RunE:  run,
	}

	cmd.Flags().StringVarP(&output, "output", "o", "yaml", "The output format. Valid values are 'json' and 'yaml'. Defaults to 'yaml'.")
	cmd.Flags().BoolVarP(&noColor, "no-color", "n", false, "Disable color output")
	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	// Prevent ascii escape codes from being printed when we are not in a TTY
	if !terminal.IsTTY() && !noColor {
		noColor = true
	}

	switch output {
	case "json":
		return config.PrintJSON(noColor, os.Stdout)
	case "yaml":
		return config.PrintYAML(noColor, os.Stdout)
	default:
		return errors.New("invalid output format. Valid values are 'json' and 'yaml'")
	}
}
