// Package build contains commands for building content templates.
package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chelnak/pdk/pkg/build"
	"github.com/chelnak/ysmrr"
	"github.com/spf13/cobra"
)

var (
	sourceDir string
	targetDir string
)

// GetBuildCmd returns a cobra.Command that implements functionality
// for building a package from a content template.
func GetBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Builds a package from a template.",
		Long: `Builds a package from a template.

When building a package you can optionally specify a source directory and a target directory.

If either flag is omitted, the current working directory will be used.`,
		PreRunE: preRun,
		RunE:    run,
	}

	cmd.Flags().StringVarP(&sourceDir, "source", "s", "", "The project directory that will be packaged.")
	cmd.Flags().StringVarP(&targetDir, "target", "t", "", "The directory where the packaged project will be output to.")

	return cmd
}

func preRun(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()

	if (sourceDir == "" || targetDir == "") && err != nil {
		return err
	}

	if sourceDir == "" {
		sourceDir = wd
	}

	sourceDir = filepath.Clean(sourceDir)

	if targetDir == "" {
		targetDir = filepath.Join(sourceDir, "pkg")
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	sm := ysmrr.NewSpinnerManager()
	spinner := sm.AddSpinner("Building package...")
	sm.Start()
	defer sm.Stop()

	builder := build.NewBuilder()
	archive, err := builder.Build(sourceDir, targetDir)
	if err != nil {
		spinner.Error()
		return err
	}

	spinner.Complete()
	message := fmt.Sprintf("Package built to %s\n", archive)
	spinner.UpdateMessage(message)
	return nil
}
