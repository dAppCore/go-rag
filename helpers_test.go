package rag

import (
	"context"
	"testing"

	"dappco.re/go"
)

// --- QueryWith tests ---

func TestHelpers_QueryWith_Good(t *testing.T) {
	t.Run("returns results from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["my-docs"] = []Point{
			{ID: "1", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "Hello from helper.", "source": "a.md", "section": "S", "category": "docs", "chunk_index": 0,
			}},
		}
		embedder := newMockEmbedder(768)

		r := QueryWith(context.Background(), store, embedder, "hello", "my-docs", 5)
		results := resultValue[[]QueryResult](t, r)

		assertLen(t, results, 1)
		assertEqual(t, "Hello from helper.", results[0].Text)
	})

	t.Run("respects topK parameter", func(t *testing.T) {
		store := newMockVectorStore()
		for i := range 10 {
			store.points["col"] = append(store.points["col"], Point{
				ID:     core.Sprintf("p%d", i),
				Vector: []float32{0.1},
				Payload: map[string]any{
					"text": core.Sprintf("Doc %d", i), "source": "d.md", "section": "", "category": "docs", "chunk_index": i,
				},
			})
		}
		embedder := newMockEmbedder(768)

		r := QueryWith(context.Background(), store, embedder, "test", "col", 3)
		results := resultValue[[]QueryResult](t, r)

		assertLen(t, results, 3)
	})

	t.Run("embedder error propagates", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = core.E("mock.embed", "embed failed", nil)

		r := QueryWith(context.Background(), store, embedder, "test", "col", 5)

		assertError(t, r)
	})

	t.Run("search error propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.searchErr = core.E("mock.search", "search failed", nil)
		embedder := newMockEmbedder(768)

		r := QueryWith(context.Background(), store, embedder, "test", "col", 5)

		assertError(t, r)
	})
}

// --- QueryContextWith tests ---

func TestHelpers_QueryContextWith_Good(t *testing.T) {
	t.Run("returns formatted context string", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["ctx-col"] = []Point{
			{ID: "1", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "Context content.", "source": "guide.md", "section": "Intro", "category": "docs", "chunk_index": 0,
			}},
		}
		embedder := newMockEmbedder(768)

		r := QueryContextWith(context.Background(), store, embedder, "question", "ctx-col", 5)
		result := resultValue[string](t, r)

		assertContains(t, result, "<retrieved_context>")
		assertContains(t, result, "Context content.")
		assertContains(t, result, "</retrieved_context>")
	})

	t.Run("empty results return empty string", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		r := QueryContextWith(context.Background(), store, embedder, "question", "empty", 5)
		result := resultValue[string](t, r)

		assertEqual(t, "", result)
	})

	t.Run("error from query propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.searchErr = core.E("mock.search", "broken", nil)
		embedder := newMockEmbedder(768)

		r := QueryContextWith(context.Background(), store, embedder, "question", "col", 5)

		assertError(t, r)
	})
}

// --- IngestDirWith tests ---

func TestHelpers_IngestDirWith_Good(t *testing.T) {
	t.Run("ingests directory into collection", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "readme.md"), "## README\n\nProject overview.\n")
		writeFile(t, core.JoinPath(dir, "guide.md"), "## Guide\n\nStep by step.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, dir, "project-docs", false)

		assertNoError(t, err)
		points := store.allPoints("project-docs")
		assertLen(t, points, 2)
	})

	t.Run("recreate flag deletes existing collection", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Doc\n\nContent.\n")

		store := newMockVectorStore()
		store.collections["col"] = 768
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, dir, "col", true)

		assertNoError(t, err)
		assertLen(t, store.deleteCalls, 1)
		assertEqual(t, "col", store.deleteCalls[0])
	})

	t.Run("error from ingest propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.existsErr = core.E("mock.collections.exists", "exists check failed", nil)
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, "/tmp", "col", false)

		assertError(t, err)
	})

	t.Run("nonexistent directory returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, "/tmp/nonexistent-go-rag-test-dir", "col", false)

		assertError(t, err)
	})
}

