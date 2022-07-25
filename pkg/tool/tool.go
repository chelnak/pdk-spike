package tool

import (
	"bytes"
	"errors"
	"github.com/chelnak/pdk/pkg/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"io"
	"path/filepath"
	"strings"
)

type Tool struct {
	Stdout   io.Reader
	Stderr   io.Reader
	ExitCode ToolExitCode
	Cfg      ToolConfig
	Args     []string
}

type ToolExitCode int64

const (
	FAILURE ToolExitCode = iota
	SUCCESS
	TOOL_ERROR
	TOOL_NOT_FOUND
)

type ToolConfig struct {
	Path      string
	Plugin    *PluginConfig    `mapstructure:"plugin"`
	Gem       *GemConfig       `mapstructure:"gem"`
	Container *ContainerConfig `mapstructure:"container"`
	Binary    *BinaryConfig    `mapstructure:"binary"`
	Puppet    *PuppetConfig    `mapstructure:"puppet"`
	Common    CommonConfig     `mapstructure:"common"`
}

// ToolConfigInfo is the housing struct for marshaling YAML data
type ToolConfigInfo struct {
	Plugin   PluginConfig `mapstructure:"plugin"`
	Defaults map[string]interface{}
}

type PluginNamespace struct {
	Id      string `mapstructure:"id"`
	Author  string `mapstructure:"author"`
	Version string `mapstructure:"version"`
}

type PluginConfig struct {
	PluginNamespace `mapstructure:",squash"`
	Display         string `mapstructure:"display"`
	UpstreamProjUrl string `mapstructure:"upstream_project_url"`
}

type BinaryConfig struct {
	Name         string        `mapstructure:"name"`
	InstallSteps *InstallSteps `mapstructure:"install_steps"`
}

type InstallSteps struct {
	Windows string `mapstructure:"windows"`
	Darwin  string `mapstructure:"darwin"`
	Linux   string `mapstructure:"linux"`
}

type ContainerConfig struct {
	Name string `mapstructure:"name"`
	Tag  string `mapstructure:"tag"`
}

type GemConfig struct {
	Name          []string                      `mapstructure:"name"`
	Executable    string                        `mapstructure:"executable"`
	BuildTools    bool                          `mapstructure:"build_tools"`
	Compatibility map[float32]map[string]string `mapstructure:"compatibility"`
}

type PuppetConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type CommonConfig struct {
	CanValidate         bool              `mapstructure:"can_validate"`
	NeedsWriteAccess    bool              `mapstructure:"needs_write_access"`
	UseScript           string            `mapstructure:"use_script"`
	RequiresGit         bool              `mapstructure:"requires_git"`
	DefaultArgs         []string          `mapstructure:"default_args"`
	HelpArg             string            `mapstructure:"help_arg"`
	SuccessExitCode     int               `mapstructure:"success_exit_code"`
	InterleaveStdOutErr bool              `mapstructure:"interleave_stdout"`
	OutputMode          *OutputModes      `mapstructure:"output_mode"`
	Env                 map[string]string `mapstructure:"env"`
}

type OutputModes struct {
	Json  string `mapstructure:"json"`
	Yaml  string `mapstructure:"yaml"`
	Junit string `mapstructure:"junit"`
}

func GetToolNamespace(toolNamespacesText string) (PluginNamespace, error) {
	var namespace PluginNamespace
	splitText := strings.Split(toolNamespacesText, "/")

	if len(splitText) < 2 || len(splitText) > 3 {
		return PluginNamespace{}, errors.New("selected tool must be in AUTHOR/ID/VERSION format, with VERSION being optional")
	}

	var version string
	if len(splitText) == 3 {
		version = splitText[2]
	}

	namespace = PluginNamespace{
		Id:      splitText[1],
		Author:  splitText[0],
		Version: version,
	}

	return namespace, nil
}

func readToolConfig(configFile string) Tool {
	// TODO Replace with config afs
	fs := afero.NewOsFs()
	afs := afero.Afero{Fs: &afero.Afero{Fs: fs}}
	// =======================================
	file, err := afs.ReadFile(configFile)
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

func ReadAllTools(toolPath string, onlyValidators bool) []ToolConfig {
	// TODO Replace with config afs
	fs := afero.NewOsFs()
	iofs := afero.IOFS{Fs: &afero.Afero{Fs: fs}}
	// =======================================
	matches, _ := iofs.Glob(toolPath + "/**/**/**/" + "prm-config.yml")

	var templates []ToolConfig
	for _, file := range matches {
		tool := readToolConfig(file)
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

func GetTable(toolPath string, onlyValidators bool) utils.TableOptions {
	toolConfigs := ReadAllTools(toolPath, onlyValidators)

	var lines [][]string
	for _, config := range toolConfigs {
		fields := config.Plugin
		line := []string{fields.Display, fields.Author, fields.Id, fields.UpstreamProjUrl, fields.Version}
		lines = append(lines, line)
	}

	return utils.TableOptions{
		Header: []string{"DisplayName", "Author", "Name", "Project_URL", "Version"},
		Lines:  lines,
	}
}
