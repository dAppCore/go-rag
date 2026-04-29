package rag

import (
	core "dappco.re/go"
	"github.com/qdrant/go-client/qdrant"
)

func ExampleDefaultQdrantConfig() {
	cfg := DefaultQdrantConfig()
	core.Println(cfg.Host, cfg.Port, cfg.UseTLS)
	// Output: localhost 6334 false
}

func ExampleNewQdrantStore() {
	client, err := NewQdrantStore("http://localhost:6333")
	if client != nil {
		defer client.Close()
	}
	core.Println(err == nil, client.config.Port)
	// Output: true 6334
}

func ExampleNewQdrantClient() {
	client, err := NewQdrantClient(DefaultQdrantConfig())
	if client != nil {
		defer client.Close()
	}
	core.Println(err == nil, client != nil)
	// Output: true true
}

func ExampleQdrantClient_Close() {
	fake := &qdrantTestAPI{}
	err := (&QdrantClient{client: fake}).Close()
	core.Println(err == nil, fake.closeCalled)
	// Output: true true
}

func ExampleQdrantClient_HealthCheck() {
	err := (&QdrantClient{client: &qdrantTestAPI{}}).HealthCheck(core.Background())
	core.Println(err == nil)
	// Output: true
}

func ExampleQdrantClient_ListCollections() {
	names, err := (&QdrantClient{client: &qdrantTestAPI{collections: []string{"docs"}}}).ListCollections(core.Background())
	core.Println(err == nil, names[0])
	// Output: true docs
}

func ExampleQdrantClient_CollectionExists() {
	exists, err := (&QdrantClient{client: &qdrantTestAPI{exists: true}}).CollectionExists(core.Background(), "docs")
	core.Println(err == nil, exists)
	// Output: true true
}

func ExampleQdrantClient_CreateCollection() {
	fake := &qdrantTestAPI{}
	err := (&QdrantClient{client: fake}).CreateCollection(core.Background(), "docs", 768)
	core.Println(err == nil, fake.created.GetCollectionName())
	// Output: true docs
}

func ExampleQdrantClient_DeleteCollection() {
	fake := &qdrantTestAPI{}
	err := (&QdrantClient{client: fake}).DeleteCollection(core.Background(), "docs")
	core.Println(err == nil, fake.deleted)
	// Output: true docs
}

func ExampleQdrantClient_CollectionInfo() {
	fake := &qdrantTestAPI{info: qdrantTestCollectionInfo(3, 768, qdrant.CollectionStatus_Green)}
	info, err := (&QdrantClient{client: fake}).CollectionInfo(core.Background(), "docs")
	core.Println(err == nil, info.VectorSize, info.Status)
	// Output: true 768 green
}

func ExampleQdrantClient_UpsertPoints() {
	fake := &qdrantTestAPI{}
	err := (&QdrantClient{client: fake}).UpsertPoints(core.Background(), "docs", []Point{{ID: "p1", Vector: []float32{0.1}, Payload: map[string]any{"text": "alpha"}}})
	core.Println(err == nil, fake.upsert.GetCollectionName())
	// Output: true docs
}

func ExampleSearchResult_GetText() {
	result := SearchResult{Payload: map[string]any{"text": "payload text"}}
	core.Println(result.GetText())
	// Output: payload text
}

func ExampleSearchResult_GetScore() {
	result := SearchResult{Score: 0.6}
	core.Println(result.GetScore())
	// Output: 0.6
}

func ExampleSearchResult_GetSource() {
	result := SearchResult{Payload: map[string]any{"source": "guide.md"}}
	core.Println(result.GetSource())
	// Output: guide.md
}

func ExampleSearchResult_HasChunkIndex() {
	result := SearchResult{ChunkIndex: 0, ChunkIndexPresent: true}
	core.Println(result.HasChunkIndex())
	// Output: true
}

func ExampleSearchResult_GetChunkIndex() {
	result := SearchResult{Payload: map[string]any{"chunk_index": float64(7)}}
	core.Println(result.GetChunkIndex())
	// Output: 7
}

func ExampleSearchResult_GetSection() {
	result := SearchResult{Payload: map[string]any{"section": "Intro"}}
	core.Println(result.GetSection())
	// Output: Intro
}

func ExampleSearchResult_GetCategory() {
	result := SearchResult{Payload: map[string]any{"category": "docs"}}
	core.Println(result.GetCategory())
	// Output: docs
}

func ExampleQdrantClient_Search() {
	fake := &qdrantTestAPI{results: []*qdrant.ScoredPoint{{
		Id:    qdrant.NewID("point-1"),
		Score: 0.9,
		Payload: map[string]*qdrant.Value{
			"text":        qdrant.NewValueString("alpha"),
			"source":      qdrant.NewValueString("guide.md"),
			"chunk_index": qdrant.NewValueInt(2),
		},
	}}}
	results, err := (&QdrantClient{client: fake}).Search(core.Background(), "docs", []float32{0.1}, 5, nil)
	core.Println(err == nil, results[0].Text, results[0].ChunkIndex)
	// Output: true alpha 2
}

func ExampleQdrantClient_Add() {
	fake := &qdrantTestAPI{}
	err := (&QdrantClient{client: fake}).Add(core.Background(), "docs", []Vector{{ID: "p1", Values: []float32{0.1}, Payload: map[string]any{"text": "alpha"}}})
	core.Println(err == nil, fake.upsert.GetCollectionName())
	// Output: true docs
}
