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
	"github.com/spf13/afero"
)

type Builder interface {
	Build(source, target string) (archivePath string, err error)
}

type builder struct {
	Tar             tar.TarI
	Gzip            gzip.GzipI
	AFS             *afero.Afero
	ConfigProcessor config_processor.ConfigProcessorI
	ConfigFile      string
}

func (b *builder) Build(source, target string) (archivePath string, err error) {
	if err := b.validateProjectStructure(source); err != nil {
		return archivePath, err
	}

	if err := b.ConfigProcessor.CheckConfig(filepath.Join(source, b.ConfigFile)); err != nil {
		return archivePath, fmt.Errorf("invalid config: %v", err.Error())
	}

	return b.makeArchive(source, target)
}

func (b *builder) validateProjectStructure(source string) error {
	// Check project dir exists
	if _, err := b.AFS.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("no project directory at %v", source)
	}

	// Check if config file exists
	if _, err := b.AFS.Stat(filepath.Join(source, b.ConfigFile)); os.IsNotExist(err) {
		return fmt.Errorf("no '%v' found in %v", b.ConfigFile, source)
	}

	// Check if content dir exists
	if _, err := b.AFS.Stat(filepath.Join(source, "content")); os.IsNotExist(err) {
		return fmt.Errorf("no 'content' dir found in %v", source)
	}

	return nil
}

func (b *builder) makeArchive(source, target string) (string, error) {
	var archivePath string

	tempDir, err := b.AFS.TempDir("", "")
	if err != nil {
		return archivePath, fmt.Errorf("could not create tempdir: %v", err)
	}

	defer func() {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil && err == nil {
			err = fmt.Errorf("error cleaning up temp dir: %v", cleanErr)
		}
	}()

	tar, err := b.Tar.Tar(source, tempDir)
	if err != nil {
		return archivePath, fmt.Errorf("could not TAR project (%v): %v", source, err)
	}

	archivePath, err = b.Gzip.Gzip(tar, target)
	if err != nil {
		return archivePath, fmt.Errorf("could not GZIP project (%v): %v", tar, err)
	}

	return archivePath, nil
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
