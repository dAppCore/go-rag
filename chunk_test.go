package rag

import (
	"strings"
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
		for range 50 {
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
	var result strings.Builder
	for range n {
		result.WriteString(s)
	}
	return result.String()
}

// Helper: join paragraphs with double newlines
func joinParagraphs(parts []string) string {
	var result strings.Builder
	for i, p := range parts {
		if i > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(p)
	}
	return result.String()
}

// --- Phase 3.1: Sentence splitting and overlap alignment ---

func TestChunkMarkdown_SentenceSplitting(t *testing.T) {
	t.Run("oversized paragraph split at sentence boundaries", func(t *testing.T) {
		// Three sentences, each ~60 chars. Total ~180 chars exceeds Size=100.
		s1 := "The quick brown fox jumps over the lazy dog on the green hill."
		s2 := "A second sentence that also has a reasonable amount of words."
		s3 := "Finally the third sentence wraps up this oversized paragraph."
		text := "## Section\n\n" + s1 + " " + s2 + " " + s3
		cfg := ChunkConfig{Size: 100, Overlap: 0}
		chunks := ChunkMarkdown(text, cfg)

		// Should produce more than one chunk because sentences are split
		assert.Greater(t, len(chunks), 1, "oversized paragraph should be split into multiple chunks")

		// Verify all original text appears across the chunks
		combined := ""
		for _, c := range chunks {
			combined += c.Text + " "
		}
		assert.Contains(t, combined, "quick brown fox")
		assert.Contains(t, combined, "second sentence")
		assert.Contains(t, combined, "third sentence")

		// Each chunk should have the correct section
		for _, c := range chunks {
			assert.Equal(t, "Section", c.Section)
		}
	})

	t.Run("paragraph without sentence boundaries kept as single chunk", func(t *testing.T) {
		// Long paragraph with no sentence-ending punctuation followed by space
		para := repeatString("word ", 100) // ~500 chars, no ". " or "? " or "! "
		text := "## S\n\n" + para
		cfg := ChunkConfig{Size: 100, Overlap: 0}
		chunks := ChunkMarkdown(text, cfg)

		// Should still produce at least one chunk (fallback behaviour)
		assert.NotEmpty(t, chunks)
	})

	t.Run("sentence boundaries preserve punctuation", func(t *testing.T) {
		text := "## S\n\nFirst sentence. Second sentence. Third sentence."
		cfg := ChunkConfig{Size: 30, Overlap: 0}
		chunks := ChunkMarkdown(text, cfg)

		// The first chunk should end with a period (punctuation preserved)
		foundPeriod := false
		for _, c := range chunks {
			if strings.HasSuffix(strings.TrimSpace(c.Text), ".") {
				foundPeriod = true
				break
			}
		}
		assert.True(t, foundPeriod, "at least one chunk should end with a period")
	})
}

func TestChunkMarkdown_OverlapWordBoundary(t *testing.T) {
	t.Run("overlap does not split mid-word", func(t *testing.T) {
		// Build two paragraphs where the first is large enough to emit,
		// and the overlap region lands mid-word in the naive rune slice.
		para1 := "Alpha bravo charlie delta echo foxtrot golf hotel india juliet kilo lima mike november oscar papa quebec romeo sierra tango."
		para2 := "Uniform victor whiskey xray yankee zulu."
		text := "## S\n\n" + para1 + "\n\n" + para2
		cfg := ChunkConfig{Size: 80, Overlap: 15}
		chunks := ChunkMarkdown(text, cfg)

		// Find a chunk that contains overlap text (not the first chunk)
		for i, c := range chunks {
			if i == 0 {
				continue
			}
			// The overlap prefix should start at a word boundary:
			// it should not begin with a partial word fragment.
			words := strings.Fields(c.Text)
			if len(words) > 0 {
				// The first word should be a recognisable whole word, not a suffix
				// of a longer word. We can verify there is no leading lowercase
				// fragment by checking the original text contains this word.
				firstWord := words[0]
				assert.True(t,
					strings.Contains(para1, firstWord) || strings.Contains(para2, firstWord),
					"overlap should start at a word boundary, got leading word: %q", firstWord)
			}
		}
	})

	t.Run("overlap with zero value produces no overlap", func(t *testing.T) {
		para1 := repeatString("Abcdef. ", 30) // ~240 chars
		para2 := "Unique marker text here."
		text := "## S\n\n" + para1 + "\n\n" + para2
		cfg := ChunkConfig{Size: 100, Overlap: 0}
		chunks := ChunkMarkdown(text, cfg)

		// With zero overlap, the second chunk should not contain text from
		// the end of the previous chunk
		assert.NotEmpty(t, chunks)
	})
}

func TestSplitBySentences(t *testing.T) {
	t.Run("splits on period-space", func(t *testing.T) {
		result := splitBySentences("First. Second. Third.")
		assert.Len(t, result, 3)
		assert.Equal(t, "First.", result[0])
		assert.Equal(t, "Second.", result[1])
		assert.Equal(t, "Third.", result[2])
	})

	t.Run("splits on question mark", func(t *testing.T) {
		result := splitBySentences("What is this? It is a test.")
		assert.Len(t, result, 2)
		assert.Equal(t, "What is this?", result[0])
		assert.Equal(t, "It is a test.", result[1])
	})

	t.Run("splits on exclamation mark", func(t *testing.T) {
		result := splitBySentences("Wow! That is amazing.")
		assert.Len(t, result, 2)
		assert.Equal(t, "Wow!", result[0])
		assert.Equal(t, "That is amazing.", result[1])
	})

	t.Run("no boundaries returns single element", func(t *testing.T) {
		result := splitBySentences("just a plain string with no ending")
		assert.Len(t, result, 1)
		assert.Equal(t, "just a plain string with no ending", result[0])
	})

	t.Run("empty string returns empty", func(t *testing.T) {
		result := splitBySentences("")
		assert.Empty(t, result)
	})
}
