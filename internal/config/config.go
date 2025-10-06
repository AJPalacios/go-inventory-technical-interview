package config

import (
	"github.com/spf13/viper"
)

// Config stores all configuration of the application
// The values are read by viper from a config file or environment variables
type Config struct {
	ServerPort  string `mapstructure:"server_port"`
	DBPath      string `mapstructure:"database_path"`
	Environment string `mapstructure:"app_environment"`
	LogLevel    string `mapstructure:"app_log_level"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
