package rag

import core "dappco.re/go"

func TestAX7_Category_Good(t *core.T) {
	category := Category("docs/flux/button.md")

	core.AssertEqual(t, "ui-component", category)
	core.AssertNotEqual(t, "documentation", category)
}

func TestAX7_Category_Bad(t *core.T) {
	category := Category("")

	core.AssertEqual(t, "documentation", category)
	core.AssertNotEqual(t, "task", category)
}

func TestAX7_Category_Ugly(t *core.T) {
	category := Category("BRAND/MASCOT/README.MD")

	core.AssertEqual(t, "brand", category)
	core.AssertNotEqual(t, "documentation", category)
}

func TestAX7_ChunkID_Good(t *core.T) {
	id := ChunkID("docs/guide.md", 1, "Go uses goroutines.")

	core.AssertNotEmpty(t, id)
	core.AssertEqual(t, id, ChunkID("docs/guide.md", 1, "Go uses goroutines."))
}

func TestAX7_ChunkID_Bad(t *core.T) {
	id := ChunkID("docs/guide.md", 1, "Go uses goroutines.")

	core.AssertNotEqual(t, id, ChunkID("docs/guide.md", 2, "Go uses goroutines."))
	core.AssertNotEqual(t, id, ChunkID("other.md", 1, "Go uses goroutines."))
}

func TestAX7_ChunkID_Ugly(t *core.T) {
	prefix := core.Concat("x", repeatString("a", 120))
	id := ChunkID("unicode.md", 0, prefix+"left")

	core.AssertEqual(t, id, ChunkID("unicode.md", 0, prefix+"right"))
	core.AssertLen(t, id, 32)
}

func TestAX7_ChunkMarkdown_Good(t *core.T) {
	chunks := ChunkMarkdown("# Title\n\nThis is body text.", ChunkConfig{Size: 100, Overlap: 0})

	core.AssertLen(t, chunks, 1)
	core.AssertEqual(t, "Title", chunks[0].Section)
	core.AssertContains(t, chunks[0].Text, "body text")
}

func TestAX7_ChunkMarkdown_Bad(t *core.T) {
	chunks := ChunkMarkdown("", DefaultChunkConfig())

	core.AssertEmpty(t, chunks)
	core.AssertEqual(t, 0, len(chunks))
}

func TestAX7_ChunkMarkdown_Ugly(t *core.T) {
	text := "## Long\n\n" + repeatString("oversized ", 80)
	chunks := ChunkMarkdown(text, ChunkConfig{Size: 40, Overlap: 10})

	core.AssertGreater(t, len(chunks), 1)
	core.AssertEqual(t, "Long", chunks[0].Section)
}

func TestAX7_ChunkMarkdownSeq_Good(t *core.T) {
	var chunks []Chunk
	for chunk := range ChunkMarkdownSeq("## A\n\nAlpha.\n\n## B\n\nBeta.", ChunkConfig{Size: 80}) {
		chunks = append(chunks, chunk)
	}

	core.AssertLen(t, chunks, 2)
	core.AssertEqual(t, "A", chunks[0].Section)
}

func TestAX7_ChunkMarkdownSeq_Bad(t *core.T) {
	var chunks []Chunk
	for chunk := range ChunkMarkdownSeq("   \n\n\t", DefaultChunkConfig()) {
		chunks = append(chunks, chunk)
	}

	core.AssertEmpty(t, chunks)
	core.AssertEqual(t, 0, len(chunks))
}

func TestAX7_ChunkMarkdownSeq_Ugly(t *core.T) {
	count := 0
	for range ChunkMarkdownSeq("## A\n\n"+repeatString("word ", 50), ChunkConfig{Size: 20}) {
		count++
		break
	}

	core.AssertEqual(t, 1, count)
	core.AssertTrue(t, count < 2)
}

func TestAX7_ChunkBySentencesSeq_Good(t *core.T) {
	var chunks []Chunk
	for chunk := range ChunkBySentencesSeq("One sentence. Two sentence.", ChunkConfig{Size: 20}) {
		chunks = append(chunks, chunk)
	}

	core.AssertGreaterOrEqual(t, len(chunks), 1)
	core.AssertContains(t, chunks[0].Text, "One")
}

