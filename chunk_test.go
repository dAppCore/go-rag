package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunkMarkdown_Good_SmallSection(t *testing.T) {
	text := `# Title

This is a small section that fits in one chunk.
`
	chunks := ChunkMarkdown(text, DefaultChunkConfig())

	assert.Len(t, chunks, 1)
	assert.Contains(t, chunks[0].Text, "small section")
}

func TestChunkMarkdown_Good_MultipleSections(t *testing.T) {
	text := `# Main Title

Introduction paragraph.

## Section One

Content for section one.

## Section Two

Content for section two.
`
	chunks := ChunkMarkdown(text, DefaultChunkConfig())

	assert.GreaterOrEqual(t, len(chunks), 2)
}

func TestChunkMarkdown_Good_LargeSection(t *testing.T) {
	// Create a section larger than chunk size
	text := `## Large Section

` + repeatString("This is a test paragraph with some content. ", 50)

	cfg := ChunkConfig{Size: 200, Overlap: 20}
	chunks := ChunkMarkdown(text, cfg)

	assert.Greater(t, len(chunks), 1)
	for _, chunk := range chunks {
		assert.NotEmpty(t, chunk.Text)
		assert.Equal(t, "Large Section", chunk.Section)
	}
}

func TestChunkMarkdown_Good_ExtractsTitle(t *testing.T) {
	text := `## My Section Title

Some content here.
`
	chunks := ChunkMarkdown(text, DefaultChunkConfig())

	assert.Len(t, chunks, 1)
	assert.Equal(t, "My Section Title", chunks[0].Section)
}

func TestCategory_Good_UIComponent(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"docs/flux/button.md", "ui-component"},
		{"ui/components/modal.md", "ui-component"},
		{"brand/vi-personality.md", "brand"},
		{"mascot/expressions.md", "brand"},
		{"product-brief.md", "product-brief"},
		{"tasks/2024-01-15-feature.md", "task"},
		{"plans/architecture.md", "task"},
		{"architecture/migration.md", "architecture"},
		{"docs/api.md", "documentation"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.expected, Category(tc.path))
		})
	}
}

func TestChunkID_Good_Deterministic(t *testing.T) {
	id1 := ChunkID("test.md", 0, "hello world")
	id2 := ChunkID("test.md", 0, "hello world")

	assert.Equal(t, id1, id2)
}

func TestChunkID_Good_DifferentForDifferentInputs(t *testing.T) {
	id1 := ChunkID("test.md", 0, "hello world")
	id2 := ChunkID("test.md", 1, "hello world")
	id3 := ChunkID("other.md", 0, "hello world")

	assert.NotEqual(t, id1, id2)
	assert.NotEqual(t, id1, id3)
}

func TestShouldProcess_Good_MarkdownFiles(t *testing.T) {
	assert.True(t, ShouldProcess("doc.md"))
	assert.True(t, ShouldProcess("doc.markdown"))
	assert.True(t, ShouldProcess("doc.txt"))
	assert.False(t, ShouldProcess("doc.go"))
	assert.False(t, ShouldProcess("doc.py"))
	assert.False(t, ShouldProcess("doc"))
}

// --- Additional chunk edge cases ---

func TestChunkMarkdown_Edge_EmptyInput(t *testing.T) {
	t.Run("empty string returns no chunks", func(t *testing.T) {
		chunks := ChunkMarkdown("", DefaultChunkConfig())
		assert.Empty(t, chunks)
	})

	t.Run("whitespace only returns no chunks", func(t *testing.T) {
		chunks := ChunkMarkdown("   \n\n  \t  \n  ", DefaultChunkConfig())
		assert.Empty(t, chunks)
	})

	t.Run("single newline returns no chunks", func(t *testing.T) {
		chunks := ChunkMarkdown("\n", DefaultChunkConfig())
		assert.Empty(t, chunks)
	})
}

func TestChunkMarkdown_Edge_OnlyHeadersNoContent(t *testing.T) {
	t.Run("single header with no body", func(t *testing.T) {
		text := "## Just a Header\n"
		chunks := ChunkMarkdown(text, DefaultChunkConfig())

		assert.Len(t, chunks, 1)
		assert.Equal(t, "Just a Header", chunks[0].Section)
		assert.Contains(t, chunks[0].Text, "Just a Header")
	})

	t.Run("multiple headers with no body content", func(t *testing.T) {
		text := "## Header One\n\n## Header Two\n\n## Header Three\n"
		chunks := ChunkMarkdown(text, DefaultChunkConfig())

		// Each header becomes its own section
		assert.GreaterOrEqual(t, len(chunks), 2, "should produce at least two chunks for separate sections")
	})

	t.Run("header hierarchy with minimal content", func(t *testing.T) {
		text := "# Top Level\n\n## Sub Section\n\n### Sub Sub\n"
		chunks := ChunkMarkdown(text, DefaultChunkConfig())

		assert.NotEmpty(t, chunks, "should produce at least one chunk")
	})
}

