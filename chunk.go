package rag

import (
	"crypto/md5"
	"fmt"
	"iter"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

// ChunkConfig holds chunking configuration.
type ChunkConfig struct {
	Size    int // Characters per chunk
	Overlap int // Overlap between chunks
}

// DefaultChunkConfig returns default chunking configuration.
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		Size:    500,
		Overlap: 50,
	}
}

// Chunk represents a text chunk with metadata.
type Chunk struct {
	Text    string
	Section string
	Index   int
}

// ChunkMarkdown splits markdown text into chunks by sections and paragraphs.
// Preserves context with configurable overlap. When a paragraph exceeds the
// configured Size, it is split at sentence boundaries. Overlap is aligned to
// word boundaries to avoid splitting mid-word.
func ChunkMarkdown(text string, cfg ChunkConfig) []Chunk {
	return slices.Collect(ChunkMarkdownSeq(text, cfg))
}

func normalizeChunkConfig(cfg ChunkConfig) ChunkConfig {
	if cfg.Size <= 0 {
		cfg.Size = 500
	}
	if cfg.Overlap < 0 || cfg.Overlap >= cfg.Size {
		cfg.Overlap = 0
	}
	return cfg
}

// ChunkBySentences splits text into chunks at sentence boundaries.
func ChunkBySentences(text string, cfg ChunkConfig) []Chunk {
	return slices.Collect(ChunkBySentencesSeq(text, cfg))
}

// ChunkBySentencesSeq returns an iterator that yields sentence-based chunks.
func ChunkBySentencesSeq(text string, cfg ChunkConfig) iter.Seq[Chunk] {
	cfg = normalizeChunkConfig(cfg)
	return func(yield func(Chunk) bool) {
		chunkIndex := 0
		if !emitSegmentsAsChunks(splitBySentencesSeq(text), "", &chunkIndex, yield) {
			return
		}
	}
}

// ChunkByParagraphs splits text into chunks at paragraph boundaries.
func ChunkByParagraphs(text string, cfg ChunkConfig) []Chunk {
	return slices.Collect(ChunkByParagraphsSeq(text, cfg))
}

// ChunkByParagraphsSeq returns an iterator that yields paragraph-based chunks.
func ChunkByParagraphsSeq(text string, cfg ChunkConfig) iter.Seq[Chunk] {
	cfg = normalizeChunkConfig(cfg)
	return func(yield func(Chunk) bool) {
		chunkIndex := 0
		if !emitSegmentsAsChunks(splitParagraphSegmentsSeq(text, cfg.Size), "", &chunkIndex, yield) {
			return
		}
	}
}

// ChunkMarkdownSeq returns an iterator that yields document chunks from markdown text.
func ChunkMarkdownSeq(text string, cfg ChunkConfig) iter.Seq[Chunk] {
	cfg = normalizeChunkConfig(cfg)

	return func(yield func(Chunk) bool) {
		chunkIndex := 0

		// Split by ## headers
		for section := range splitBySectionsSeq(text) {
			section = strings.TrimSpace(section)
			if section == "" {
				continue
			}

			// Extract section title
			lines := strings.SplitN(section, "\n", 2)
			title := ""
			if strings.HasPrefix(lines[0], "#") {
				title = strings.TrimLeft(lines[0], "#")
				title = strings.TrimSpace(title)
			}

			// If section is small enough, yield as-is
			if runeLen(section) <= cfg.Size {
				if !yield(Chunk{
					Text:    section,
					Section: title,
					Index:   chunkIndex,
				}) {
					return
				}
				chunkIndex++
				continue
			}

			// Otherwise, chunk by paragraphs
			segments := splitParagraphSegmentsSeq(section, cfg.Size)
			if !emitChunksFromSegments(segments, cfg, title, &chunkIndex, yield) {
				return
			}
		}
	}
}

// emitChunksFromSegments accumulates segments into chunks and yields them.
func emitChunksFromSegments(segments iter.Seq[string], cfg ChunkConfig, section string, chunkIndex *int, yield func(Chunk) bool) bool {
	currentChunk := ""

	flush := func() bool {
		if strings.TrimSpace(currentChunk) == "" {
			return true
		}
		if !yield(Chunk{
			Text:    strings.TrimSpace(currentChunk),
			Section: section,
			Index:   *chunkIndex,
		}) {
			return false
		}
		*chunkIndex = *chunkIndex + 1
		return true
	}

	for seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		if currentChunk == "" {
			currentChunk = seg
			continue
		}

		if runeLen(currentChunk)+runeLen(seg)+2 <= cfg.Size {
			currentChunk += "\n\n" + seg
			continue
		}

		if !flush() {
			return false
		}

		// Start new chunk with overlap from previous chunk, aligned to the
		// nearest word boundary.
		currentChunk = overlapPrefix(currentChunk, cfg.Overlap, seg)
	}

	return flush()
}

// emitSegmentsAsChunks yields each segment as its own chunk.
func emitSegmentsAsChunks(segments iter.Seq[string], section string, chunkIndex *int, yield func(Chunk) bool) bool {
	for seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		if !yield(Chunk{
			Text:    seg,
			Section: section,
			Index:   *chunkIndex,
		}) {
			return false
		}
		*chunkIndex = *chunkIndex + 1
	}
	return true
}

// splitParagraphSegmentsSeq yields paragraph-sized segments, falling back to
// sentence-sized segments for oversized paragraphs.
func splitParagraphSegmentsSeq(text string, size int) iter.Seq[string] {
	return func(yield func(string) bool) {
		for para := range splitByParagraphsSeq(text) {
			para = strings.TrimSpace(para)
			if para == "" {
				continue
			}
			if runeLen(para) <= size {
				if !yield(para) {
					return
				}
				continue
			}
			for s := range splitBySentencesSeq(para) {
				if !yield(s) {
					return
				}
			}
		}
	}
}

