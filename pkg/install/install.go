// Package install handles the installtion of a Puppet Content template
// package.
package install

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/chelnak/pdk/pkg/exec_runner"
	"github.com/chelnak/pdk/pkg/pct_config_processor"
	"github.com/puppetlabs/pct/pkg/config_processor"

	"github.com/puppetlabs/pct/pkg/gzip"
	"github.com/puppetlabs/pct/pkg/httpclient"
	"github.com/puppetlabs/pct/pkg/tar"
	"github.com/spf13/afero"
)

type ConfigParams struct {
	ID      string `mapstructure:"id"`
	Author  string `mapstructure:"author"`
	Version string `mapstructure:"version"`
}

type Installer interface {
	Install(templatePkg, targetDir string, force bool) (string, error)
	InstallClone(GitURI, targetDir string, force bool) (string, error)
}

type installer struct {
	Tar             tar.TarI
	Gunzip          gzip.GunzipI
	AFS             *afero.Afero
	IOFS            *afero.IOFS
	HTTPClient      httpclient.HTTPClientI
	Exec            exec_runner.ExecRunner
	ConfigProcessor config_processor.ConfigProcessorI
	ConfigFile      string
}

func (p *installer) Install(templatePkg, targetDir string, force bool) (namespacedPath string, err error) {
	// Check if the template package path is a url
	if strings.HasPrefix(templatePkg, "http") {
		// Download the tar.gz file and change templatePkg to its download path
		err := p.processDownload(&templatePkg)
		if err != nil {
			return "", err
		}
	}

	if _, err := p.AFS.Stat(templatePkg); os.IsNotExist(err) {
		return "", fmt.Errorf("no package at %v", templatePkg)
	}

	// create a temporary Directory to extract the tar.gz to
	tempDir, err := p.AFS.TempDir("", "")
	defer func() {
		if removeErr := p.AFS.RemoveAll(tempDir); removeErr != nil {
			err = fmt.Errorf("error cleaning up temp dir: %v", removeErr)
		}
	}()

	if err != nil {
		return "", fmt.Errorf("could not create tempdir to gunzip package: %v", err)
	}

	// gunzip the tar.gz to created tempdir
	tarfile, err := p.Gunzip.Gunzip(templatePkg, tempDir)
	if err != nil {
		return "", fmt.Errorf("could not extract TAR from GZIP (%v): %v", templatePkg, err)
	}

	// untar the above archive to the temp dir
	untarPath, err := p.Tar.Untar(tarfile, tempDir)
	if err != nil {
		return "", fmt.Errorf("could not UNTAR package (%v): %v", templatePkg, err)
	}

	// Process the configuration file and set up namespacedPath and relocate config and content to it
	namespacedPath, err = p.InstallFromConfig(filepath.Join(untarPath, p.ConfigFile), targetDir, force)
	if err != nil {
		return "", fmt.Errorf("invalid config: %v", err.Error())
	}

	return namespacedPath, nil
}

func (p *installer) processDownload(templatePkg *string) (err error) {
	u, err := url.ParseRequestURI(*templatePkg)
	if err != nil {
		return fmt.Errorf("could not parse package url %s: %v", *templatePkg, err)
	}
	// Create a temporary Directory to download the tar.gz to
	tempDownloadDir, err := p.AFS.TempDir("", "")
	defer func() {
		if removeErr := p.AFS.Remove(tempDownloadDir); removeErr != nil {
			err = fmt.Errorf("error cleaning up temp dir: %v", removeErr)
		}
	}()
	if err != nil {
		return fmt.Errorf("could not create tempdir to download package: %v", err)
	}
	// Download template and assign location to templatePkg
	*templatePkg, err = p.downloadTemplate(u, tempDownloadDir)
	if err != nil {
		return fmt.Errorf("could not effectively download package: %v", err)
	}
	return nil
}

