package syzgydb

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
)

// ollama_embed_text connects to the configured Ollama server and runs the configured text model
// to generate an embedding for the given text.
func ollama_embed_text(text string) ([]float64, error) {
    // Ensure the global configuration is set
    if GlobalConfig == nil {
        return nil, fmt.Errorf("global configuration is not set")
    }

    // Prepare the request payload
    payload := map[string]interface{}{
        "text":  text,
        "model": GlobalConfig.TextModel,
    }
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request payload: %v", err)
    }

    // Construct the request URL
    url := fmt.Sprintf("http://%s/embed", GlobalConfig.OllamaServer)

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
        Embedding []float64 `json:"embedding"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, fmt.Errorf("failed to decode response: %v", err)
    }

    return response.Embedding, nil
}
