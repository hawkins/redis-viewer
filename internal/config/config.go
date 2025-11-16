// Package config handles configuration loading
package config

import "github.com/spf13/viper"

// Config represents the application configuration
type Config struct {
	Addrs      []string
	DB         int
	Username   string
	Password   string
	MasterName string `mapstructure:"master_name"`
	Limit      int64
}

// Get retrieves configuration from Viper
func Get() Config {
	var config Config
	_ = viper.Unmarshal(&config)
	return config
}