func (p *installer) InstallClone(GitURI string, targetDir string, force bool) (namespacedPath string, err error) {
	// Create temp dir
	tempDir, err := p.AFS.TempDir("", "")
	defer func() {
		if cleanErr := os.RemoveAll(tempDir); cleanErr != nil {
			err = fmt.Errorf("error cleaning up temp dir: %v", cleanErr)
		}
	}()

	// Validate git URI
	_, err = url.ParseRequestURI(GitURI)
	if err != nil {
		return "", fmt.Errorf("could not parse package uri %s: %v", GitURI, err)
	}

	// Clone git repository to temp folder
	folderPath, err := p.cloneTemplate(GitURI, tempDir)
	if err != nil {
		return "", fmt.Errorf("could not clone git repository: %v", err)
	}

	// Remove .git folder from cloned repository
	err = p.AFS.RemoveAll(filepath.Join(folderPath, ".git"))
	if err != nil {
		return "", fmt.Errorf("failed to remove '.git' directory")
	}

	return p.InstallFromConfig(filepath.Join(folderPath, p.ConfigFile), targetDir, force)
}

func (p *installer) cloneTemplate(GitURI string, tempDir string) (string, error) {
	clonePath := filepath.Join(tempDir, "temp")

	err := p.Exec.Command("git", "clone", GitURI, clonePath)
	if err != nil {
		return "", err
	}

	_, err = p.Exec.Output()
	if err != nil {
		return "", err
	}
	return clonePath, nil
}

func (p *installer) downloadTemplate(targetURL *url.URL, downloadDir string) (downloadPath string, err error) {
	// Get the file contents from URL
	response, err := p.HTTPClient.Get(targetURL.String())
	if err != nil {
		return "", err
	}

	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	if response.StatusCode != 200 {
		message := fmt.Sprintf("Received response code %d when trying to download from %s", response.StatusCode, targetURL.String())
		return "", errors.New(message)
	}

	// Create the empty file
	fileName := filepath.Base(targetURL.Path)
	downloadPath = filepath.Join(downloadDir, fileName)
	file, err := p.AFS.Create(downloadPath)
	if err != nil {
		return "", err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	// Write file contents
	err = p.AFS.WriteReader(downloadPath, response.Body)
	if err != nil {
		return "", err
	}

	return downloadPath, nil
}

func (p *installer) InstallFromConfig(configFile, targetDir string, force bool) (string, error) {
	info, err := p.ConfigProcessor.GetConfigMetadata(configFile)
	if err != nil {
		return "", err
	}

	// Create namespaced directory and move contents of temp folder to it
	installedPkgPath := filepath.Join(targetDir, info.Author, info.Id)

	err = p.AFS.MkdirAll(installedPkgPath, 0750)
	if err != nil {
		return "", err
	}

	installedPkgPath = filepath.Join(installedPkgPath, info.Version)
	untarredPkgDir := filepath.Dir(configFile)

	// finally move to the full path
	errMsgPrefix := "Unable to install in namespace:"
	err = p.AFS.Rename(untarredPkgDir, installedPkgPath)
	if err != nil {
		// if a template already exists
		if !force {
			// error unless forced
			return "", fmt.Errorf("%s Package already installed", errMsgPrefix)
		} else {
			// remove the exiting template
			err = p.AFS.RemoveAll(installedPkgPath)
			if err != nil {
				return "", fmt.Errorf("%s Unable to overwrite existing package: %v", errMsgPrefix, err)
			}
			// perform the move again
			err = p.AFS.Rename(untarredPkgDir, installedPkgPath)
			if err != nil {
				return "", fmt.Errorf("%s Unable to force install: %v", errMsgPrefix, err)
			}
		}
	}

	return installedPkgPath, err
}

func NewInstaller() Installer {
	fs := afero.NewOsFs()
	execRunner := exec_runner.NewExecRunner()

	return &installer{
		Tar:             &tar.Tar{AFS: &afero.Afero{Fs: fs}},
		Gunzip:          &gzip.Gunzip{AFS: &afero.Afero{Fs: fs}},
		AFS:             &afero.Afero{Fs: fs},
		IOFS:            &afero.IOFS{Fs: fs},
		Exec:            execRunner,
		ConfigProcessor: &pct_config_processor.PctConfigProcessor{AFS: &afero.Afero{Fs: fs}},
		ConfigFile:      "pct-config.yml",
	}
}