// --- IngestFileWith tests ---

func TestHelpers_IngestFileWith_Good(t *testing.T) {
	t.Run("ingests a single file", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "single.md")
		writeFile(t, path, "## Title\n\nFile content for testing.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		r := IngestFileWith(context.Background(), store, embedder, path, "col")
		count := resultValue[int](t, r)

		assertEqual(t, 1, count)

		points := store.allPoints("col")
		assertLen(t, points, 1)
		assertContains(t, points[0].Payload["text"], "File content for testing.")
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		r := IngestFileWith(context.Background(), store, embedder, "/tmp/nonexistent-test-file.md", "col")

		assertError(t, r)
	})

	t.Run("empty file returns zero count", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "empty.md")
		writeFile(t, path, "  \n  ")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		r := IngestFileWith(context.Background(), store, embedder, path, "col")
		count := resultValue[int](t, r)

		assertEqual(t, 0, count)
	})

	t.Run("embedder error propagates", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "doc.md")
		writeFile(t, path, "## Title\n\nContent.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = core.E("mock.embed", "embed broken", nil)

		r := IngestFileWith(context.Background(), store, embedder, path, "col")

		assertError(t, r)
	})

	t.Run("store error propagates", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "doc.md")
		writeFile(t, path, "## Title\n\nContent.\n")

		store := newMockVectorStore()
		store.upsertErr = core.E("mock.upsert", "upsert broken", nil)
		embedder := newMockEmbedder(768)

		r := IngestFileWith(context.Background(), store, embedder, path, "col")

		assertError(t, r)
	})
}

// --- JoinResults tests ---

func TestHelpers_JoinResults_Good(t *testing.T) {
	t.Run("joins non-empty result text with blank lines", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First result."},
			{Text: "   "},
			{Text: "Second result."},
		}

		output := JoinResults(results)

		assertEqual(t, "First result.\n\nSecond result.", output)
	})

	t.Run("works with SearchResult values", func(t *testing.T) {
		results := []SearchResult{
			{Payload: map[string]any{"text": "Alpha."}},
			{},
			{Payload: map[string]any{"text": "Beta."}},
		}

		output := JoinResults(results)

		assertEqual(t, "Alpha.\n\nBeta.", output)
	})

	t.Run("empty input returns empty string", func(t *testing.T) {
		assertEqual(t, "", JoinResults[QueryResult](nil))
		assertEqual(t, "", JoinResults([]QueryResult{}))
	})
}

type testDefaultQdrant struct {
	*mockVectorStore
	healthErr error
	closeErr  error
}

func (s *testDefaultQdrant) HealthCheck(core.Context) core.Result {
	return core.ResultOf(nil, s.healthErr)
}

func (s *testDefaultQdrant) Close() core.Result {
	return core.ResultOf(nil, s.closeErr)
}

type testDefaultOllama struct {
	*mockEmbedder
	verifyErr error
}

func (e *testDefaultOllama) VerifyModel(core.Context) core.Result {
	return core.ResultOf(nil, e.verifyErr)
}

func installDefaultClients(t *core.T, store defaultQdrantClient, embedder defaultOllamaClient) {
	t.Helper()
	oldQdrant := newDefaultQdrantClient
	oldOllama := newDefaultOllamaClient
	newDefaultQdrantClient = func() (defaultQdrantClient, error) { return store, nil }
	newDefaultOllamaClient = func() (defaultOllamaClient, error) { return embedder, nil }
	t.Cleanup(func() {
		newDefaultQdrantClient = oldQdrant
		newDefaultOllamaClient = oldOllama
	})
}

