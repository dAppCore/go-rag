package rag

import (
	"context"
	"fmt"
	"html"
	"iter"
	"slices"
	"strings"

	"dappco.re/go/core/log"
)

// QueryConfig holds query configuration.
type QueryConfig struct {
	Collection string
	Limit      uint64
	Threshold  float32 // Minimum similarity score (0-1)
	Category   string  // Filter by category
	Keywords   bool    // When true, extract keywords from query and boost matching results
}

// DefaultQueryConfig returns default query configuration.
func DefaultQueryConfig() QueryConfig {
	return QueryConfig{
		Collection: "hostuk-docs",
		Limit:      5,
		Threshold:  0.5,
	}
}

// QueryResult represents a query result with metadata.
type QueryResult struct {
	Text       string
	Source     string
	Section    string
	Category   string
	ChunkIndex int
	Score      float32
}

func (r QueryResult) GetText() string {
	return r.Text
}

func (r QueryResult) GetScore() float32 {
	return r.Score
}

func (r QueryResult) GetSource() string {
	return r.Source
}

func (r QueryResult) GetChunkIndex() int {
	return r.ChunkIndex
}

// Query searches for similar documents in the vector store.
func Query(ctx context.Context, store VectorStore, embedder Embedder, query string, cfg QueryConfig) ([]QueryResult, error) {
	it, err := QuerySeq(ctx, store, embedder, query, cfg)
	if err != nil {
		return nil, err
	}
	return slices.Collect(it), nil
}

// QuerySeq returns an iterator that yields query results from the vector store.
func QuerySeq(ctx context.Context, store VectorStore, embedder Embedder, query string, cfg QueryConfig) (iter.Seq[QueryResult], error) {
	// Generate embedding for query
	embedding, err := embedder.Embed(ctx, query)
	if err != nil {
		return nil, log.E("rag.Query", "error generating query embedding", err)
	}

	// Build filter
	var filter map[string]string
	if cfg.Category != "" {
		filter = map[string]string{"category": cfg.Category}
	}

	// Search vector store
	results, err := store.Search(ctx, cfg.Collection, embedding, cfg.Limit, filter)
	if err != nil {
		return nil, log.E("rag.Query", "error searching", err)
	}

	// Filter by threshold and convert to iterator
	filteredIt := func(yield func(QueryResult) bool) {
		var queryResults []QueryResult

		for _, r := range results {
			if r.Score < cfg.Threshold {
				continue
			}

			queryResults = append(queryResults, searchResultToQueryResult(r))
		}

		// Apply keyword boosting when enabled
		if cfg.Keywords && len(queryResults) > 0 {
			keywords := extractKeywords(query)
			if len(keywords) > 0 {
				queryResults = KeywordFilter(queryResults, keywords)
			}
		}

		if len(queryResults) > 0 && cfg.Limit > 0 {
			queryResults = Rank(queryResults, int(cfg.Limit))
		}

		for _, qr := range queryResults {
			if !yield(qr) {
				return
			}
		}
	}

	return filteredIt, nil
}

type rankedResult interface {
	GetText() string
	GetScore() float32
	GetSource() string
	GetChunkIndex() int
}

// Rank sorts results by score descending, removes duplicate documents, and
// truncates the slice to topK results.
func Rank[T rankedResult](results []T, topK int) []T {
	if len(results) == 0 {
		return nil
	}
	if topK <= 0 || topK > len(results) {
		topK = len(results)
	}

	sorted := make([]T, len(results))
	copy(sorted, results)
	slices.SortFunc(sorted, func(a, b T) int {
		if a.GetScore() > b.GetScore() {
			return -1
		}
		if a.GetScore() < b.GetScore() {
			return 1
		}
		if a.GetSource() < b.GetSource() {
			return -1
		}
		if a.GetSource() > b.GetSource() {
			return 1
		}
		if a.GetChunkIndex() < b.GetChunkIndex() {
			return -1
		}
		if a.GetChunkIndex() > b.GetChunkIndex() {
			return 1
		}
		return 0
	})

	seen := make(map[string]struct{}, len(sorted))
	ranked := make([]T, 0, topK)
	for _, result := range sorted {
		key := rankKey(result)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ranked = append(ranked, result)
		if len(ranked) == topK {
			break
		}
	}

	return ranked
}