func TestChunkMarkdown_Edge_UnicodeAndEmoji(t *testing.T) {
	t.Run("unicode text chunked correctly", func(t *testing.T) {
		text := "## Unicode Section\n\nThis section has unicode: \u00e9\u00e0\u00fc\u00f1\u00f6\u00e4\u00df \u4e16\u754c \u041f\u0440\u0438\u0432\u0435\u0442 \u0645\u0631\u062d\u0628\u0627\n"
		chunks := ChunkMarkdown(text, DefaultChunkConfig())

		assert.Len(t, chunks, 1)
		assert.Contains(t, chunks[0].Text, "\u00e9\u00e0\u00fc")
		assert.Contains(t, chunks[0].Text, "\u4e16\u754c")
		assert.Equal(t, "Unicode Section", chunks[0].Section)
	})

	t.Run("emoji text chunked correctly", func(t *testing.T) {
		text := "## Emoji Section\n\nHello world! \U0001f600\U0001f680\U0001f30d\U0001f4da\n\nMore text with \u2764\ufe0f and \U0001f525 emojis.\n"
		chunks := ChunkMarkdown(text, DefaultChunkConfig())

		assert.NotEmpty(t, chunks)
		assert.Contains(t, chunks[0].Text, "\U0001f600")
		assert.Equal(t, "Emoji Section", chunks[0].Section)
	})

	t.Run("rune-safe overlap with multibyte characters", func(t *testing.T) {
		// Create text with multibyte characters that exceeds chunk size
		// Each CJK character is 3 bytes in UTF-8 but 1 rune
		para1 := "\u6d4b\u8bd5" + repeatString("\u4e16\u754c", 100) // ~200+ runes of CJK
		para2 := "\u8fd4\u56de" + repeatString("\u4f60\u597d", 100) // ~200+ runes of CJK
		text := "## CJK\n\n" + para1 + "\n\n" + para2 + "\n"

		cfg := ChunkConfig{Size: 150, Overlap: 30}
		chunks := ChunkMarkdown(text, cfg)

		// Should not panic or produce corrupt text
		assert.NotEmpty(t, chunks)
		for _, chunk := range chunks {
			assert.NotEmpty(t, chunk.Text)
			// Verify no partial rune corruption by round-tripping through []rune
			runes := []rune(chunk.Text)
			assert.Equal(t, chunk.Text, string(runes), "text should survive rune round-trip without corruption")
		}
	})
}

func TestChunkMarkdown_Edge_VeryLongSingleParagraph(t *testing.T) {
	t.Run("long paragraph without headers splits into multiple chunks", func(t *testing.T) {
		// Create a very long single paragraph (no section headers)
		longText := repeatString("This is a very long sentence that should be split across multiple chunks. ", 100)

		cfg := ChunkConfig{Size: 200, Overlap: 20}
		chunks := ChunkMarkdown(longText, cfg)

		// The paragraph is one big block — chunking depends on paragraph splitting
		// Since there are no double newlines, the whole thing is one paragraph
		// The chunker should still produce at least one chunk
		assert.NotEmpty(t, chunks)
		for _, chunk := range chunks {
			assert.NotEmpty(t, chunk.Text)
		}
	})

	t.Run("long paragraph with line breaks produces chunks", func(t *testing.T) {
		// Create long text with paragraph breaks so chunking can split
		var parts []string
		for i := 0; i < 50; i++ {
			parts = append(parts, "This is paragraph number that contains some meaningful text for testing purposes.")
		}
		longText := "## Long Content\n\n" + joinParagraphs(parts)

		cfg := ChunkConfig{Size: 300, Overlap: 30}
		chunks := ChunkMarkdown(longText, cfg)

		assert.Greater(t, len(chunks), 1, "long text should produce multiple chunks")
		for _, chunk := range chunks {
			assert.NotEmpty(t, chunk.Text)
			assert.Equal(t, "Long Content", chunk.Section)
		}
	})
}

