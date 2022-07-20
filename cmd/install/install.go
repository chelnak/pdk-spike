// Package install contains commands for installing templates and tools.
package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chelnak/pdk/internal/stringutils"
	"github.com/chelnak/pdk/pkg/install"
	"github.com/chelnak/ysmrr"
	"github.com/spf13/cobra"
)

var (
	source string
	target string
	force  bool
)

// GetInstallCmd returns a cobra.Command that implements functionality
// for installing a template package.
func GetInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Installs a template package in tar.gz format or from a git repository.",
		Long:    "Installs a template package in tar.gz format or from a git repository.",
		PreRunE: preRun,
		RunE:    run,
	}

	cmd.Flags().StringVarP(&source, "source", "s", "", "The path of the template package.")
	_ = cmd.MarkFlagRequired("source")

	cmd.Flags().StringVarP(&target, "target", "t", "", "The directory where the template package will be installed.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force the installation of the template package.")

	return cmd
}

func preRun(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()

	if target == "" && err != nil {
		return err
	}

	if target == "" {
		target = filepath.Clean(wd)
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	sm := ysmrr.NewSpinnerManager()

	spinner := sm.AddSpinner("Installing package...")
	sm.Start()
	defer sm.Stop()

	installer := install.NewInstaller()

	var i string
	var err error
	if stringutils.IsGitURL(source) {
		i, err = installer.InstallClone(source, target, force)
	} else if stringutils.IsTarGZ(source) {
		i, err = installer.Install(source, target, force)
	} else {
		spinner.Error()
		return fmt.Errorf("invalid source path: %s", source)
	}

	if err != nil {
		spinner.Error()
		return err
	}

	message := fmt.Sprintf("Installed %s\n", i)
	spinner.UpdateMessage(message)
	spinner.Complete()
	return nil
}