func TestAX7_ChunkBySentencesSeq_Bad(t *core.T) {
	var chunks []Chunk
	for chunk := range ChunkBySentencesSeq("", DefaultChunkConfig()) {
		chunks = append(chunks, chunk)
	}

	core.AssertEmpty(t, chunks)
	core.AssertEqual(t, 0, len(chunks))
}

func TestAX7_ChunkBySentencesSeq_Ugly(t *core.T) {
	var chunks []Chunk
	for chunk := range ChunkBySentencesSeq("Alpha. Beta. Gamma.", ChunkConfig{Size: 8, Overlap: 6}) {
		chunks = append(chunks, chunk)
	}

	core.AssertGreater(t, len(chunks), 1)
	core.AssertContains(t, chunks[1].Text, "Alpha")
}

func TestAX7_ChunkByParagraphsSeq_Good(t *core.T) {
	var chunks []Chunk
	for chunk := range ChunkByParagraphsSeq("First paragraph.\n\nSecond paragraph.", ChunkConfig{Size: 80}) {
		chunks = append(chunks, chunk)
	}

	core.AssertLen(t, chunks, 1)
	core.AssertContains(t, chunks[0].Text, "Second paragraph")
}

func TestAX7_ChunkByParagraphsSeq_Bad(t *core.T) {
	var chunks []Chunk
	for chunk := range ChunkByParagraphsSeq("\n\n", DefaultChunkConfig()) {
		chunks = append(chunks, chunk)
	}

	core.AssertEmpty(t, chunks)
	core.AssertEqual(t, 0, len(chunks))
}

func TestAX7_ChunkByParagraphsSeq_Ugly(t *core.T) {
	var chunks []Chunk
	for chunk := range ChunkByParagraphsSeq("One paragraph.\n\n"+repeatString("long paragraph ", 20), ChunkConfig{Size: 30, Overlap: 8}) {
		chunks = append(chunks, chunk)
	}

	core.AssertGreater(t, len(chunks), 1)
	core.AssertEqual(t, 0, chunks[0].Index)
}

func TestAX7_DefaultChunkConfig_Bad(t *core.T) {
	cfg := DefaultChunkConfig()

	core.AssertNotEqual(t, 0, cfg.Size)
	core.AssertNotEqual(t, 0, cfg.Overlap)
}

func TestAX7_DefaultChunkConfig_Ugly(t *core.T) {
	cfg := DefaultChunkConfig()
	cfg.Size = -1

	core.AssertEqual(t, 500, DefaultChunkConfig().Size)
	core.AssertEqual(t, -1, cfg.Size)
}

func TestAX7_FileExtensions_Bad(t *core.T) {
	extensions := FileExtensions()

	core.AssertFalse(t, core.Contains(core.Join(",", extensions...), ".go"))
	core.AssertFalse(t, core.Contains(core.Join(",", extensions...), ".exe"))
}

func TestAX7_FileExtensions_Ugly(t *core.T) {
	extensions := FileExtensions()
	extensions[0] = ".mutated"

	core.AssertEqual(t, ".md", FileExtensions()[0])
	core.AssertEqual(t, ".mutated", extensions[0])
}

func TestAX7_ShouldProcess_Good(t *core.T) {
	ok := ShouldProcess("docs/guide.md")

	core.AssertTrue(t, ok)
	core.AssertTrue(t, ShouldProcess("notes.txt"))
}

func TestAX7_ShouldProcess_Bad(t *core.T) {
	ok := ShouldProcess("cmd/main.go")

	core.AssertFalse(t, ok)
	core.AssertFalse(t, ShouldProcess("Makefile"))
}

func TestAX7_ShouldProcess_Ugly(t *core.T) {
	ok := ShouldProcess("DOCS/README.MARKDOWN")

	core.AssertTrue(t, ok)
	core.AssertTrue(t, ShouldProcess("archive.PDF"))
}
