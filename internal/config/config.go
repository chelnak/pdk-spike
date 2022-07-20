package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

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
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("PDK")

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
