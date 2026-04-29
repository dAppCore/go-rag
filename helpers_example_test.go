package rag

import core "dappco.re/go"

func ExampleQueryWith() {
	store := newMockVectorStore()
	store.points["docs"] = []Point{{ID: "p1", Vector: []float32{0.1}, Payload: map[string]any{"text": "answer", "source": "guide.md", "chunk_index": 0}}}
	results, err := QueryWith(core.Background(), store, newMockEmbedder(2), "guide", "docs", 3)
	core.Println(err == nil, results[0].Text)
	// Output: true answer
}

func ExampleQueryContextWith() {
	store := newMockVectorStore()
	store.points["docs"] = []Point{{ID: "p1", Vector: []float32{0.1}, Payload: map[string]any{"text": "answer", "source": "guide.md", "chunk_index": 0}}}
	text, err := QueryContextWith(core.Background(), store, newMockEmbedder(2), "guide", "docs", 3)
	core.Println(err == nil, core.Contains(text, "answer"))
	// Output: true true
}

func ExampleIngestDirWith() {
	dirResult := core.MkdirTemp("", "rag-dir-helper-example-*")
	dir := dirResult.Value.(string)
	defer core.RemoveAll(dir)
	core.WriteFile(core.PathJoin(dir, "guide.md"), []byte("## Guide\n\nHello world."), 0o644)

	store := newMockVectorStore()
	err := IngestDirWith(core.Background(), store, newMockEmbedder(2), dir, "docs", false)
	core.Println(err == nil, len(store.points["docs"]))
	// Output: true 1
}

func ExampleIngestFileWith() {
	dirResult := core.MkdirTemp("", "rag-file-helper-example-*")
	dir := dirResult.Value.(string)
	defer core.RemoveAll(dir)
	path := core.PathJoin(dir, "guide.md")
	core.WriteFile(path, []byte("## Guide\n\nHello world."), 0o644)

	count, err := IngestFileWith(core.Background(), newMockVectorStore(), newMockEmbedder(2), path, "docs")
	core.Println(err == nil, count)
	// Output: true 1
}

func ExampleQueryDocs() {
	oldQdrant := newDefaultQdrantClient
	oldOllama := newDefaultOllamaClient
	defer func() {
		newDefaultQdrantClient = oldQdrant
		newDefaultOllamaClient = oldOllama
	}()
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 1, Payload: map[string]any{"text": "answer", "source": "guide.md", "chunk_index": 0}}}, nil
	}
	newDefaultQdrantClient = func() (defaultQdrantClient, error) { return store, nil }
	newDefaultOllamaClient = func() (defaultOllamaClient, error) { return &testDefaultOllama{mockEmbedder: newMockEmbedder(2)}, nil }

	results, err := QueryDocs(core.Background(), "guide", "docs", 3)
	core.Println(err == nil, results[0].Text)
	// Output: true answer
}

func ExampleQueryDocsContext() {
	oldQdrant := newDefaultQdrantClient
	oldOllama := newDefaultOllamaClient
	defer func() {
		newDefaultQdrantClient = oldQdrant
		newDefaultOllamaClient = oldOllama
	}()
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 1, Payload: map[string]any{"text": "context answer", "source": "guide.md", "chunk_index": 0}}}, nil
	}
	newDefaultQdrantClient = func() (defaultQdrantClient, error) { return store, nil }
	newDefaultOllamaClient = func() (defaultOllamaClient, error) { return &testDefaultOllama{mockEmbedder: newMockEmbedder(2)}, nil }

	text, err := QueryDocsContext(core.Background(), "guide", "docs", 3)
	core.Println(err == nil, core.Contains(text, "context answer"))
	// Output: true true
}

func ExampleIngestDirectory() {
	oldQdrant := newDefaultQdrantClient
	oldOllama := newDefaultOllamaClient
	defer func() {
		newDefaultQdrantClient = oldQdrant
		newDefaultOllamaClient = oldOllama
	}()
	dirResult := core.MkdirTemp("", "rag-default-dir-example-*")
	dir := dirResult.Value.(string)
	defer core.RemoveAll(dir)
	core.WriteFile(core.PathJoin(dir, "guide.md"), []byte("## Guide\n\nHello world."), 0o644)
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	newDefaultQdrantClient = func() (defaultQdrantClient, error) { return store, nil }
	newDefaultOllamaClient = func() (defaultOllamaClient, error) { return &testDefaultOllama{mockEmbedder: newMockEmbedder(2)}, nil }

	err := IngestDirectory(core.Background(), dir, "docs", false)
	core.Println(err == nil, len(store.points["docs"]))
	// Output: true 1
}

func ExampleIngestSingleFile() {
	oldQdrant := newDefaultQdrantClient
	oldOllama := newDefaultOllamaClient
	defer func() {
		newDefaultQdrantClient = oldQdrant
		newDefaultOllamaClient = oldOllama
	}()
	dirResult := core.MkdirTemp("", "rag-default-file-example-*")
	dir := dirResult.Value.(string)
	defer core.RemoveAll(dir)
	path := core.PathJoin(dir, "guide.md")
	core.WriteFile(path, []byte("## Guide\n\nHello world."), 0o644)
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	newDefaultQdrantClient = func() (defaultQdrantClient, error) { return store, nil }
	newDefaultOllamaClient = func() (defaultOllamaClient, error) { return &testDefaultOllama{mockEmbedder: newMockEmbedder(2)}, nil }

	count, err := IngestSingleFile(core.Background(), path, "docs")
	core.Println(err == nil, count)
	// Output: true 1
}

func ExampleJoinResults() {
	text := JoinResults([]QueryResult{{Text: " Alpha "}, {Text: ""}, {Text: "Beta"}})
	core.Println(text)
	// Output: Alpha
	//
	// Beta
}
