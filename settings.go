package syzgydb

// Config holds the configuration settings for the service.
type Config struct {
	OllamaServer string `mapstructure:"ollama_server"`
	TextModel    string `mapstructure:"text_model"`
	ImageModel   string `mapstructure:"image_model"`
	DataFolder   string `mapstructure:"data_folder"`
	SyzgyHost    string `mapstructure:"syzgy_host"`
	RandGen    *rand.Rand  // Random number generator
}

import "math/rand"

var globalRandGen *rand.Rand

var globalConfig Config

func init() {
	globalConfig = Config{
		OllamaServer: "default_ollama_server",
		TextModel:    "default_text_model",
		ImageModel:   "default_image_model",
		DataFolder:   "default_data_folder",
		SyzgyHost:    "default_syzgy_host",
		RandGen:      rand.New(rand.NewSource(rand.Int63())), // Default random generator
	}
}

func Configure(cfg Config) {
	cfg.RandGen = rand.New(cfg.RandSource) // Initialize RandGen with the provided RandSource
	globalConfig = cfg
	globalRandGen = cfg.RandGen
}
