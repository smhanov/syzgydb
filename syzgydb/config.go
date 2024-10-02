package main

import (
	"fmt"

	"github.com/smhanov/syzgydb"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	// Bind command-line flags
	pflag.String("ollama-server", "", "Hostname and port of the Ollama server")
	pflag.String("text-model", "", "Name of the text embedding model")
	pflag.String("image-model", "", "Name of the image embedding model")
	pflag.String("config", "", "Path to the configuration file")
}

func LoadConfig() error {
	// Set default values
	viper.SetDefault("ollama_server", "localhost:11434")
	viper.SetDefault("text_model", "all-minilm")
	viper.SetDefault("image_model", "minicpm-v")

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// Bind environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("SYZGY")

	// Read configuration file if specified
	configFile := viper.GetString("config")
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("syzgy.conf")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/syzgydb/")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Using defaults and command line/environent options\n     (%v)\n", err)
	}

	// Unmarshal configuration into struct
	var cfg syzgydb.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("unable to decode into struct, %v", err)
	}

	// Assign the loaded configuration to the global variable
	syzgydb.Configure(cfg)
	// Print out the configuration values
	fmt.Println("Configuration values:")
	fmt.Printf("Ollama Server: %s\n", viper.GetString("ollama_server"))
	fmt.Printf("Text Model: %s\n", viper.GetString("text_model"))
	fmt.Printf("Image Model: %s\n", viper.GetString("image_model"))

	return nil
}
