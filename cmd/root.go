// Package cmd is responsible for holding all cobra commands that
// make up the cli.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/chelnak/pdk/cmd/build"
	"github.com/chelnak/pdk/cmd/config"
	"github.com/chelnak/pdk/cmd/content"
	"github.com/chelnak/pdk/cmd/exec"
	"github.com/chelnak/pdk/cmd/explain"
	"github.com/chelnak/pdk/cmd/install"
	"github.com/chelnak/pdk/cmd/runtime"
	"github.com/chelnak/pdk/cmd/validate"
	appConfig "github.com/chelnak/pdk/internal/config"
	"github.com/spf13/cobra"
)

var (
	errSilent  = errors.New("ErrSilent")
	configFile string
)

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pdk",
		Short: "PDK - The shortest path to better modules.",
		Long: `PDK - The shortest path to better modules.

The Puppet Development Kit includes key Puppet code development and testing tools for Linux, Windows, and OS X workstations,
so you can install one package with the tools you need to create and validate new modules.

PDK includes testing tools, a complete module skeleton, and command line tools to help you create, validate, and run tests on Puppet modules.`,
		SilenceErrors:     true,
		SilenceUsage:      true,
		PersistentPreRunE: persistentPreRun,
	}

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to a config file. This will override the default config file located at $HOME/.config/puppetlabs/pdk/.pdk.yaml.")
	_ = rootCmd.MarkFlagFilename("config", "yaml", "yml")

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode.")

	return rootCmd
}

func persistentPreRun(cmd *cobra.Command, args []string) error {
	return appConfig.InitConfig(configFile)
}

func formatError(err error) {
	fmt.Println("\n❌ It looks like something went wrong!\n\nFor more details try running the command again with --debug.")
	fmt.Println("\nReported errors:")
	fmt.Fprintln(os.Stderr, fmt.Errorf("• %s", err))
	fmt.Println()
}

// Execute is the entrypoint for the cli. It is called directly from main.
func Execute() int {
	rootCmd := getRootCmd()

	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(err)
		cmd.Println(cmd.UsageString())
		return errSilent
	})

	rootCmd.AddCommand(content.GetContentCmd())
	rootCmd.AddCommand(build.GetBuildCmd())
	rootCmd.AddCommand(install.GetInstallCmd())
	rootCmd.AddCommand(exec.GetExecCmd())
	rootCmd.AddCommand(validate.GetValidateCmd())
	rootCmd.AddCommand(runtime.GetRuntimeCmd())
	rootCmd.AddCommand(explain.GetExplainCmd())
	rootCmd.AddCommand(config.GetConfigCmd())

	if err := rootCmd.Execute(); err != nil {
		if err != errSilent {
			formatError(err)
		}

		return 1
	}

	return 0
}
