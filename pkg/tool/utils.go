package tool

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"

	"github.com/chelnak/pdk/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type Utils struct {
	afs  *afero.Afero
	iofs *afero.IOFS
}

func NewToolUtils(afs *afero.Afero, iofs *afero.IOFS) *Utils {
	return &Utils{
		afs:  afs,
		iofs: iofs,
	}
}

// GetToolNamespace converts a namespace as text into a namepsace object.
// Works with or without a specified version.
// e.g. "puppetlabs/epp/0.2.0" or "puppetlabs/epp"
func (t *Utils) GetToolNamespace(toolNamespacesText string) (Namespace, error) {
	var namespace Namespace
	splitText := strings.Split(toolNamespacesText, "/")

	if len(splitText) < 2 || len(splitText) > 3 {
		return Namespace{}, errors.New("selected tool must be in AUTHOR/ID/VERSION format, with VERSION being optional")
	}

	var version string
	if len(splitText) == 3 {
		version = splitText[2]
	}

	namespace = Namespace{
		ID:      splitText[1],
		Author:  splitText[0],
		Version: version,
	}

	return namespace, nil
}

func (t *Utils) readToolConfig(configFile string) Tool {
	file, err := t.afs.ReadFile(configFile)
	if err != nil {
		log.Error().Msgf("unable to read tool config, %s", configFile)
	}

	var tool Tool

	viper.SetConfigType("yaml")
	err = viper.ReadConfig(bytes.NewBuffer(file))
	if err != nil {
		log.Error().Msgf("unable to read tool config, %s: %s", configFile, err.Error())
		return Tool{}
	}
	err = viper.Unmarshal(&tool.Cfg)

	if err != nil {
		log.Error().Msgf("unable to parse tool config, %s", configFile)
		return Tool{}
	}

	return tool
}

func (t *Utils) ReadToolConfigs(toolPath string, onlyValidators bool) []ToolConfig {
	matches, _ := t.iofs.Glob(toolPath + "/**/**/**/" + "prm-config.yml")

	var templates []ToolConfig
	for _, file := range matches {
		tool := t.readToolConfig(file)
		if tool.Cfg.Plugin != nil {
			if onlyValidators && !tool.Cfg.Common.CanValidate {
				continue
			}
			tool.Cfg.Path = filepath.Dir(file)
			templates = append(templates, tool.Cfg)
		}
	}

	return templates
}

// GetTable reads tools and returns a table of tools to be displayed
func (t *Utils) GetTable(toolPath string, onlyValidators bool) utils.TableOptions {
	toolConfigs := t.ReadToolConfigs(toolPath, onlyValidators)

	var lines [][]string
	for _, config := range toolConfigs {
		fields := config.Plugin
		line := []string{fields.Display, fields.Author, fields.ID, fields.UpstreamProjURL, fields.Version}
		lines = append(lines, line)
	}

	return utils.TableOptions{
		Header: []string{"DisplayName", "Author", "Name", "Project_URL", "Version"},
		Lines:  lines,
	}
}
