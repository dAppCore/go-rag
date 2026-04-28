package rag

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	core "dappco.re/go"
	"github.com/ollama/ollama/api"
)

func ax7OllamaClient(t *core.T, handler http.HandlerFunc) (*OllamaClient, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	baseURL, err := url.Parse(server.URL)
	core.RequireNoError(t, err)

	return &OllamaClient{
		client: api.NewClient(baseURL, server.Client()),
		config: OllamaConfig{Model: "nomic-embed-text"},
	}, server.Close
}

func TestAX7_DefaultOllamaConfig_Bad(t *core.T) {
	cfg := DefaultOllamaConfig()

	core.AssertNotEqual(t, "", cfg.Host)
	core.AssertNotEqual(t, 0, cfg.Port)
}

func TestAX7_DefaultOllamaConfig_Ugly(t *core.T) {
	cfg := DefaultOllamaConfig()
	cfg.Model = "mutated"

	core.AssertEqual(t, "nomic-embed-text", DefaultOllamaConfig().Model)
	core.AssertEqual(t, "mutated", cfg.Model)
}

func TestAX7_NewOllamaClient_Bad(t *core.T) {
	client, err := NewOllamaClient(OllamaConfig{Host: "localhost", Port: 0})

	core.AssertError(t, err)
	core.AssertNil(t, client)
}

func TestAX7_NewOllamaClient_Ugly(t *core.T) {
	client, err := NewOllamaClient(OllamaConfig{Host: "localhost", Port: 11434, Model: ""})

	core.AssertNoError(t, err)
	core.AssertEqual(t, "", client.Model())
}

func TestAX7_NewOllamaEmbedder_Good(t *core.T) {
	client, err := NewOllamaEmbedder("http://localhost:11434", "custom-model")

	core.AssertNoError(t, err)
	core.AssertEqual(t, "custom-model", client.Model())
}

func TestAX7_NewOllamaEmbedder_Bad(t *core.T) {
	client, err := NewOllamaEmbedder("http://[::1", "custom-model")

	core.AssertError(t, err)
	core.AssertNil(t, client)
}

func TestAX7_NewOllamaEmbedder_Ugly(t *core.T) {
	client, err := NewOllamaEmbedder("", "")

	core.AssertNoError(t, err)
	core.AssertEqual(t, "nomic-embed-text", client.Model())
}

func TestAX7_OllamaClient_Embed_Good(t *core.T) {
	client, closeServer := ax7OllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`))
	})
	defer closeServer()

	vector, err := client.Embed(core.Background(), "hello")
	core.AssertNoError(t, err)
	core.AssertEqual(t, []float32{0.1, 0.2}, vector)
}

func TestAX7_OllamaClient_Embed_Bad(t *core.T) {
	client, closeServer := ax7OllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[]}`))
	})
	defer closeServer()

	vector, err := client.Embed(core.Background(), "hello")
	core.AssertError(t, err)
	core.AssertNil(t, vector)
}

func TestAX7_OllamaClient_Embed_Ugly(t *core.T) {
	client, closeServer := ax7OllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not-json}`))
	})
	defer closeServer()

	vector, err := client.Embed(core.Background(), "hello")
	core.AssertError(t, err)
	core.AssertNil(t, vector)
}

func TestAX7_OllamaClient_EmbedBatch_Good(t *core.T) {
	calls := 0
	client, closeServer := ax7OllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`))
	})
	defer closeServer()

	vectors, err := client.EmbedBatch(core.Background(), []string{"first", "second"})
	core.AssertNoError(t, err)
	core.AssertLen(t, vectors, 2)
	core.AssertEqual(t, 2, calls)
}

func TestAX7_OllamaClient_EmbedBatch_Bad(t *core.T) {
	client, closeServer := ax7OllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[]}`))
	})
	defer closeServer()

	vectors, err := client.EmbedBatch(core.Background(), []string{"first"})
	core.AssertError(t, err)
	core.AssertNil(t, vectors)
}

func TestAX7_OllamaClient_EmbedBatch_Ugly(t *core.T) {
	client := &OllamaClient{}
	vectors, err := client.EmbedBatch(core.Background(), nil)

	core.AssertNoError(t, err)
	core.AssertEmpty(t, vectors)
}

func TestAX7_OllamaClient_EmbedDimension_Good(t *core.T) {
	client := &OllamaClient{config: OllamaConfig{Model: "mxbai-embed-large"}}

	core.AssertEqual(t, uint64(1024), client.EmbedDimension())
	core.AssertGreater(t, client.EmbedDimension(), uint64(768))
}

func TestAX7_OllamaClient_EmbedDimension_Bad(t *core.T) {
	client := &OllamaClient{config: OllamaConfig{Model: "unknown"}}

	core.AssertEqual(t, uint64(768), client.EmbedDimension())
	core.AssertNotEqual(t, uint64(1024), client.EmbedDimension())
}

func TestAX7_OllamaClient_EmbedDimension_Ugly(t *core.T) {
	client := &OllamaClient{config: OllamaConfig{Model: ""}}

	core.AssertEqual(t, uint64(768), client.EmbedDimension())
	core.AssertGreater(t, client.EmbedDimension(), uint64(0))
}

func TestAX7_OllamaClient_Model_Good(t *core.T) {
	client := &OllamaClient{config: OllamaConfig{Model: "nomic-embed-text"}}

	core.AssertEqual(t, "nomic-embed-text", client.Model())
	core.AssertNotEmpty(t, client.Model())
}

func TestAX7_OllamaClient_Model_Bad(t *core.T) {
	client := &OllamaClient{}

	core.AssertEqual(t, "", client.Model())
	core.AssertEmpty(t, client.Model())
}

func TestAX7_OllamaClient_Model_Ugly(t *core.T) {
	client := &OllamaClient{config: OllamaConfig{Model: "model/with:tag"}}

	core.AssertEqual(t, "model/with:tag", client.Model())
	core.AssertContains(t, client.Model(), ":tag")
}

func TestAX7_OllamaClient_VerifyModel_Good(t *core.T) {
	client, closeServer := ax7OllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[0.1]]}`))
	})
	defer closeServer()

	err := client.VerifyModel(core.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestAX7_OllamaClient_VerifyModel_Bad(t *core.T) {
	client, closeServer := ax7OllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[]}`))
	})
	defer closeServer()

	err := client.VerifyModel(core.Background())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not available")
}

func TestAX7_OllamaClient_VerifyModel_Ugly(t *core.T) {
	client, closeServer := ax7OllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`server failed`))
	})
	defer closeServer()

	err := client.VerifyModel(core.Background())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "nomic-embed-text")
}
