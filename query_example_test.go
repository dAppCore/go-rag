package rag

import core "dappco.re/go"

func ExampleDefaultQueryConfig() {
	cfg := DefaultQueryConfig()
	core.Println(cfg.Collection, cfg.Limit)
	// Output: hostuk-docs 5
}

func ExampleQueryResult_GetText() {
	result := QueryResult{Text: "answer text"}
	core.Println(result.GetText())
	// Output: answer text
}

func ExampleQueryResult_GetScore() {
	result := QueryResult{Score: 0.8}
	core.Println(result.GetScore())
	// Output: 0.8
}

func ExampleQueryResult_GetSource() {
	result := QueryResult{Source: "docs/source.md"}
	core.Println(result.GetSource())
	// Output: docs/source.md
}

func ExampleQueryResult_HasChunkIndex() {
	result := QueryResult{ChunkIndex: 0, ChunkIndexPresent: true}
	core.Println(result.HasChunkIndex())
	// Output: true
}

func ExampleQueryResult_GetChunkIndex() {
	result := QueryResult{ChunkIndex: 5, ChunkIndexPresent: true}
	core.Println(result.GetChunkIndex())
	// Output: 5
}

func ExampleRank() {
	results := []QueryResult{{Text: "low", Score: 0.1}, {Text: "high", Score: 0.9}}
	ranked := Rank(results, 1)
	core.Println(ranked[0].Text)
	// Output: high
}

func ExampleQuery() {
	store := newMockVectorStore()
	store.points["docs"] = []Point{{ID: "p1", Vector: []float32{0.1}, Payload: map[string]any{"text": "hit", "source": "guide.md", "chunk_index": 0}}}
	cfg := QueryConfig{Collection: "docs", Limit: 5, Threshold: 0}
	results, err := Query(core.Background(), store, newMockEmbedder(2), "guide", cfg)
	core.Println(err == nil, len(results), results[0].Text)
	// Output: true 1 hit
}

func ExampleQuerySeq() {
	store := newMockVectorStore()
	store.points["docs"] = []Point{{ID: "p1", Vector: []float32{0.1}, Payload: map[string]any{"text": "hit", "source": "guide.md", "chunk_index": 0}}}
	cfg := QueryConfig{Collection: "docs", Limit: 5, Threshold: 0}
	seq, err := QuerySeq(core.Background(), store, newMockEmbedder(2), "guide", cfg)
	count := 0
	for range seq {
		count++
	}
	core.Println(err == nil, count)
	// Output: true 1
}

func ExampleFormatResultsText() {
	text := FormatResultsText([]QueryResult{{Text: "Body", Source: "guide.md", Category: "docs", Score: 0.9}})
	core.Println(core.Contains(text, "guide.md"))
	// Output: true
}

func ExampleFormatResultsContext() {
	text := FormatResultsContext([]QueryResult{{Text: "<tag>", Source: "guide.md", Category: "docs"}})
	core.Println(core.Contains(text, "&lt;tag&gt;"))
	// Output: true
}

func ExampleFormatResultsJSON() {
	text := FormatResultsJSON([]QueryResult{{Text: "Body", Source: "guide.md", Category: "docs", Score: 0.92345}})
	core.Println(core.Contains(text, "guide.md"))
	// Output: true
}
