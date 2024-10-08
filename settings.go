package syzgydb

// Config holds the configuration settings for the service.
type Config struct {
	OllamaServer string `mapstructure:"ollama_server"`
	TextModel    string `mapstructure:"text_model"`
	ImageModel   string `mapstructure:"image_model"`
	DataFolder   string `mapstructure:"data_folder"`
	SyzgyHost    string `mapstructure:"syzgy_host"`
}

import "math/rand"

var GlobalConfig Config

func init() {
	GlobalConfig = Config{
		OllamaServer: "default_ollama_server",
		TextModel:    "default_text_model",
		ImageModel:   "default_image_model",
		DataFolder:   "default_data_folder",
		SyzgyHost:    "default_syzgy_host",
		RandSource:   rand.NewSource(rand.Int63()), // Default random source
	}
}

func Configure(cfg Config) {
	GlobalConfig = cfg
}
