package rag

import (
	"crypto/md5"
	"fmt"
	"iter"
	"path/filepath"
	"slices"
	"strings"
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

// ChunkMarkdownSeq returns an iterator that yields document chunks from markdown text.
func ChunkMarkdownSeq(text string, cfg ChunkConfig) iter.Seq[Chunk] {
	if cfg.Size <= 0 {
		cfg.Size = 500
	}
	if cfg.Overlap < 0 || cfg.Overlap >= cfg.Size {
		cfg.Overlap = 0
	}

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
			if len(section) <= cfg.Size {
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
			currentChunk := ""
			for para := range splitByParagraphsSeq(section) {
				para = strings.TrimSpace(para)
				if para == "" {
					continue
				}

				// If the paragraph itself exceeds Size, split at sentence
				// boundaries and treat each sentence (or group of sentences)
				// as a separate sub-paragraph.
				for sp := range yieldSubParas(para, cfg.Size) {
					sp = strings.TrimSpace(sp)
					if sp == "" {
						continue
					}

					if len(currentChunk)+len(sp)+2 <= cfg.Size {
						if currentChunk != "" {
							currentChunk += "\n\n" + sp
						} else {
							currentChunk = sp
						}
					} else {
						if currentChunk != "" {
							if !yield(Chunk{
								Text:    strings.TrimSpace(currentChunk),
								Section: title,
								Index:   chunkIndex,
							}) {
								return
							}
							chunkIndex++
						}
						// Start new chunk with overlap from previous,
						// aligned to the nearest word boundary.
						currentChunk = overlapPrefix(currentChunk, cfg.Overlap, sp)
					}
				}
			}

			// Don't forget the last chunk of the section
			if strings.TrimSpace(currentChunk) != "" {
				if !yield(Chunk{
					Text:    strings.TrimSpace(currentChunk),
					Section: title,
					Index:   chunkIndex,
				}) {
					return
				}
				chunkIndex++
			}
		}
	}
}

func yieldSubParas(para string, size int) iter.Seq[string] {
	return func(yield func(string) bool) {
		if len(para) <= size {
			yield(para)
			return
		}
		for s := range splitBySentencesSeq(para) {
			if !yield(s) {
				return
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

// splitBySentences splits text at sentence boundaries (". ", "? ", "! ").
// Returns the original text in a single-element slice when no boundaries are found.
func splitBySentences(text string) []string {
	return slices.Collect(splitBySentencesSeq(text))
}

// splitBySentencesSeq returns an iterator that yields sentences split at
// boundaries (". ", "? ", "! ").
func splitBySentencesSeq(text string) iter.Seq[string] {
	return func(yield func(string) bool) {
		remaining := text

		for len(remaining) > 0 {
			// Find the earliest sentence boundary
			bestIdx := -1
			var bestSep string
			for _, sep := range []string{". ", "? ", "! "} {
				idx := strings.Index(remaining, sep)
				if idx >= 0 && (bestIdx < 0 || idx < bestIdx) {
					bestIdx = idx
					bestSep = sep
				}
			}

			if bestIdx < 0 {
				// No more boundaries — yield remainder if not empty
				if s := strings.TrimSpace(remaining); s != "" {
					if !yield(s) {
						return
					}
				}
				break
			}

			// Include the punctuation mark in the sentence, but not the trailing space
			sentence := remaining[:bestIdx+len(bestSep)-1]
			if s := strings.TrimSpace(sentence); s != "" {
				if !yield(s) {
					return
				}
			}
			remaining = remaining[bestIdx+len(bestSep):]
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
