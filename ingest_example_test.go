package rag

import core "dappco.re/go"

func ExampleDefaultIngestConfig() {
	cfg := DefaultIngestConfig()
	core.Println(cfg.Collection, cfg.BatchSize)
	// Output: hostuk-docs 100
}

func ExampleIngest() {
	dirResult := core.MkdirTemp("", "rag-ingest-example-*")
	dir := dirResult.Value.(string)
	defer core.RemoveAll(dir)
	core.WriteFile(core.PathJoin(dir, "guide.md"), []byte("## Guide\n\nHello world."), 0o644)

	cfg := IngestConfig{Directory: dir, Collection: "docs", Chunk: DefaultChunkConfig(), BatchSize: 10}
	stats, err := Ingest(core.Background(), newMockVectorStore(), newMockEmbedder(2), cfg, nil)
	core.Println(err == nil, stats.Files, stats.Chunks)
	// Output: true 1 1
}

func ExampleIngestFile() {
	dirResult := core.MkdirTemp("", "rag-file-example-*")
	dir := dirResult.Value.(string)
	defer core.RemoveAll(dir)
	path := core.PathJoin(dir, "guide.md")
	core.WriteFile(path, []byte("## Guide\n\nHello world."), 0o644)

	count, err := IngestFile(core.Background(), newMockVectorStore(), newMockEmbedder(2), "docs", path, DefaultChunkConfig())
	core.Println(err == nil, count)
	// Output: true 1
}