func TestHelpers_IngestDirWith_Bad(t *core.T) {
	r := IngestDirWith(core.Background(), newMockVectorStore(), newMockEmbedder(2), core.PathJoin(t.TempDir(), "missing"), "docs", false)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "accessing directory")
}

func TestHelpers_IngestDirWith_Ugly(t *core.T) {
	dir := t.TempDir()
	r := IngestDirWith(core.Background(), newMockVectorStore(), newMockEmbedder(2), dir, "docs", true)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "no matching files")
}

func TestHelpers_IngestFileWith_Bad(t *core.T) {
	r := IngestFileWith(core.Background(), newMockVectorStore(), newMockEmbedder(2), core.PathJoin(t.TempDir(), "missing.md"), "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "reading file")
}

func TestHelpers_IngestFileWith_Ugly(t *core.T) {
	path := core.PathJoin(t.TempDir(), "empty.md")
	writeFile(t, path, "")
	r := IngestFileWith(core.Background(), newMockVectorStore(), newMockEmbedder(2), path, "docs")
	count := r.Value.(int)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, 0, count)
}

func TestHelpers_QueryDocs_Good(t *core.T) {
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 1, Payload: map[string]any{"text": "answer", "source": "a.md", "chunk_index": 0}}}, nil
	}
	installDefaultClients(t, store, &testDefaultOllama{mockEmbedder: newMockEmbedder(2)})
	r := QueryDocs(core.Background(), "question", "docs", 3)
	results := r.Value.([]QueryResult)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "answer", results[0].Text)
}

func TestHelpers_QueryDocs_Bad(t *core.T) {
	oldQdrant := newDefaultQdrantClient
	newDefaultQdrantClient = func() (defaultQdrantClient, error) { return nil, core.NewError("factory failed") }
	t.Cleanup(func() { newDefaultQdrantClient = oldQdrant })
	r := QueryDocs(core.Background(), "question", "docs", 3)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "factory failed")
}

func TestHelpers_QueryDocs_Ugly(t *core.T) {
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore(), closeErr: core.NewError("close ignored")}
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) { return nil, nil }
	installDefaultClients(t, store, &testDefaultOllama{mockEmbedder: newMockEmbedder(2)})
	r := QueryDocs(core.Background(), "question", "docs", -1)
	results := r.Value.([]QueryResult)

	core.AssertTrue(t, r.OK)
	core.AssertEmpty(t, results)
}

func TestHelpers_QueryDocsContext_Good(t *core.T) {
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 1, Payload: map[string]any{"text": "context answer", "source": "a.md", "chunk_index": 0}}}, nil
	}
	installDefaultClients(t, store, &testDefaultOllama{mockEmbedder: newMockEmbedder(2)})
	r := QueryDocsContext(core.Background(), "question", "docs", 3)
	text := r.Value.(string)

	core.AssertTrue(t, r.OK)
	core.AssertContains(t, text, "context answer")
}

func TestHelpers_QueryDocsContext_Bad(t *core.T) {
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	embedder := &testDefaultOllama{mockEmbedder: newMockEmbedder(2)}
	embedder.embedErr = core.NewError("embed failed")
	installDefaultClients(t, store, embedder)
	r := QueryDocsContext(core.Background(), "question", "docs", 3)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "embedding")
}

func TestHelpers_QueryDocsContext_Ugly(t *core.T) {
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) { return nil, nil }
	installDefaultClients(t, store, &testDefaultOllama{mockEmbedder: newMockEmbedder(2)})
	r := QueryDocsContext(core.Background(), "question", "docs", 3)
	text := r.Value.(string)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "", text)
}

func TestHelpers_IngestDirectory_Good(t *core.T) {
	dir := t.TempDir()
	writeFile(t, core.PathJoin(dir, "guide.md"), "# Guide\n\nHello world.")
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	installDefaultClients(t, store, &testDefaultOllama{mockEmbedder: newMockEmbedder(2)})
	r := IngestDirectory(core.Background(), dir, "docs", false)

	core.AssertTrue(t, r.OK)
	core.AssertLen(t, store.points["docs"], 1)
}

