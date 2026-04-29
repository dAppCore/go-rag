package rag

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	core "dappco.re/go"
	"github.com/ollama/ollama/api"
)

// --- DefaultOllamaConfig tests ---

func TestOllama_DefaultOllamaConfig_Good(t *testing.T) {
	t.Run("returns expected default values", func(t *testing.T) {
		cfg := DefaultOllamaConfig()

		assertEqual(t, "localhost", cfg.Host, "default host should be localhost")
		assertEqual(t, 11434, cfg.Port, "default port should be 11434")
		assertEqual(t, "nomic-embed-text", cfg.Model, "default model should be nomic-embed-text")
	})
}

// --- NewOllamaClient tests ---

func TestOllama_NewOllamaClient_Good(t *testing.T) {
	t.Run("constructs client with default config", func(t *testing.T) {
		r := NewOllamaClient(DefaultOllamaConfig())
		client := resultValue[*OllamaClient](t, r)

		assertNotNil(t, client)
		assertEqual(t, "nomic-embed-text", client.Model())
		assertEqual(t, uint64(768), client.EmbedDimension())
	})

	t.Run("constructs client with custom config", func(t *testing.T) {
		cfg := OllamaConfig{
			Host:  "10.0.0.1",
			Port:  8080,
			Model: "mxbai-embed-large",
		}
		r := NewOllamaClient(cfg)
		client := resultValue[*OllamaClient](t, r)

		assertNotNil(t, client)
		assertEqual(t, "mxbai-embed-large", client.Model())
		assertEqual(t, uint64(1024), client.EmbedDimension())
	})
}

// --- EmbedDimension tests ---

func TestOllama_EmbedDimension_Good(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected uint64
	}{
		{
			name:     "nomic-embed-text returns 768",
			model:    "nomic-embed-text",
			expected: 768,
		},
		{
			name:     "mxbai-embed-large returns 1024",
			model:    "mxbai-embed-large",
			expected: 1024,
		},
		{
			name:     "all-minilm returns 384",
			model:    "all-minilm",
			expected: 384,
		},
		{
			name:     "unknown model defaults to 768",
			model:    "some-unknown-model",
			expected: 768,
		},
		{
			name:     "empty model name defaults to 768",
			model:    "",
			expected: 768,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Construct OllamaClient directly with the config to avoid needing a live server.
			// We only set the config field; client is nil but EmbedDimension does not use it.
			client := &OllamaClient{
				config: OllamaConfig{Model: tc.model},
			}

			dim := client.EmbedDimension()
			assertEqual(t, tc.expected, dim)
		})
	}
}

// --- Model tests ---

func TestOllama_Model_Good(t *testing.T) {
	t.Run("returns the configured model name", func(t *testing.T) {
		client := &OllamaClient{
			config: OllamaConfig{Model: "nomic-embed-text"},
		}

		assertEqual(t, "nomic-embed-text", client.Model())
	})

	t.Run("returns custom model name", func(t *testing.T) {
		client := &OllamaClient{
			config: OllamaConfig{Model: "custom-model"},
		}

		assertEqual(t, "custom-model", client.Model())
	})
}

// --- EmbedBatch tests ---

func TestOllama_EmbedBatch_Good(t *testing.T) {
	var requestCount int
	var capturedInput []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		body, err := io.ReadAll(r.Body)
		assertNoError(t, err)

		var req struct {
			Input string `json:"input"`
		}
		result := core.JSONUnmarshalString(string(body), &req)
		assertTrue(t, result.OK, "request body should decode")
		capturedInput = append(capturedInput, req.Input)

		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`))
			return
		}
		_, _ = w.Write([]byte(`{"embeddings":[[0.3,0.4]]}`))
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	assertNoError(t, err)

	client := &OllamaClient{
		client: api.NewClient(baseURL, server.Client()),
		config: OllamaConfig{Model: "nomic-embed-text"},
	}

	r := client.EmbedBatch(context.Background(), []string{"first", "second"})
	vectors := resultValue[[][]float32](t, r)
	assertLen(t, vectors, 2)
	assertEqual(t, []float32{0.1, 0.2}, vectors[0])
	assertEqual(t, []float32{0.3, 0.4}, vectors[1])
	assertEqual(t, 2, requestCount, "batch embedding should call Embed once per input")
	assertEqual(t, []string{"first", "second"}, capturedInput)
}

func TestOllama_EmbedBatch_Bad(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[]}`))
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	assertNoError(t, err)

	client := &OllamaClient{
		client: api.NewClient(baseURL, server.Client()),
		config: OllamaConfig{Model: "nomic-embed-text"},
	}

	r := client.EmbedBatch(context.Background(), []string{"first", "second"})
	assertError(t, r)
	assertContains(t, r.Error(), "item 1")
	assertEqual(t, 2, requestCount)
}

func testOllamaClient(t *core.T, handler http.HandlerFunc) (*OllamaClient, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	baseURL, err := url.Parse(server.URL)
	core.RequireNoError(t, err)

	return &OllamaClient{
		client: api.NewClient(baseURL, server.Client()),
		config: OllamaConfig{Model: "nomic-embed-text"},
	}, server.Close
}

func TestOllama_DefaultOllamaConfig_Bad(t *core.T) {
	cfg := DefaultOllamaConfig()

	core.AssertNotEqual(t, "", cfg.Host)
	core.AssertNotEqual(t, 0, cfg.Port)
}

