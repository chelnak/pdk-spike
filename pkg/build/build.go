// Package build contains commands for building a pdk project.
package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chelnak/pdk/pkg/pct_config_processor"
	"github.com/puppetlabs/pct/pkg/config_processor"
	"github.com/puppetlabs/pct/pkg/gzip"
	"github.com/puppetlabs/pct/pkg/tar"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

type Builder interface {
	Build(sourceDir, targetDir string) (gzipArchiveFilePath string, err error)
}

type builder struct {
	Tar             tar.TarI
	Gzip            gzip.GzipI
	AFS             *afero.Afero
	ConfigProcessor config_processor.ConfigProcessorI
	ConfigFile      string
}

func (b *builder) Build(sourceDir, targetDir string) (gzipArchiveFilePath string, err error) {
	// Check project dir exists
	if _, err := b.AFS.Stat(sourceDir); os.IsNotExist(err) {
		return "", fmt.Errorf("no project directory at %v", sourceDir)
	}

	// Check if config file exists
	if _, err := b.AFS.Stat(filepath.Join(sourceDir, b.ConfigFile)); os.IsNotExist(err) {
		return "", fmt.Errorf("no '%v' found in %v", b.ConfigFile, sourceDir)
	}

	err = b.ConfigProcessor.CheckConfig(filepath.Join(sourceDir, b.ConfigFile))
	if err != nil {
		return "", fmt.Errorf("invalid config: %v", err.Error())
	}

	// Check if content dir exists
	if _, err := b.AFS.Stat(filepath.Join(sourceDir, "content")); os.IsNotExist(err) {
		return "", fmt.Errorf("no 'content' dir found in %v", sourceDir)
	}

	// Create temp dir and TAR project there
	tempDir, err := b.AFS.TempDir("", "")
	defer func() {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			err = fmt.Errorf("error cleaning up temp dir: %v", cleanErr)
		}
	}()

	if err != nil {
		log.Error().Msgf("could not create tempdir to TAR project: %v", err)
		return "", err
	}

	tarArchiveFilePath, err := b.Tar.Tar(sourceDir, tempDir)
	if err != nil {
		log.Error().Msgf("could not TAR project (%v): %v", sourceDir, err)
		return "", err
	}

	// GZIP the TAR created in the temp dir and output to the /pkg directory in the target directory
	gzipArchiveFilePath, err = b.Gzip.Gzip(tarArchiveFilePath, targetDir)
	if err != nil {
		log.Error().Msgf("could not GZIP project TAR archive (%v): %v", tarArchiveFilePath, err)
		return "", err
	}

	return gzipArchiveFilePath, nil
}

func NewBuilder() Builder {
	fs := afero.NewOsFs()

	return &builder{
		Tar:             &tar.Tar{AFS: &afero.Afero{Fs: fs}},
		Gzip:            &gzip.Gzip{AFS: &afero.Afero{Fs: fs}},
		AFS:             &afero.Afero{Fs: fs},
		ConfigProcessor: &pct_config_processor.PctConfigProcessor{AFS: &afero.Afero{Fs: fs}},
		ConfigFile:      "pct-config.yml",
	}
}
