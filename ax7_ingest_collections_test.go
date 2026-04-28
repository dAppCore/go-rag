package rag

import core "dappco.re/go"

type ax7DefaultQdrant struct {
	*mockVectorStore
	healthErr error
	closeErr  error
}

func (s *ax7DefaultQdrant) HealthCheck(core.Context) error {
	return s.healthErr
}

func (s *ax7DefaultQdrant) Close() error {
	return s.closeErr
}

type ax7DefaultOllama struct {
	*mockEmbedder
	verifyErr error
}

func (e *ax7DefaultOllama) VerifyModel(core.Context) error {
	return e.verifyErr
}

func ax7InstallDefaultClients(t *core.T, store defaultQdrantClient, embedder defaultOllamaClient) {
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

func ax7WriteFile(t *core.T, path string, content string) {
	t.Helper()
	result := core.WriteFile(path, []byte(content), 0o644)
	core.AssertTrue(t, result.OK, result.Error())
}

func TestAX7_DefaultIngestConfig_Bad(t *core.T) {
	cfg := DefaultIngestConfig()

	core.AssertNotEqual(t, "", cfg.Collection)
	core.AssertNotEqual(t, 0, cfg.BatchSize)
}

func TestAX7_DefaultIngestConfig_Ugly(t *core.T) {
	cfg := DefaultIngestConfig()
	cfg.Collection = "mutated"

	core.AssertEqual(t, "hostuk-docs", DefaultIngestConfig().Collection)
	core.AssertEqual(t, "mutated", cfg.Collection)
}

func TestAX7_ListCollections_Bad(t *core.T) {
	store := newMockVectorStore()
	store.listErr = core.NewError("list failed")
	names, err := ListCollections(core.Background(), store)

	core.AssertError(t, err)
	core.AssertNil(t, names)
}

func TestAX7_ListCollections_Ugly(t *core.T) {
	store := newMockVectorStore()
	names, err := ListCollections(core.Background(), store)

	core.AssertNoError(t, err)
	core.AssertEmpty(t, names)
}

func TestAX7_ListCollectionsSeq_Bad(t *core.T) {
	store := newMockVectorStore()
	store.listErr = core.NewError("list failed")
	seq, err := ListCollectionsSeq(core.Background(), store)

	core.AssertError(t, err)
	core.AssertNil(t, seq)
}

func TestAX7_ListCollectionsSeq_Ugly(t *core.T) {
	store := newMockVectorStore()
	seq, err := ListCollectionsSeq(core.Background(), store)
	count := 0
	for range seq {
		count++
	}

	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_DeleteCollection_Bad(t *core.T) {
	store := newMockVectorStore()
	store.deleteErr = core.NewError("delete failed")
	err := DeleteCollection(core.Background(), store, "docs")

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "delete failed")
}

func TestAX7_DeleteCollection_Ugly(t *core.T) {
	store := newMockVectorStore()
	err := DeleteCollection(core.Background(), store, "")

	core.AssertNoError(t, err)
	core.AssertLen(t, store.deleteCalls, 1)
}

func TestAX7_CollectionStats_Bad(t *core.T) {
	store := newMockVectorStore()
	store.infoErr = core.NewError("info failed")
	info, err := CollectionStats(core.Background(), store, "docs")

	core.AssertError(t, err)
	core.AssertNil(t, info)
}

func TestAX7_CollectionStats_Ugly(t *core.T) {
	store := newMockVectorStore()
	store.collections["empty"] = 384
	info, err := CollectionStats(core.Background(), store, "empty")

	core.AssertNoError(t, err)
	core.AssertEqual(t, uint64(0), info.PointCount)
}

func TestAX7_Ingest_Bad(t *core.T) {
	dir := t.TempDir()
	path := core.PathJoin(dir, "not-dir.md")
	ax7WriteFile(t, path, "content")
	_, err := Ingest(core.Background(), newMockVectorStore(), newMockEmbedder(2), IngestConfig{Directory: path, Collection: "docs", Chunk: DefaultChunkConfig()}, nil)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not a directory")
}

func TestAX7_Ingest_Ugly(t *core.T) {
	dir := t.TempDir()
	_, err := Ingest(core.Background(), newMockVectorStore(), newMockEmbedder(2), IngestConfig{Directory: dir, Collection: "docs", Chunk: DefaultChunkConfig()}, nil)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no matching files")
}

func TestAX7_IngestFile_Bad(t *core.T) {
	count, err := IngestFile(core.Background(), newMockVectorStore(), newMockEmbedder(2), "docs", core.PathJoin(t.TempDir(), "missing.md"), DefaultChunkConfig())

	core.AssertError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_IngestFile_Ugly(t *core.T) {
	path := core.PathJoin(t.TempDir(), "empty.md")
	ax7WriteFile(t, path, " \n\t")
	count, err := IngestFile(core.Background(), newMockVectorStore(), newMockEmbedder(2), "docs", path, DefaultChunkConfig())

	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_IngestDirWith_Bad(t *core.T) {
	err := IngestDirWith(core.Background(), newMockVectorStore(), newMockEmbedder(2), core.PathJoin(t.TempDir(), "missing"), "docs", false)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "accessing directory")
}

func TestAX7_IngestDirWith_Ugly(t *core.T) {
	dir := t.TempDir()
	err := IngestDirWith(core.Background(), newMockVectorStore(), newMockEmbedder(2), dir, "docs", true)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no matching files")
}

func TestAX7_IngestFileWith_Bad(t *core.T) {
	count, err := IngestFileWith(core.Background(), newMockVectorStore(), newMockEmbedder(2), core.PathJoin(t.TempDir(), "missing.md"), "docs")

	core.AssertError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_IngestFileWith_Ugly(t *core.T) {
	path := core.PathJoin(t.TempDir(), "empty.md")
	ax7WriteFile(t, path, "")
	count, err := IngestFileWith(core.Background(), newMockVectorStore(), newMockEmbedder(2), path, "docs")

	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_QueryDocs_Good(t *core.T) {
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore()}
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 1, Payload: map[string]any{"text": "answer", "source": "a.md", "chunk_index": 0}}}, nil
	}
	ax7InstallDefaultClients(t, store, &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2)})
	results, err := QueryDocs(core.Background(), "question", "docs", 3)

	core.AssertNoError(t, err)
	core.AssertEqual(t, "answer", results[0].Text)
}

