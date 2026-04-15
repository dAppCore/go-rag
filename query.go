package rag

import (
	"context"
	"html"
	"iter"
	"math"
	"slices"

	"dappco.re/go/core"
)

// QueryConfig holds query configuration.
// cfg := QueryConfig{Collection: "project-docs", Limit: 5, Threshold: 0.6}
type QueryConfig struct {
	Collection string
	Limit      uint64
	Threshold  float32 // Minimum similarity score (0-1)
	Category   string  // Filter by category
	Keywords   bool    // When true, extract keywords from query and boost matching results
}

// DefaultQueryConfig returns default query configuration.
// cfg := DefaultQueryConfig()
func DefaultQueryConfig() QueryConfig {
	return QueryConfig{
		Collection: "hostuk-docs",
		Limit:      5,
		Threshold:  0.5,
	}
}

// QueryResult represents a query result with metadata.
// result := QueryResult{Source: "docs/go.md", Section: "Concurrency", Score: 0.92}
type QueryResult struct {
	Text       string
	Source     string
	Section    string
	Category   string
	ChunkIndex int
	Score      float32
}

// GetText returns the result text (satisfies the rankedResult interface).
func (r QueryResult) GetText() string { return r.Text }

// GetScore returns the result similarity score (satisfies the rankedResult interface).
func (r QueryResult) GetScore() float32 { return r.Score }

// GetSource returns the result source path (satisfies the rankedResult interface).
func (r QueryResult) GetSource() string { return r.Source }

// GetChunkIndex returns the source chunk index (satisfies the rankedResult interface).
func (r QueryResult) GetChunkIndex() int {
	return r.ChunkIndex
}

// rankedResult is implemented by any result type that carries enough
// identity and score data to participate in Rank / deduplication.
//
//	var _ rankedResult = QueryResult{}
type rankedResult interface {
	GetText() string
	GetScore() float32
	GetSource() string
	GetChunkIndex() int
}

