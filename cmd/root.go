// Package cmd is responsible for holding all cobra commands that
// make up the cli.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/chelnak/pdk/cmd/build"
	"github.com/chelnak/pdk/cmd/content"
	"github.com/chelnak/pdk/cmd/exec"
	"github.com/chelnak/pdk/cmd/explain"
	"github.com/chelnak/pdk/cmd/install"
	"github.com/chelnak/pdk/cmd/runtime"
	"github.com/chelnak/pdk/cmd/validate"
	"github.com/spf13/cobra"
)

var errSilent = errors.New("ErrSilent")

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pdk",
		Short: "PDK - The shortest path to better modules.",
		Long: `PDK - The shortest path to better modules.

The Puppet Development Kit includes key Puppet code development and testing tools for Linux, Windows, and OS X workstations,
so you can install one package with the tools you need to create and validate new modules.

PDK includes testing tools, a complete module skeleton, and command line tools to help you create, validate, and run tests on Puppet modules.`,
		Args:          cobra.MinimumNArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		Run:           nil,
	}

	return rootCmd
}

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

	if err := rootCmd.Execute(); err != nil {
		if err != errSilent {
			fmt.Fprintln(os.Stderr, fmt.Errorf("❌ %s", err))
		}

		return 1
	}

	return 0
}