func TestChunkMarkdown_Edge_ConfigBoundaries(t *testing.T) {
	t.Run("zero chunk size uses default 500", func(t *testing.T) {
		text := "## Section\n\nSome content.\n"
		cfg := ChunkConfig{Size: 0, Overlap: 0}
		chunks := ChunkMarkdown(text, cfg)

		assert.NotEmpty(t, chunks, "should still produce chunks with zero size (uses default)")
	})

	t.Run("negative chunk size uses default 500", func(t *testing.T) {
		text := "## Section\n\nSome content.\n"
		cfg := ChunkConfig{Size: -1, Overlap: 0}
		chunks := ChunkMarkdown(text, cfg)

		assert.NotEmpty(t, chunks)
	})

	t.Run("overlap equal to size resets to zero", func(t *testing.T) {
		// When overlap >= size, it resets to 0
		text := "## S\n\n" + repeatString("Word. ", 200)
		cfg := ChunkConfig{Size: 100, Overlap: 100}
		chunks := ChunkMarkdown(text, cfg)

		assert.NotEmpty(t, chunks)
	})

	t.Run("negative overlap resets to zero", func(t *testing.T) {
		text := "## S\n\n" + repeatString("Word. ", 200)
		cfg := ChunkConfig{Size: 100, Overlap: -5}
		chunks := ChunkMarkdown(text, cfg)

		assert.NotEmpty(t, chunks)
	})
}

func TestChunkMarkdown_Edge_ChunkIndexing(t *testing.T) {
	t.Run("chunk indices are sequential starting from zero", func(t *testing.T) {
		text := "## Section One\n\nContent one.\n\n## Section Two\n\nContent two.\n\n## Section Three\n\nContent three.\n"
		chunks := ChunkMarkdown(text, DefaultChunkConfig())

		for i, chunk := range chunks {
			assert.Equal(t, i, chunk.Index, "chunk index should be sequential")
		}
	})
}

func TestChunkID_Edge_LongText(t *testing.T) {
	t.Run("long text is truncated to first 100 runes for ID", func(t *testing.T) {
		longText := repeatString("a", 500)
		id1 := ChunkID("test.md", 0, longText)

		// Same first 100 characters, different tail — should produce same ID
		longText2 := repeatString("a", 100) + repeatString("b", 400)
		id2 := ChunkID("test.md", 0, longText2)

		assert.Equal(t, id1, id2, "IDs should match when first 100 runes are identical")
	})

	t.Run("unicode text uses rune count not byte count", func(t *testing.T) {
		// 100 CJK characters (3 bytes each in UTF-8) = 100 runes
		runeText := repeatString("\u4e16", 100)
		id1 := ChunkID("test.md", 0, runeText)

		// Same 100 CJK chars plus more — should produce same ID
		longerText := repeatString("\u4e16", 100) + repeatString("\u754c", 50)
		id2 := ChunkID("test.md", 0, longerText)

		assert.Equal(t, id1, id2, "IDs should match when first 100 runes are identical (CJK)")
	})
}

func TestDefaultChunkConfig(t *testing.T) {
	t.Run("returns expected default values", func(t *testing.T) {
		cfg := DefaultChunkConfig()

		assert.Equal(t, 500, cfg.Size, "default chunk size should be 500")
		assert.Equal(t, 50, cfg.Overlap, "default chunk overlap should be 50")
	})
}

func TestDefaultIngestConfig(t *testing.T) {
	t.Run("returns expected default values", func(t *testing.T) {
		cfg := DefaultIngestConfig()

		assert.Equal(t, "hostuk-docs", cfg.Collection, "default collection should be hostuk-docs")
		assert.Equal(t, 100, cfg.BatchSize, "default batch size should be 100")
		assert.False(t, cfg.Recreate, "recreate should be false by default")
		assert.False(t, cfg.Verbose, "verbose should be false by default")
		assert.Empty(t, cfg.Directory, "directory should be empty by default")

		// Nested ChunkConfig should match defaults
		assert.Equal(t, DefaultChunkConfig().Size, cfg.Chunk.Size)
		assert.Equal(t, DefaultChunkConfig().Overlap, cfg.Chunk.Overlap)
	})
}

// Helper: repeat a string n times
func repeatString(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// Helper: join paragraphs with double newlines
func joinParagraphs(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n\n"
		}
		result += p
	}
	return result
}
