// Package validate contains commands for validating puppet Content
// with the configured backend.
package validate

import (
	"errors"
	"github.com/chelnak/pdk/pkg/validate"
	"github.com/spf13/cobra"
)

var arguments validate.ValidatorOptions

// GetValidateCmd returns a cobra.Command that implements functionality
// for validating a puppet content. It will use installed tools that support
// validation.
func GetValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "validate",
		Short:   "Validates Puppet Content with a given tool.",
		Long:    "Validates Puppet Content with a given tool.",
		PreRunE: preExecute,
		RunE:    run,
	}

	cmd.Flags().StringVarP(&arguments.ToolPath, "tool-path", "", "", "The path to the installed tools.")
	cmd.Flags().StringVarP(&arguments.CodePath, "code-path", "", ".", "The path to the code that is to be validated.")
	cmd.Flags().StringVarP(&arguments.CachePath, "cache-path", "", "", "The path to cache used by PDK.")
	cmd.Flags().StringVar(&arguments.ToolArgs, "tool-args", "", "Additional arguments to pass to a tool")
	cmd.Flags().BoolVarP(&arguments.AlwaysBuild, "always-build", "", false, "Additional arguments to pass to a tool")
	cmd.Flags().StringVarP(&arguments.ResultsView, "results-view", "", "", "Control where results are outputted to, either 'terminal' or 'file'")
	cmd.Flags().BoolVarP(&arguments.Serial, "serial", "", false, "Runs validation one tool at a time instead of in parallel")
	cmd.Flags().IntVarP(&arguments.WorkerCount, "worker-count", "", 2, "Control worker count for running validation tools in parallel")
	cmd.Flags().StringVarP(&arguments.Group, "group", "", "", "Control worker count for running validation tools in parallel")
	cmd.Flags().BoolVarP(&arguments.List, "list", "l", false, "Runs validation one tool at a time instead of in parallel")

	return cmd
}

func setDefaults() {

}

func preExecute(cmd *cobra.Command, args []string) error {
	// Check flags and args, and set defaults
	arguments.Args = args

	if arguments.ToolPath == "" { // TODO this needs changed obviously
		arguments.ToolPath = "/Users/peter.murphy/Projects/prm/dist/notel_prm_darwin_amd64_v1/tools/"
	}

	if arguments.CodePath == "" {
		return errors.New("invalid code-path provided")
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	validator := validate.NewValidator(arguments)

	if arguments.List {
		validator.List()
		return nil
	}

	return validator.Run()
}
