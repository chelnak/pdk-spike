// Package config is responsible for managing the configuration of the pdk.
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// Config is a package level variable that holds the current configuration
// initialized by the InitConfig method.
var Config config

type config struct {
	AlwaysBuild   bool   `json:"always_build" yaml:"always_build" mapstructure:"always_build"`
	Backend       string `json:"backend" yaml:"backend" mapstructure:"backend"`
	CacheDir      string `json:"cache_dir" yaml:"cache_dir" mapstructure:"cache_dir"`
	CodeDir       string `json:"code_dir" yaml:"code_dir" mapstructure:"code_dir"`
	PuppetVersion string `json:"puppet_version" yaml:"puppet_version" mapstructure:"puppet_version"`
	ResultsView   string `json:"results_view" yaml:"results_view" mapstructure:"results_view"`
	ToolArgs      string `json:"tool_args" yaml:"tool_args" mapstructure:"tool_args"`
	ToolPath      string `json:"tool_path" yaml:"tool_path" mapstructure:"tool_path"`
	ToolTimeout   int    `json:"tool_timeout" yaml:"tool_timeout" mapstructure:"tool_timeout"`
}

func InitConfig(cfgFile string) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)

		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("error reading config file: %v", err)
		}
	} else {
		home, _ := os.UserHomeDir()

		viper.SetConfigName(".pdk")
		viper.SetConfigType("yaml")

		cfgPath := filepath.Join(home, ".config", "puppetlabs", "pdk")
		viper.AddConfigPath(cfgPath)

		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			if err := os.MkdirAll(cfgPath, 0750); err != nil {
				return fmt.Errorf("failed to create config directory: %s", err)
			}
		}

		setDefaults()

		if err := viper.ReadInConfig(); err != nil {
			err := viper.SafeWriteConfig()
			if err != nil {
				return fmt.Errorf("failed to write config: %s", err)
			}
		}

		viper.AutomaticEnv()
		viper.SetEnvPrefix("PDK")

		if err := viper.Unmarshal(&Config); err != nil {
			return fmt.Errorf("failed to unmarshal config: %s", err)
		}
	}

	return nil
}

func setDefaults() {
	// PRM config defaults
	viper.SetDefault("always_build", false)
	viper.SetDefault("backend", "docker")
	viper.SetDefault("cache_dir", "")
	viper.SetDefault("code_dir", "")
	viper.SetDefault("puppet_version", "7.14.0")
	viper.SetDefault("results_view", "terminal")
	viper.SetDefault("tool_args", "")
	viper.SetDefault("tool_path", "") // this should be configured config directory /tools
	viper.SetDefault("tool_timeout", 1800)
}

// Set sets the value of the given key to the given value.
// This method needs far more validation that it currently has.
// it also needs to be able to handle complex types and should probably
// be validated against a schema/struct.
func Set(key string, value interface{}) error {
	viper.Set(key, value)

	err := viper.WriteConfig()
	if err != nil {
		return err
	}

	return nil
}

type writeOptions struct {
	data      string
	lexerName string
	noColor   bool
	writer    io.Writer
}

// Should change to use Puppet colors
func prettyWrite(opts writeOptions) error {
	lexer := lexers.Get(opts.lexerName)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	lexer = chroma.Coalesce(lexer)

	style := styles.Get("native")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal16m")

	if opts.noColor {
		formatter = formatters.Get("noop")
	}

	iterator, err := lexer.Tokenise(nil, opts.data)
	if err != nil {
		return err
	}

	return formatter.Format(opts.writer, style, iterator)
}

// PrintJSON prints the current configuration to the terminal in JSON format.
func PrintJSON(noColor bool, writer io.Writer) error {
	var ifac interface{}
	err := viper.Unmarshal(&ifac)
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(ifac, "", "  ")
	b = append(b, '\n')
	if err != nil {
		return err
	}

	opts := writeOptions{
		data:      string(b),
		lexerName: "json",
		noColor:   noColor,
		writer:    writer,
	}

	return prettyWrite(opts)
}

// PrintYAML prints the current configuration to the terminal in YAML format.
func PrintYAML(noColor bool, writer io.Writer) error {
	var ifac interface{}
	err := viper.Unmarshal(&ifac)
	if err != nil {
		return err
	}

	b, err := yaml.Marshal(ifac)
	y := []byte("---\n")
	y = append(y, b...)
	if err != nil {
		return err
	}

	opts := writeOptions{
		data:      string(y),
		lexerName: "yaml",
		noColor:   noColor,
		writer:    writer,
	}

	return prettyWrite(opts)
}