// Rank sorts results by score (descending), removes duplicate documents by
// (source, chunk-index) or text identity, and returns the first topK.
//
//	top := Rank(results, 5)
func Rank[T rankedResult](results []T, topK int) []T {
	if len(results) == 0 {
		return nil
	}
	if topK <= 0 {
		return nil
	}
	if topK > len(results) {
		topK = len(results)
	}

	sorted := make([]T, len(results))
	copy(sorted, results)
	slices.SortFunc(sorted, func(a, b T) int {
		switch {
		case a.GetScore() > b.GetScore():
			return -1
		case a.GetScore() < b.GetScore():
			return 1
		case a.GetSource() < b.GetSource():
			return -1
		case a.GetSource() > b.GetSource():
			return 1
		case a.GetChunkIndex() < b.GetChunkIndex():
			return -1
		case a.GetChunkIndex() > b.GetChunkIndex():
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
		if len(ranked) >= topK {
			break
		}
	}
	return ranked
}

// rankKey produces a deduplication key for a ranked result: prefer
// (source, chunkIndex) when source is known, fall back to text, then score.
func rankKey[T rankedResult](result T) string {
	if result.GetSource() != "" {
		return core.Sprintf("%s:%d", result.GetSource(), result.GetChunkIndex())
	}
	if text := result.GetText(); text != "" {
		return text
	}
	return core.Sprintf("score:%f", result.GetScore())
}

// Query searches for similar documents in the vector store.
// Query(ctx, store, embedder, "how do goroutines work?", DefaultQueryConfig())
func Query(ctx context.Context, store VectorStore, embedder Embedder, query string, cfg QueryConfig) ([]QueryResult, error) {
	it, err := QuerySeq(ctx, store, embedder, query, cfg)
	if err != nil {
		return nil, err
	}
	return slices.Collect(it), nil
}

// QuerySeq returns an iterator that yields query results from the vector store.
// it, _ := QuerySeq(ctx, store, embedder, "how do goroutines work?", DefaultQueryConfig())
func QuerySeq(ctx context.Context, store VectorStore, embedder Embedder, query string, cfg QueryConfig) (iter.Seq[QueryResult], error) {
	// Generate embedding for query
	embedding, err := embedder.Embed(ctx, query)
	if err != nil {
		return nil, core.E("rag.Query", "error generating query embedding", err)
	}

	// Build filter
	var filter map[string]string
	if cfg.Category != "" {
		filter = map[string]string{"category": cfg.Category}
	}

	// Search vector store
	results, err := store.Search(ctx, cfg.Collection, embedding, cfg.Limit, filter)
	if err != nil {
		return nil, core.E("rag.Query", "error searching", err)
	}

	// Filter by threshold and convert to iterator
	filteredIt := func(yield func(QueryResult) bool) {
		var queryResults []QueryResult

		for _, r := range results {
			if r.Score < cfg.Threshold {
				continue
			}

			qr := QueryResult{
				Text:       r.GetText(),
				Source:     r.GetSource(),
				Section:    r.GetSection(),
				Category:   r.GetCategory(),
				ChunkIndex: r.GetChunkIndex(),
				Score:      r.Score,
			}

			queryResults = append(queryResults, qr)
		}

		// Apply keyword boosting when enabled
		if cfg.Keywords && len(queryResults) > 0 {
			keywords := extractKeywords(query)
			if len(keywords) > 0 {
				queryResults = KeywordFilter(queryResults, keywords)
			}
		}

		queryResults = Rank(queryResults, int(cfg.Limit))

		for _, qr := range queryResults {
			if !yield(qr) {
				return
			}
		}
	}

	return filteredIt, nil
}

// FormatResultsText formats query results as plain text.
// text := FormatResultsText(results)
func FormatResultsText(results []QueryResult) string {
	if len(results) == 0 {
		return "No results found."
	}

	sb := core.NewBuilder()
	for i, r := range results {
		sb.WriteString(core.Sprintf("\n--- Result %d (score: %.2f) ---\n", i+1, r.Score))
		sb.WriteString(core.Sprintf("Source: %s\n", r.Source))
		if r.Section != "" {
			sb.WriteString(core.Sprintf("Section: %s\n", r.Section))
		}
		sb.WriteString(core.Sprintf("Category: %s\n\n", r.Category))
		sb.WriteString(r.Text)
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatResultsContext formats query results for LLM context injection.
// promptContext := FormatResultsContext(results)
func FormatResultsContext(results []QueryResult) string {
	if len(results) == 0 {
		return ""
	}

	sb := core.NewBuilder()
	sb.WriteString("<retrieved_context>\n")
	for _, r := range results {
		// Escape XML special characters to prevent malformed output
		sb.WriteString(core.Sprintf("<document source=\"%s\" section=\"%s\" category=\"%s\">\n",
			html.EscapeString(r.Source),
			html.EscapeString(r.Section),
			html.EscapeString(r.Category)))
		sb.WriteString(html.EscapeString(r.Text))
		sb.WriteString("\n</document>\n\n")
	}
	sb.WriteString("</retrieved_context>")
	return sb.String()
}

// FormatResultsJSON formats query results as JSON-like output.
// payload := FormatResultsJSON(results)
func FormatResultsJSON(results []QueryResult) string {
	if len(results) == 0 {
		return "[]"
	}

	formatted := make([]struct {
		Source   string  `json:"source"`
		Section  string  `json:"section"`
		Category string  `json:"category"`
		Score    float64 `json:"score"`
		Text     string  `json:"text"`
	}, len(results))

	for i, r := range results {
		formatted[i] = struct {
			Source   string  `json:"source"`
			Section  string  `json:"section"`
			Category string  `json:"category"`
			Score    float64 `json:"score"`
			Text     string  `json:"text"`
		}{
			Source:   r.Source,
			Section:  r.Section,
			Category: r.Category,
			Score:    math.Round(float64(r.Score)*10000) / 10000,
			Text:     r.Text,
		}
	}

	return core.JSONMarshalString(formatted)
}
