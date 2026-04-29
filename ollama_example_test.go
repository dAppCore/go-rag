package rag

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	core "dappco.re/go"
	"github.com/ollama/ollama/api"
)

func newExampleOllamaClient(handler http.HandlerFunc) (*OllamaClient, func()) {
	server := httptest.NewServer(handler)
	baseURL, _ := url.Parse(server.URL)
	return &OllamaClient{
		client: api.NewClient(baseURL, server.Client()),
		config: OllamaConfig{Model: "nomic-embed-text"},
	}, server.Close
}

func ExampleDefaultOllamaConfig() {
	cfg := DefaultOllamaConfig()
	core.Println(cfg.Host, cfg.Port, cfg.Model)
	// Output: localhost 11434 nomic-embed-text
}

func ExampleNewOllamaEmbedder() {
	client, err := NewOllamaEmbedder("http://localhost:11434", "custom-model")
	core.Println(err == nil, client.Model())
	// Output: true custom-model
}

func ExampleNewOllamaClient() {
	client, err := NewOllamaClient(DefaultOllamaConfig())
	core.Println(err == nil, client.Model())
	// Output: true nomic-embed-text
}

func ExampleOllamaClient_EmbedDimension() {
	client := &OllamaClient{config: OllamaConfig{Model: "mxbai-embed-large"}}
	core.Println(client.EmbedDimension())
	// Output: 1024
}

func ExampleOllamaClient_Embed() {
	client, closeServer := newExampleOllamaClient(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`))
	})
	defer closeServer()

	vector, err := client.Embed(core.Background(), "hello")
	core.Println(err == nil, len(vector))
	// Output: true 2
}

func ExampleOllamaClient_EmbedBatch() {
	client, closeServer := newExampleOllamaClient(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`))
	})
	defer closeServer()

	vectors, err := client.EmbedBatch(core.Background(), []string{"first", "second"})
	core.Println(err == nil, len(vectors))
	// Output: true 2
}

func ExampleOllamaClient_VerifyModel() {
	client, closeServer := newExampleOllamaClient(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"embeddings":[[0.1]]}`))
	})
	defer closeServer()

	err := client.VerifyModel(core.Background())
	core.Println(err == nil)
	// Output: true
}

func ExampleOllamaClient_Model() {
	client := &OllamaClient{config: OllamaConfig{Model: "nomic-embed-text"}}
	core.Println(client.Model())
	// Output: nomic-embed-text
}