func TestAX7_QueryDocs_Bad(t *core.T) {
	oldQdrant := newDefaultQdrantClient
	newDefaultQdrantClient = func() (defaultQdrantClient, error) { return nil, core.NewError("factory failed") }
	t.Cleanup(func() { newDefaultQdrantClient = oldQdrant })
	results, err := QueryDocs(core.Background(), "question", "docs", 3)

	core.AssertError(t, err)
	core.AssertNil(t, results)
}

func TestAX7_QueryDocs_Ugly(t *core.T) {
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore(), closeErr: core.NewError("close ignored")}
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) { return nil, nil }
	ax7InstallDefaultClients(t, store, &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2)})
	results, err := QueryDocs(core.Background(), "question", "docs", -1)

	core.AssertNoError(t, err)
	core.AssertEmpty(t, results)
}

func TestAX7_QueryDocsContext_Good(t *core.T) {
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore()}
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 1, Payload: map[string]any{"text": "context answer", "source": "a.md", "chunk_index": 0}}}, nil
	}
	ax7InstallDefaultClients(t, store, &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2)})
	text, err := QueryDocsContext(core.Background(), "question", "docs", 3)

	core.AssertNoError(t, err)
	core.AssertContains(t, text, "context answer")
}

func TestAX7_QueryDocsContext_Bad(t *core.T) {
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore()}
	embedder := &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2)}
	embedder.embedErr = core.NewError("embed failed")
	ax7InstallDefaultClients(t, store, embedder)
	text, err := QueryDocsContext(core.Background(), "question", "docs", 3)

	core.AssertError(t, err)
	core.AssertEqual(t, "", text)
}

func TestAX7_QueryDocsContext_Ugly(t *core.T) {
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore()}
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) { return nil, nil }
	ax7InstallDefaultClients(t, store, &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2)})
	text, err := QueryDocsContext(core.Background(), "question", "docs", 3)

	core.AssertNoError(t, err)
	core.AssertEqual(t, "", text)
}

func TestAX7_IngestDirectory_Good(t *core.T) {
	dir := t.TempDir()
	ax7WriteFile(t, core.PathJoin(dir, "guide.md"), "# Guide\n\nHello world.")
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore()}
	ax7InstallDefaultClients(t, store, &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2)})
	err := IngestDirectory(core.Background(), dir, "docs", false)

	core.AssertNoError(t, err)
	core.AssertLen(t, store.points["docs"], 1)
}

func TestAX7_IngestDirectory_Bad(t *core.T) {
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore(), healthErr: core.NewError("health failed")}
	ax7InstallDefaultClients(t, store, &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2)})
	err := IngestDirectory(core.Background(), t.TempDir(), "docs", false)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "health check")
}

func TestAX7_IngestDirectory_Ugly(t *core.T) {
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore()}
	embedder := &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2), verifyErr: core.NewError("model missing")}
	ax7InstallDefaultClients(t, store, embedder)
	err := IngestDirectory(core.Background(), t.TempDir(), "docs", false)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "model missing")
}

func TestAX7_IngestSingleFile_Good(t *core.T) {
	path := core.PathJoin(t.TempDir(), "guide.md")
	ax7WriteFile(t, path, "# Guide\n\nHello world.")
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore()}
	ax7InstallDefaultClients(t, store, &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2)})
	count, err := IngestSingleFile(core.Background(), path, "docs")

	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, count)
}

func TestAX7_IngestSingleFile_Bad(t *core.T) {
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore(), healthErr: core.NewError("health failed")}
	ax7InstallDefaultClients(t, store, &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2)})
	count, err := IngestSingleFile(core.Background(), core.PathJoin(t.TempDir(), "guide.md"), "docs")

	core.AssertError(t, err)
	core.AssertEqual(t, 0, count)
}

func TestAX7_IngestSingleFile_Ugly(t *core.T) {
	store := &ax7DefaultQdrant{mockVectorStore: newMockVectorStore()}
	embedder := &ax7DefaultOllama{mockEmbedder: newMockEmbedder(2), verifyErr: core.NewError("model missing")}
	ax7InstallDefaultClients(t, store, embedder)
	count, err := IngestSingleFile(core.Background(), core.PathJoin(t.TempDir(), "guide.md"), "docs")

	core.AssertError(t, err)
	core.AssertEqual(t, 0, count)
}
