package syzgydb

import (
	"fmt"
	"log"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds the configuration settings for the service.
type Config struct {
	OllamaServer string `mapstructure:"ollama_server"`
	TextModel    string `mapstructure:"text_model"`
	ImageModel   string `mapstructure:"image_model"`
}

var GlobalConfig *Config

// LoadConfig initializes and loads the configuration settings.
func LoadConfig() error {
	// Set default values
	viper.SetDefault("ollama_server", "localhost:8080")
	viper.SetDefault("text_model", "default-text-model")
	viper.SetDefault("image_model", "default-image-model")

	// Bind command-line flags
	pflag.String("ollama-server", "", "Hostname and port of the Ollama server")
	pflag.String("text-model", "", "Name of the text embedding model")
	pflag.String("image-model", "", "Name of the image embedding model")
	pflag.String("config", "", "Path to the configuration file")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// Bind environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("APP")

	// Read configuration file if specified
	configFile := viper.GetString("config")
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/syzgydb/")
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Could not read config file: %v", err)
	}

	// Unmarshal configuration into struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %v", err)
	}

	// Validate required settings
	if cfg.OllamaServer == "" {
		return nil, fmt.Errorf("Ollama Server configuration is required")
	}

	// Assign the loaded configuration to the global variable
	GlobalConfig = &cfg

	return nil
}
