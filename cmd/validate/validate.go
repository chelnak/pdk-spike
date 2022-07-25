// Package validate contains commands for validating puppet Content
// with the configured backend.
package validate

import (
	"os"

	"github.com/chelnak/pdk/internal/config"
	"github.com/chelnak/pdk/pkg/validate"
	"github.com/spf13/cobra"
)

var arguments validate.ValidatorOptions

// GetValidateCmd returns a cobra.Command that implements functionality
// for validating a puppet content. It will use installed tools that support
// validation.
func GetValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "validate <tool>",
		Long:    "Validates Puppet Content with a given tool.",
		PreRunE: preExecute,
		RunE:    run,
		Example: "pdk validate puppetlabs/epp --tool-args='templates/motd.epp'",
	}

	cmd.Flags().StringVarP(&arguments.ToolPath, "tool-path", "", config.Config.ToolPath, "The path to the installed tools.")
	cmd.Flags().StringVarP(&arguments.CodePath, "code-path", "", "", "The path to the code that is to be validated.")
	cmd.Flags().StringVarP(&arguments.CachePath, "cache-path", "", config.Config.CachePath, "The path to cache used by PDK.")
	cmd.Flags().StringVar(&arguments.ToolArgs, "tool-args", "", "Additional arguments to pass to a tool")
	cmd.Flags().BoolVarP(&arguments.AlwaysBuild, "always-build", "", false, "Additional arguments to pass to a tool")
	cmd.Flags().BoolVarP(&arguments.Serial, "serial", "", false, "Runs validation one tool at a time instead of in parallel")
	cmd.Flags().IntVarP(&arguments.WorkerCount, "worker-count", "", 2, "Control worker count for running validation tools in parallel")
	cmd.Flags().StringVarP(&arguments.Group, "group", "", "", "Control worker count for running validation tools in parallel")
	cmd.Flags().BoolVarP(&arguments.List, "list", "l", false, "Lists validators")

	return cmd
}

func preExecute(cmd *cobra.Command, args []string) error {
	// Check flags and args, and set defaults
	arguments.Args = args

	if arguments.ToolPath == "" {
		arguments.ToolPath = config.Config.ToolPath
	}

	if arguments.CodePath == "" {
		workingDir, err := os.Getwd()
		if err != nil {
			return err
		}
		arguments.CodePath = workingDir
	}

	if arguments.CachePath == "" {
		arguments.CachePath = config.Config.CachePath
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	validator := validate.NewValidator(arguments)

	if arguments.List {
		validator.List()
		return nil
	}

	if len(args) != 0 || arguments.Group != "" {
		return validator.Run()
	}

	return cmd.Help()
}