func TestHelpers_IngestDirectory_Bad(t *core.T) {
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore(), healthErr: core.NewError("health failed")}
	installDefaultClients(t, store, &testDefaultOllama{mockEmbedder: newMockEmbedder(2)})
	r := IngestDirectory(core.Background(), t.TempDir(), "docs", false)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "health check")
}

func TestHelpers_IngestDirectory_Ugly(t *core.T) {
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	embedder := &testDefaultOllama{mockEmbedder: newMockEmbedder(2), verifyErr: core.NewError("model missing")}
	installDefaultClients(t, store, embedder)
	r := IngestDirectory(core.Background(), t.TempDir(), "docs", false)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "model missing")
}

func TestHelpers_IngestSingleFile_Good(t *core.T) {
	path := core.PathJoin(t.TempDir(), "guide.md")
	writeFile(t, path, "# Guide\n\nHello world.")
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	installDefaultClients(t, store, &testDefaultOllama{mockEmbedder: newMockEmbedder(2)})
	r := IngestSingleFile(core.Background(), path, "docs")
	count := r.Value.(int)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, 1, count)
}

func TestHelpers_IngestSingleFile_Bad(t *core.T) {
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore(), healthErr: core.NewError("health failed")}
	installDefaultClients(t, store, &testDefaultOllama{mockEmbedder: newMockEmbedder(2)})
	r := IngestSingleFile(core.Background(), core.PathJoin(t.TempDir(), "guide.md"), "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "health")
}

func TestHelpers_IngestSingleFile_Ugly(t *core.T) {
	store := &testDefaultQdrant{mockVectorStore: newMockVectorStore()}
	embedder := &testDefaultOllama{mockEmbedder: newMockEmbedder(2), verifyErr: core.NewError("model missing")}
	installDefaultClients(t, store, embedder)
	r := IngestSingleFile(core.Background(), core.PathJoin(t.TempDir(), "guide.md"), "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "model missing")
}

func TestHelpers_QueryWith_Bad(t *core.T) {
	embedder := newMockEmbedder(2)
	embedder.embedErr = core.NewError("embed failed")
	r := QueryWith(core.Background(), newMockVectorStore(), embedder, "query", "docs", 3)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "embedding")
}

func TestHelpers_QueryWith_Ugly(t *core.T) {
	store := newMockVectorStore()
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 1, Payload: map[string]any{"text": "hit"}}}, nil
	}
	r := QueryWith(core.Background(), store, newMockEmbedder(2), "query", "docs", -1)
	results := r.Value.([]QueryResult)

	core.AssertTrue(t, r.OK)
	core.AssertEmpty(t, results)
}

func TestHelpers_QueryContextWith_Bad(t *core.T) {
	embedder := newMockEmbedder(2)
	embedder.embedErr = core.NewError("embed failed")
	r := QueryContextWith(core.Background(), newMockVectorStore(), embedder, "query", "docs", 3)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "embedding")
}

func TestHelpers_QueryContextWith_Ugly(t *core.T) {
	store := newMockVectorStore()
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return nil, nil
	}
	r := QueryContextWith(core.Background(), store, newMockEmbedder(2), "query", "docs", 3)
	text := r.Value.(string)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "", text)
}

func TestHelpers_JoinResults_Bad(t *core.T) {
	output := JoinResults[QueryResult](nil)

	core.AssertEqual(t, "", output)
	core.AssertEmpty(t, output)
}

func TestHelpers_JoinResults_Ugly(t *core.T) {
	output := JoinResults([]QueryResult{{Text: "  Alpha  "}, {Text: ""}, {Text: "\nBeta\n"}})

	core.AssertEqual(t, "Alpha\n\nBeta", output)
	core.AssertNotContains(t, output, "\n\n\n")
}
