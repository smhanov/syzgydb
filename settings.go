package syzgydb

// Config holds the configuration settings for the service.
type Config struct {
	OllamaServer string `mapstructure:"ollama_server"`
	TextModel    string `mapstructure:"text_model"`
	ImageModel   string `mapstructure:"image_model"`
}

var GlobalConfig *Config

/*
Configure sets the global configuration for the syzgydb package.

Parameters:
- cfg: The configuration struct to set as the global configuration.
*/
func Configure(cfg Config) {
	GlobalConfig = &cfg
}
