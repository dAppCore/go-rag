package rag

import (
	"crypto/md5"
	"iter"
	"slices"
	"strings"
	"unicode"

	"dappco.re/go/core"
)

// ChunkConfig holds chunking configuration.
// cfg := ChunkConfig{Size: 500, Overlap: 50}
type ChunkConfig struct {
	Size    int // Characters per chunk
	Overlap int // Overlap between chunks
}

// DefaultChunkConfig returns default chunking configuration.
// cfg := DefaultChunkConfig()
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		Size:    500,
		Overlap: 50,
	}
}

// Chunk represents a text chunk with metadata.
// chunk := Chunk{Text: "Go uses goroutines.", Section: "Concurrency", Index: 0}
type Chunk struct {
	Text    string
	Section string
	Index   int
}

// ChunkMarkdown splits markdown text into chunks by sections and paragraphs.
// Preserves context with configurable overlap. When a paragraph exceeds the
// configured Size, it is split at sentence boundaries. Overlap is aligned to
// word boundaries to avoid splitting mid-word.
// chunks := ChunkMarkdown(markdown, DefaultChunkConfig())
func ChunkMarkdown(text string, cfg ChunkConfig) []Chunk {
	return slices.Collect(ChunkMarkdownSeq(text, cfg))
}

// ChunkMarkdownSeq returns an iterator that yields document chunks from markdown text.
// for chunk := range ChunkMarkdownSeq(markdown, DefaultChunkConfig()) { _ = chunk }
func ChunkMarkdownSeq(text string, cfg ChunkConfig) iter.Seq[Chunk] {
	if cfg.Size <= 0 {
		cfg.Size = 500
	}
	if cfg.Overlap < 0 || cfg.Overlap >= cfg.Size {
		cfg.Overlap = 0
	}
	text = normalizeLineEndings(text)

	return func(yield func(Chunk) bool) {
		chunkIndex := 0
		currentParentSection := ""

		// Split by ## headers
		for section := range splitBySectionsSeq(text) {
			section = core.Trim(section)
			if section == "" {
				continue
			}

			// Extract section title and preserve parent heading context.
			parentTitle, title := markdownSectionTitle(section)
			if parentTitle != "" {
				currentParentSection = parentTitle
			}
			if title == "" {
				title = currentParentSection
			} else if currentParentSection != "" && title != currentParentSection && parentTitle == "" {
				title = currentParentSection + " / " + title
			} else if currentParentSection != "" && title != currentParentSection && parentTitle != "" {
				title = currentParentSection + " / " + title
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
			currentChunk := ""
			for para := range splitByParagraphsSeq(section) {
				para = core.Trim(para)
				if para == "" {
					continue
				}

				// If the paragraph itself exceeds Size, split at sentence
				// boundaries and treat each sentence (or group of sentences)
				// as a separate sub-paragraph.
				for sp := range yieldSubParas(para, cfg.Size) {
					sp = core.Trim(sp)
					if sp == "" {
						continue
					}

					if runeLen(currentChunk)+runeLen(sp)+2 <= cfg.Size {
						if currentChunk != "" {
							currentChunk += "\n\n" + sp
						} else {
							currentChunk = sp
						}
					} else {
						if currentChunk != "" {
							if !yield(Chunk{
								Text:    core.Trim(currentChunk),
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
			if core.Trim(currentChunk) != "" {
				if !yield(Chunk{
					Text:    core.Trim(currentChunk),
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
		if runeLen(para) <= size {
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

	start := len(runes) - overlap
	if start < 0 {
		start = 0
	}
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}

	overlapText := strings.TrimLeftFunc(string(runes[start:]), unicode.IsSpace)
	overlapText = trimLeadingNonWordRunes(overlapText)

	if overlapText == "" {
		return newPara
	}

	return overlapText + "\n\n" + newPara
}

// ChunkBySentences splits text into chunks aligned to sentence boundaries.
// Useful when strict size control matters more than section context: each
// emitted chunk stays below cfg.Size, and overlap is aligned to word
// boundaries to avoid splitting mid-word.
//
//	chunks := ChunkBySentences(text, DefaultChunkConfig())
func ChunkBySentences(text string, cfg ChunkConfig) []Chunk {
	return slices.Collect(ChunkBySentencesSeq(text, cfg))
}

// ChunkBySentencesSeq returns an iterator that yields sentence-aligned chunks.
// for chunk := range ChunkBySentencesSeq(text, DefaultChunkConfig()) { _ = chunk }
func ChunkBySentencesSeq(text string, cfg ChunkConfig) iter.Seq[Chunk] {
	if cfg.Size <= 0 {
		cfg.Size = 500
	}
	if cfg.Overlap < 0 || cfg.Overlap >= cfg.Size {
		cfg.Overlap = 0
	}
	text = normalizeLineEndings(text)

	return func(yield func(Chunk) bool) {
		chunkIndex := 0
		currentParentSection := ""

		for section := range splitBySectionsSeq(text) {
			section = core.Trim(section)
			if section == "" {
				continue
			}

			parentTitle, title := markdownSectionTitle(section)
			if parentTitle != "" {
				currentParentSection = parentTitle
			}
			if title == "" {
				title = currentParentSection
			} else if currentParentSection != "" && title != currentParentSection {
				title = currentParentSection + " / " + title
			}

			trimmed := core.Trim(section)
			currentChunk := ""

			for sentence := range splitBySentencesSeq(trimmed) {
				sentence = core.Trim(sentence)
				if sentence == "" {
					continue
				}

				// Accumulate while within size budget.
				sep := " "
				if currentChunk == "" {
					sep = ""
				}
				if runeLen(currentChunk)+runeLen(sep)+runeLen(sentence) <= cfg.Size {
					currentChunk = currentChunk + sep + sentence
					continue
				}

				// Emit current chunk if non-empty.
				if currentChunk != "" {
					if !yield(Chunk{
						Text:    core.Trim(currentChunk),
						Section: title,
						Index:   chunkIndex,
					}) {
						return
					}
					chunkIndex++
				}

				// Start a new chunk with overlap carried from the previous chunk.
				currentChunk = overlapPrefixInline(currentChunk, cfg.Overlap, sentence)
			}

			if core.Trim(currentChunk) != "" {
				if !yield(Chunk{
					Text:    core.Trim(currentChunk),
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

// ChunkByParagraphs splits text into chunks aligned to paragraph boundaries
// (blank lines). Paragraphs that individually exceed cfg.Size are split at
// sentence boundaries, then re-accumulated.
//
//	chunks := ChunkByParagraphs(text, DefaultChunkConfig())
func ChunkByParagraphs(text string, cfg ChunkConfig) []Chunk {
	return slices.Collect(ChunkByParagraphsSeq(text, cfg))
}

// ChunkByParagraphsSeq returns an iterator that yields paragraph-aligned chunks.
// for chunk := range ChunkByParagraphsSeq(text, DefaultChunkConfig()) { _ = chunk }
func ChunkByParagraphsSeq(text string, cfg ChunkConfig) iter.Seq[Chunk] {
	if cfg.Size <= 0 {
		cfg.Size = 500
	}
	if cfg.Overlap < 0 || cfg.Overlap >= cfg.Size {
		cfg.Overlap = 0
	}
	text = normalizeLineEndings(text)

	return func(yield func(Chunk) bool) {
		chunkIndex := 0
		currentParentSection := ""

		for section := range splitBySectionsSeq(text) {
			section = core.Trim(section)
			if section == "" {
				continue
			}

			parentTitle, title := markdownSectionTitle(section)
			if parentTitle != "" {
				currentParentSection = parentTitle
			}
			if title == "" {
				title = currentParentSection
			} else if currentParentSection != "" && title != currentParentSection {
				title = currentParentSection + " / " + title
			}

			trimmed := core.Trim(section)
			currentChunk := ""

			for para := range splitByParagraphsSeq(trimmed) {
				para = core.Trim(para)
				if para == "" {
					continue
				}

				for sp := range yieldSubParas(para, cfg.Size) {
					sp = core.Trim(sp)
					if sp == "" {
						continue
					}

					if runeLen(currentChunk)+runeLen(sp)+2 <= cfg.Size {
						if currentChunk != "" {
							currentChunk += "\n\n" + sp
						} else {
							currentChunk = sp
						}
						continue
					}

					if currentChunk != "" {
						if !yield(Chunk{
							Text:    core.Trim(currentChunk),
							Section: title,
							Index:   chunkIndex,
						}) {
							return
						}
						chunkIndex++
					}
					currentChunk = overlapPrefix(currentChunk, cfg.Overlap, sp)
				}
			}

			if core.Trim(currentChunk) != "" {
				if !yield(Chunk{
					Text:    core.Trim(currentChunk),
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

// overlapPrefixInline is the single-line variant of overlapPrefix used by
// sentence chunking, where a space separator is more natural than "\n\n".
func overlapPrefixInline(prevChunk string, overlap int, newSentence string) string {
	if overlap <= 0 {
		return newSentence
	}
	runes := []rune(prevChunk)
	if len(runes) <= overlap {
		return newSentence
	}
	start := len(runes) - overlap
	if start < 0 {
		start = 0
	}
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}
	overlapText := strings.TrimLeftFunc(string(runes[start:]), unicode.IsSpace)
	overlapText = trimLeadingNonWordRunes(overlapText)
	if overlapText == "" {
		return newSentence
	}
	return overlapText + " " + newSentence
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
		text = normalizeLineEndings(text)
		remaining := text

		for len(remaining) > 0 {
			boundary := -1
			for i, r := range remaining {
				switch r {
				case '.', '!', '?', '\n':
					boundary = i + len(string(r))
					break
				}
				if boundary >= 0 {
					break
				}
			}

			if boundary < 0 {
				if s := core.Trim(remaining); s != "" {
					if !yield(s) {
						return
					}
				}
				break
			}

			sentence := core.Trim(remaining[:boundary])
			if sentence != "" {
				if !yield(sentence) {
					return
				}
			}
			remaining = strings.TrimLeftFunc(remaining[boundary:], unicode.IsSpace)
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
		currentSection := core.NewBuilder()
		for _, line := range core.Split(text, "\n") {
			// Check if this line is a ## header
			if core.HasPrefix(line, "## ") {
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
		text = normalizeLineEndings(text)
		// Replace multiple newlines with a marker, then split
		normalized := text
		for core.Contains(normalized, "\n\n\n") {
			normalized = core.Replace(normalized, "\n\n\n", "\n\n")
		}
		for _, s := range core.Split(normalized, "\n\n") {
			if !yield(s) {
				return
			}
		}
	}
}

// Category determines the document category from file path.
// category := Category("docs/architecture/guide.md")
func Category(path string) string {
	lower := core.Lower(path)

	switch {
	case core.Contains(lower, "flux") || core.Contains(lower, "ui/component"):
		return "ui-component"
	case core.Contains(lower, "brand") || core.Contains(lower, "mascot"):
		return "brand"
	case core.Contains(lower, "brief"):
		return "product-brief"
	case core.Contains(lower, "help") || core.Contains(lower, "draft"):
		return "help-doc"
	case core.Contains(lower, "task") || core.Contains(lower, "plan"):
		return "task"
	case core.Contains(lower, "architecture") || core.Contains(lower, "migration"):
		return "architecture"
	default:
		return "documentation"
	}
}

// ChunkID generates a unique ID for a chunk.
// id := ChunkID("docs/guide.md", 0, "Go uses goroutines.")
func ChunkID(path string, index int, text string) string {
	// Use first 100 runes of text for uniqueness (rune-safe for UTF-8)
	runes := []rune(text)
	if len(runes) > 100 {
		runes = runes[:100]
	}
	textPart := string(runes)
	data := core.Sprintf("%s:%d:%s", path, index, textPart)
	hash := md5.Sum([]byte(data))
	return core.Sprintf("%x", hash)
}

// FileExtensions returns the file extensions to process.
// exts := FileExtensions()
func FileExtensions() []string {
	return []string{".md", ".markdown", ".pdf", ".txt"}
}

// ShouldProcess checks if a file should be processed based on extension.
// ok := ShouldProcess("docs/guide.md")
func ShouldProcess(path string) bool {
	ext := core.Lower(core.PathExt(path))
	return slices.Contains(FileExtensions(), ext)
}

func normalizeLineEndings(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

func trimHeadingPrefix(line string) string {
	for core.HasPrefix(line, "#") {
		line = core.TrimPrefix(line, "#")
	}
	return core.Trim(line)
}

func markdownSectionTitle(section string) (string, string) {
	var parentTitle string
	var title string

	for _, line := range core.Split(section, "\n") {
		trimmed := core.Trim(line)
		if trimmed == "" || !core.HasPrefix(trimmed, "#") {
			continue
		}

		level := 0
		for level < len(trimmed) && trimmed[level] == '#' {
			level++
		}
		if level == 0 || level > 6 {
			continue
		}
		if level < len(trimmed) && trimmed[level] != ' ' {
			continue
		}

		heading := trimHeadingPrefix(trimmed)
		if heading == "" {
			continue
		}

		switch level {
		case 1:
			parentTitle = heading
			if title == "" {
				title = heading
			}
		case 2:
			title = heading
			return parentTitle, title
		default:
			if title == "" {
				title = heading
			}
		}
	}

	return parentTitle, title
}

func indexOf(s, substr string) int {
	if substr == "" || len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func runeLen(text string) int {
	return len([]rune(text))
}

func trimLeadingNonWordRunes(text string) string {
	for i, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return text[i:]
		}
	}
	return ""
}