// overlapPrefix builds the start of a new chunk by taking word-boundary-aligned
// overlap text from the previous chunk and prepending it to the new paragraph.
func overlapPrefix(prevChunk string, overlap int, newPara string) string {
	if overlap <= 0 {
		return newPara
	}

	runes := []rune(prevChunk)
	if len(runes) <= overlap {
		return newPara
	}

	// Slice from the end of the previous chunk
	overlapRunes := runes[len(runes)-overlap:]

	// Align to the nearest word boundary: find the first space within the
	// overlap slice and start after it to avoid a partial leading word.
	overlapText := string(overlapRunes)
	if idx := strings.IndexByte(overlapText, ' '); idx >= 0 {
		overlapText = overlapText[idx+1:]
	}

	if overlapText == "" {
		return newPara
	}

	return overlapText + "\n\n" + newPara
}

// splitBySentences splits text at sentence boundaries and returns a slice.
func splitBySentences(text string) []string {
	return slices.Collect(splitBySentencesSeq(text))
}

// splitBySentencesSeq returns an iterator that yields sentences split at
// sentence boundaries. Boundaries include punctuation marks and newlines.
func splitBySentencesSeq(text string) iter.Seq[string] {
	return func(yield func(string) bool) {
		text = strings.ReplaceAll(text, "\r\n", "\n")

		start := 0
		for i := 0; i < len(text); {
			r, size := utf8.DecodeRuneInString(text[i:])

			if r == '\n' {
				if s := strings.TrimSpace(text[start:i]); s != "" {
					if !yield(s) {
						return
					}
				}
				start = i + size
				i = start
				continue
			}

			if r == '.' || r == '!' || r == '?' {
				end := i + size
				j := end
				for j < len(text) {
					next, nextSize := utf8.DecodeRuneInString(text[j:])
					if next == ' ' || next == '\t' || next == '\n' || next == '\r' {
						j += nextSize
						continue
					}
					break
				}
				if j == len(text) || j > end {
					if s := strings.TrimSpace(text[start:end]); s != "" {
						if !yield(s) {
							return
						}
					}
					start = j
					i = j
					continue
				}
			}

			i += size
		}

		if s := strings.TrimSpace(text[start:]); s != "" {
			if !yield(s) {
				return
			}
		}
	}
}

// splitBySections splits text by ## headers while preserving the header with its content.
func splitBySections(text string) []string {
	return slices.Collect(splitBySectionsSeq(text))
}

// splitBySectionsSeq returns an iterator that yields text sections split by ## headers.
func splitBySectionsSeq(text string) iter.Seq[string] {
	return func(yield func(string) bool) {
		var currentSection strings.Builder
		for line := range strings.SplitSeq(text, "\n") {
			// Check if this line is a ## header
			if strings.HasPrefix(line, "## ") {
				// Yield previous section if exists
				if currentSection.Len() > 0 {
					if !yield(currentSection.String()) {
						return
					}
					currentSection.Reset()
				}
			}
			currentSection.WriteString(line)
			currentSection.WriteString("\n")
		}

		// Don't forget the last section
		if currentSection.Len() > 0 {
			yield(currentSection.String())
		}
	}
}

// splitByParagraphs splits text by double newlines.
func splitByParagraphs(text string) []string {
	return slices.Collect(splitByParagraphsSeq(text))
}

// splitByParagraphsSeq returns an iterator that yields paragraphs split by double newlines.
func splitByParagraphsSeq(text string) iter.Seq[string] {
	return func(yield func(string) bool) {
		// Replace multiple newlines with a marker, then split
		normalized := text
		for strings.Contains(normalized, "\n\n\n") {
			normalized = strings.ReplaceAll(normalized, "\n\n\n", "\n\n")
		}
		for s := range strings.SplitSeq(normalized, "\n\n") {
			if !yield(s) {
				return
			}
		}
	}
}

func runeLen(text string) int {
	return utf8.RuneCountInString(text)
}

// Category determines the document category from file path.
func Category(path string) string {
	lower := strings.ToLower(path)

	switch {
	case strings.Contains(lower, "flux") || strings.Contains(lower, "ui/component"):
		return "ui-component"
	case strings.Contains(lower, "brand") || strings.Contains(lower, "mascot"):
		return "brand"
	case strings.Contains(lower, "brief"):
		return "product-brief"
	case strings.Contains(lower, "help") || strings.Contains(lower, "draft"):
		return "help-doc"
	case strings.Contains(lower, "task") || strings.Contains(lower, "plan"):
		return "task"
	case strings.Contains(lower, "architecture") || strings.Contains(lower, "migration"):
		return "architecture"
	default:
		return "documentation"
	}
}

// ChunkID generates a unique ID for a chunk.
func ChunkID(path string, index int, text string) string {
	// Use first 100 runes of text for uniqueness (rune-safe for UTF-8)
	runes := []rune(text)
	if len(runes) > 100 {
		runes = runes[:100]
	}
	textPart := string(runes)
	data := fmt.Sprintf("%s:%d:%s", path, index, textPart)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// FileExtensions returns the file extensions to process.
func FileExtensions() []string {
	return []string{".md", ".markdown", ".txt"}
}

// ShouldProcess checks if a file should be processed based on extension.
func ShouldProcess(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return slices.Contains(FileExtensions(), ext)
}
