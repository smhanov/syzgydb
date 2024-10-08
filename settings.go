package syzgydb

import "math/rand"

// Config holds the configuration settings for the service.
type Config struct {
	OllamaServer string `mapstructure:"ollama_server"`
	TextModel    string `mapstructure:"text_model"`
	ImageModel   string `mapstructure:"image_model"`
	DataFolder   string `mapstructure:"data_folder"`
	SyzgyHost    string `mapstructure:"syzgy_host"`

	// If non-zero, we will use psuedorandom numbers so everything is predictable for testing.
	RandomSeed int64
}

var globalConfig Config

func init() {
	globalConfig = Config{
		OllamaServer: "default_ollama_server",
		TextModel:    "default_text_model",
		ImageModel:   "default_image_model",
		DataFolder:   "default_data_folder",
		SyzgyHost:    "default_syzgy_host",
	}

	myRandom = &myRandomType{}
}

func Configure(cfg Config) {
	globalConfig = cfg
	if cfg.RandomSeed != 0 {
		myRandom.Seed(cfg.RandomSeed)
	} else {
		myRandom.rand = nil
	}
}

type myRandomType struct {
	rand *rand.Rand
}

func (r *myRandomType) Intn(n int) int {
	if r.rand == nil {
		return rand.Intn(n)
	}
	return r.rand.Intn(n)
}

func (r *myRandomType) NormFloat64() float64 {
	if r.rand == nil {
		return rand.NormFloat64()
	}
	return r.rand.NormFloat64()
}

func (r *myRandomType) Float64() float64 {
	if r.rand == nil {
		return rand.Float64()
	}
	return r.rand.Float64()
}

func (r *myRandomType) Seed(n int64) {
	r.rand = rand.New(rand.NewSource(n))
}

func (r *myRandomType) ThreadsafeNew() *myRandomType {
	if r.rand == nil {
		return r
	}
	return &myRandomType{rand.New(rand.NewSource(r.rand.Int63()))}
}

var myRandom *myRandomType
