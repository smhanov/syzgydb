package main

import (
	"fmt"
	"os"
	"strings"

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
	pflag.String("data-folder", "./data", "Path to the data folder")
	pflag.String("syzgy-host", "0.0.0.0:8080", "Host and port for the Syzygy server")
	pflag.String("html-root", "./html", "Root directory for serving HTML files")

	f := pflag.CommandLine
	normalizeFunc := f.GetNormalizeFunc()
	f.SetNormalizeFunc(func(fs *pflag.FlagSet, name string) pflag.NormalizedName {
		result := normalizeFunc(fs, name)
		name = strings.ReplaceAll(string(result), "-", "_")
		return pflag.NormalizedName(name)
	})
}

func LoadConfig() error {
	// Set default values
	viper.SetDefault("syzgy_host", "0.0.0.0:8080")
	viper.SetDefault("ollama_server", "127.0.0.1:11434")
	viper.SetDefault("text_model", "all-minilm")
	viper.SetDefault("image_model", "minicpm-v")
	viper.SetDefault("data_folder", "./data")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Parse command-line flags
	pflag.Parse()

	// Bind command-line flags to Viper
	viper.BindPFlags(pflag.CommandLine)

	// Bind environment variables
	viper.AutomaticEnv()

	// Read configuration file if specified
	configFile := viper.GetString("config")
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("syzgy.conf")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Using defaults and command line/environent options\n     (%v)\n", err)
	}

	// Unmarshal configuration into struct
	var cfg syzgydb.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("unable to decode into struct, %v", err)
	}

	// Ensure the data folder exists
	dataFolder := viper.GetString("data_folder")
	if _, err := os.Stat(dataFolder); os.IsNotExist(err) {
		if err := os.MkdirAll(dataFolder, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create data folder: %v", err)
		}
	}
	fmt.Println("Configuration values:")
	fmt.Printf("Ollama Server: %s\n", cfg.OllamaServer)
	fmt.Printf("Text Model: %s\n", cfg.TextModel)
	fmt.Printf("Image Model: %s\n", cfg.ImageModel)
	fmt.Printf("Data Folder: %s\n", cfg.DataFolder)
	fmt.Printf("Port: %s\n", cfg.SyzgyHost)
	fmt.Printf("HTML Root: %s\n", cfg.HTMLRoot)

	// Assign the loaded configuration to the global variable
	syzgydb.Configure(cfg)

	return nil
}
