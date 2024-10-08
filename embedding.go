package syzgydb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
)

const maxCacheSize = 100

var (
	embeddingCache = newLRUCache(maxCacheSize)
	cacheMutex     sync.RWMutex
)

type EmbedTextFunc func(text []string, useCache bool) ([][]float64, error)

// Default implementation of the embedding function
var embedText EmbedTextFunc = EmbedText

// EmbedText connects to the configured Ollama server and runs the configured text model
// to generate an embedding for the given text.
func EmbedText(texts []string, useCache bool) ([][]float64, error) {
	// Check the cache first if useCache is true
	if useCache {
		cachedEmbeddings := make([][]float64, len(texts))
		allCached := true

		cacheMutex.RLock()
		for i, text := range texts {
			if embedding, found := embeddingCache.get(text); found {
				cachedEmbeddings[i] = embedding
			} else {
				allCached = false
				break
			}
		}
		cacheMutex.RUnlock()

		if allCached {
			return cachedEmbeddings, nil
		}
	}

	// Prepare the request payload
	payload := map[string]interface{}{
		"model": globalConfig.TextModel,
		"input": texts,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %v", err)
	}

	// Construct the request URL
	url := globalConfig.OllamaServer
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	url = fmt.Sprintf("%s/api/embed", url)
	log.Printf("Sending to %v %v", url, payload)

	// Make the HTTP request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama server: %v", err)
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get embedding: %s", string(bodyBytes))
	}

	// Parse the response
	var response struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Check if embeddings are present
	if len(response.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings found in response")
	}

	// Store the new embeddings in the cache if useCache is true
	if useCache {
		cacheMutex.Lock()
		for i, text := range texts {
			embeddingCache.put(text, response.Embeddings[i])
		}
		cacheMutex.Unlock()
	}

	return response.Embeddings, nil
}
