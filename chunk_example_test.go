package rag

import core "dappco.re/go"

func ExampleDefaultChunkConfig() {
	cfg := DefaultChunkConfig()
	core.Println(cfg.Size, cfg.Overlap)
	// Output: 500 50
}

func ExampleChunkMarkdown() {
	chunks := ChunkMarkdown("## Intro\n\nGo uses goroutines.", ChunkConfig{Size: 100})
	core.Println(len(chunks), chunks[0].Section)
	// Output: 1 Intro
}

func ExampleChunkMarkdownSeq() {
	count := 0
	for chunk := range ChunkMarkdownSeq("## A\n\nAlpha.\n\n## B\n\nBeta.", ChunkConfig{Size: 80}) {
		if chunk.Text != "" {
			count++
		}
	}
	core.Println(count)
	// Output: 2
}

func ExampleChunkBySentences() {
	chunks := ChunkBySentences("Alpha. Beta.", ChunkConfig{Size: 8, Overlap: 0})
	core.Println(len(chunks), chunks[0].Text)
	// Output: 2 Alpha.
}

func ExampleChunkBySentencesSeq() {
	count := 0
	for range ChunkBySentencesSeq("Alpha. Beta.", ChunkConfig{Size: 8}) {
		count++
	}
	core.Println(count)
	// Output: 2
}

func ExampleChunkByParagraphs() {
	chunks := ChunkByParagraphs("First paragraph.\n\nSecond paragraph.", ChunkConfig{Size: 80})
	core.Println(len(chunks), core.Contains(chunks[0].Text, "Second"))
	// Output: 1 true
}

func ExampleChunkByParagraphsSeq() {
	count := 0
	for range ChunkByParagraphsSeq("First.\n\nSecond.", ChunkConfig{Size: 8}) {
		count++
	}
	core.Println(count)
	// Output: 2
}

func ExampleCategory() {
	core.Println(Category("docs/architecture/migration.md"))
	// Output: architecture
}

func ExampleChunkID() {
	id := ChunkID("docs/guide.md", 0, "Go uses goroutines.")
	core.Println(len(id))
	// Output: 32
}

func ExampleFileExtensions() {
	core.Println(core.Join(",", FileExtensions()...))
	// Output: .md,.markdown,.pdf,.txt
}

func ExampleShouldProcess() {
	core.Println(ShouldProcess("docs/guide.md"), ShouldProcess("cmd/main.go"))
	// Output: true false
}