func rankKey[T rankedResult](result T) string {
	if result.GetSource() != "" {
		return fmt.Sprintf("%s:%d", result.GetSource(), result.GetChunkIndex())
	}
	if text := result.GetText(); text != "" {
		return text
	}
	return fmt.Sprintf("score:%f", result.GetScore())
}

func searchResultToQueryResult(r SearchResult) QueryResult {
	qr := QueryResult{
		Text:       r.Text,
		Source:     r.Source,
		Section:    r.Section,
		Category:   r.Category,
		ChunkIndex: r.Index,
		Score:      r.Score,
	}

	// Fall back to the raw payload for compatibility with older stores and
	// test doubles that only populate payload fields.
	if qr.Text == "" {
		if text, ok := r.Payload["text"].(string); ok {
			qr.Text = text
		}
	}
	if qr.Source == "" {
		if source, ok := r.Payload["source"].(string); ok {
			qr.Source = source
		}
	}
	if qr.Section == "" {
		if section, ok := r.Payload["section"].(string); ok {
			qr.Section = section
		}
	}
	if qr.Category == "" {
		if category, ok := r.Payload["category"].(string); ok {
			qr.Category = category
		}
	}
	if qr.ChunkIndex == 0 {
		switch idx := r.Payload["chunk_index"].(type) {
		case int64:
			qr.ChunkIndex = int(idx)
		case float64:
			qr.ChunkIndex = int(idx)
		case int:
			qr.ChunkIndex = idx
		case float32:
			qr.ChunkIndex = int(idx)
		case string:
			fmt.Sscanf(idx, "%d", &qr.ChunkIndex)
		}
	}

	return qr
}

// FormatResultsText formats query results as plain text.
func FormatResultsText(results []QueryResult) string {
	if len(results) == 0 {
		return "No results found."
	}

	var sb strings.Builder
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("\n--- Result %d (score: %.2f) ---\n", i+1, r.Score))
		sb.WriteString(fmt.Sprintf("Source: %s\n", r.Source))
		if r.Section != "" {
			sb.WriteString(fmt.Sprintf("Section: %s\n", r.Section))
		}
		sb.WriteString(fmt.Sprintf("Category: %s\n\n", r.Category))
		sb.WriteString(r.Text)
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatResultsContext formats query results for LLM context injection.
func FormatResultsContext(results []QueryResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<retrieved_context>\n")
	for _, r := range results {
		// Escape XML special characters to prevent malformed output
		fmt.Fprintf(&sb, "<document source=\"%s\" section=\"%s\" category=\"%s\">\n",
			html.EscapeString(r.Source),
			html.EscapeString(r.Section),
			html.EscapeString(r.Category))
		sb.WriteString(html.EscapeString(r.Text))
		sb.WriteString("\n</document>\n\n")
	}
	sb.WriteString("</retrieved_context>")
	return sb.String()
}

// FormatResultsJSON formats query results as JSON-like output.
func FormatResultsJSON(results []QueryResult) string {
	if len(results) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, r := range results {
		sb.WriteString("  {\n")
		sb.WriteString(fmt.Sprintf("    \"source\": %q,\n", r.Source))
		sb.WriteString(fmt.Sprintf("    \"section\": %q,\n", r.Section))
		sb.WriteString(fmt.Sprintf("    \"category\": %q,\n", r.Category))
		sb.WriteString(fmt.Sprintf("    \"score\": %.4f,\n", r.Score))
		sb.WriteString(fmt.Sprintf("    \"text\": %q\n", r.Text))
		if i < len(results)-1 {
			sb.WriteString("  },\n")
		} else {
			sb.WriteString("  }\n")
		}
	}
	sb.WriteString("]")
	return sb.String()
}