func TestOllama_DefaultOllamaConfig_Ugly(t *core.T) {
	cfg := DefaultOllamaConfig()
	cfg.Model = "mutated"

	core.AssertEqual(t, "nomic-embed-text", DefaultOllamaConfig().Model)
	core.AssertEqual(t, "mutated", cfg.Model)
}

func TestOllama_NewOllamaClient_Bad(t *core.T) {
	r := NewOllamaClient(OllamaConfig{Host: "localhost", Port: 0})

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "port out of range")
}

func TestOllama_NewOllamaClient_Ugly(t *core.T) {
	r := NewOllamaClient(OllamaConfig{Host: "localhost", Port: 11434, Model: ""})
	client := r.Value.(*OllamaClient)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "", client.Model())
}

func TestOllama_NewOllamaEmbedder_Good(t *core.T) {
	r := NewOllamaEmbedder("http://localhost:11434", "custom-model")
	client := r.Value.(*OllamaClient)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "custom-model", client.Model())
}

func TestOllama_NewOllamaEmbedder_Bad(t *core.T) {
	r := NewOllamaEmbedder("http://[::1", "custom-model")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "invalid Ollama endpoint")
}

func TestOllama_NewOllamaEmbedder_Ugly(t *core.T) {
	r := NewOllamaEmbedder("", "")
	client := r.Value.(*OllamaClient)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "nomic-embed-text", client.Model())
}

func TestOllama_OllamaClient_Embed_Good(t *core.T) {
	client, closeServer := testOllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`))
	})
	defer closeServer()

	r := client.Embed(core.Background(), "hello")
	vector := r.Value.([]float32)
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, []float32{0.1, 0.2}, vector)
}

func TestOllama_OllamaClient_Embed_Bad(t *core.T) {
	client, closeServer := testOllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[]}`))
	})
	defer closeServer()

	r := client.Embed(core.Background(), "hello")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "empty embedding")
}

func TestOllama_OllamaClient_Embed_Ugly(t *core.T) {
	client, closeServer := testOllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not-json}`))
	})
	defer closeServer()

	r := client.Embed(core.Background(), "hello")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "failed to generate embedding")
}

func TestOllama_OllamaClient_EmbedBatch_Good(t *core.T) {
	calls := 0
	client, closeServer := testOllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`))
	})
	defer closeServer()

	r := client.EmbedBatch(core.Background(), []string{"first", "second"})
	vectors := r.Value.([][]float32)
	core.AssertTrue(t, r.OK)
	core.AssertLen(t, vectors, 2)
	core.AssertEqual(t, 2, calls)
}

func TestOllama_OllamaClient_EmbedBatch_Bad(t *core.T) {
	client, closeServer := testOllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[]}`))
	})
	defer closeServer()

	r := client.EmbedBatch(core.Background(), []string{"first"})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "item 0")
}

func TestOllama_OllamaClient_EmbedBatch_Ugly(t *core.T) {
	client := &OllamaClient{}
	r := client.EmbedBatch(core.Background(), nil)
	vectors := r.Value.([][]float32)

	core.AssertTrue(t, r.OK)
	core.AssertEmpty(t, vectors)
}

func TestOllama_OllamaClient_EmbedDimension_Good(t *core.T) {
	client := &OllamaClient{config: OllamaConfig{Model: "mxbai-embed-large"}}

	core.AssertEqual(t, uint64(1024), client.EmbedDimension())
	core.AssertGreater(t, client.EmbedDimension(), uint64(768))
}

func TestOllama_OllamaClient_EmbedDimension_Bad(t *core.T) {
	client := &OllamaClient{config: OllamaConfig{Model: "unknown"}}

	core.AssertEqual(t, uint64(768), client.EmbedDimension())
	core.AssertNotEqual(t, uint64(1024), client.EmbedDimension())
}

func TestOllama_OllamaClient_EmbedDimension_Ugly(t *core.T) {
	client := &OllamaClient{config: OllamaConfig{Model: ""}}

	core.AssertEqual(t, uint64(768), client.EmbedDimension())
	core.AssertGreater(t, client.EmbedDimension(), uint64(0))
}

func TestOllama_OllamaClient_Model_Good(t *core.T) {
	client := &OllamaClient{config: OllamaConfig{Model: "nomic-embed-text"}}

	core.AssertEqual(t, "nomic-embed-text", client.Model())
	core.AssertNotEmpty(t, client.Model())
}

func TestOllama_OllamaClient_Model_Bad(t *core.T) {
	client := &OllamaClient{}

	core.AssertEqual(t, "", client.Model())
	core.AssertEmpty(t, client.Model())
}

func TestOllama_OllamaClient_Model_Ugly(t *core.T) {
	client := &OllamaClient{config: OllamaConfig{Model: "model/with:tag"}}

	core.AssertEqual(t, "model/with:tag", client.Model())
	core.AssertContains(t, client.Model(), ":tag")
}

func TestOllama_OllamaClient_VerifyModel_Good(t *core.T) {
	client, closeServer := testOllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[0.1]]}`))
	})
	defer closeServer()

	r := client.VerifyModel(core.Background())
	core.AssertTrue(t, r.OK)
	core.AssertNil(t, r.Value)
}

func TestOllama_OllamaClient_VerifyModel_Bad(t *core.T) {
	client, closeServer := testOllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[]}`))
	})
	defer closeServer()

	r := client.VerifyModel(core.Background())
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not available")
}

func TestOllama_OllamaClient_VerifyModel_Ugly(t *core.T) {
	client, closeServer := testOllamaClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`server failed`))
	})
	defer closeServer()

	r := client.VerifyModel(core.Background())
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "nomic-embed-text")
}
