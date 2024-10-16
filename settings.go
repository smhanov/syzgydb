package syzgydb

import (
	"hash/fnv"
	"math/rand"
	"net"
)

// Config holds the configuration settings for the service.
type Config struct {
	OllamaServer string `mapstructure:"ollama_server"`
	TextModel    string `mapstructure:"text_model"`
	ImageModel   string `mapstructure:"image_model"`
	DataFolder   string `mapstructure:"data_folder"`
	SyzgyHost    string `mapstructure:"syzgy_host"`

	// If non-zero, we will use psuedorandom numbers so everything is predictable for testing.
	RandomSeed int64

	// Replication engine settings
	ReplicationOwnURL   string   `mapstructure:"replication_own_url"`
	ReplicationPeerURLs []string `mapstructure:"replication_peer_urls"`
	ReplicationJWTKey   string   `mapstructure:"replication_jwt_key"`
	NodeID              uint64   `mapstructure:"node_id"`
}

var globalConfig Config

func init() {
	globalConfig = Config{
		OllamaServer:        "default_ollama_server",
		TextModel:           "default_text_model",
		ImageModel:          "default_image_model",
		DataFolder:          "default_data_folder",
		SyzgyHost:           "default_syzgy_host",
		ReplicationOwnURL:   "http://localhost:8080",
		ReplicationPeerURLs: []string{},
		ReplicationJWTKey:   "",
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

// GetServerHash returns a unique consistent hash for the server
func GetServerHash() (uint64, error) {
	// Get the MAC address
	interfaces, err := net.Interfaces()
	if err != nil {
		return 0, err
	}

	var mac net.HardwareAddr
	for _, intf := range interfaces {
		if len(intf.HardwareAddr) > 0 {
			mac = intf.HardwareAddr
			break
		}
	}

	if mac == nil {
		return 0, fmt.Errorf("no valid MAC address found")
	}

	// Use FNV-1a hash
	h := fnv.New64a()
	h.Write(mac)
	return h.Sum64(), nil
}
