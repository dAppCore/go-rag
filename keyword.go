package rag

import (
	"iter"
	"math"
	"slices"

	"dappco.re/go/core"
)

// KeywordResult represents a keyword-search hit with TF-IDF score and
// backing chunk metadata. Emitted by KeywordIndex.Search.
//
//	result := KeywordResult{Source: "docs/guide.md", Score: 0.42, Text: "..."}
type KeywordResult struct {
	// Text is the chunk text that matched the query terms.
	Text string
	// Source is the source document path when available.
	Source string
	// Section is the Markdown section attached to the chunk.
	Section string
	// ChunkIndex is the chunk's zero-based source position.
	ChunkIndex int
	// Score is the TF-IDF relevance score.
	Score float32
}

// GetText returns the result text (satisfies textResult / rankedResult).
func (r KeywordResult) GetText() string { return r.Text }

// GetScore returns the TF-IDF similarity score.
func (r KeywordResult) GetScore() float32 { return r.Score }

// GetSource returns the source path.
func (r KeywordResult) GetSource() string { return r.Source }

// HasChunkIndex reports whether this result carries explicit chunk metadata.
func (r KeywordResult) HasChunkIndex() bool { return true }

// GetChunkIndex returns the source chunk index.
func (r KeywordResult) GetChunkIndex() int {
	return r.ChunkIndex
}

// KeywordIndex is a lightweight TF-IDF keyword search index over a fixed set
// of chunks. It is the fallback search path when Qdrant or Ollama are
// unavailable — no network, no embeddings, just bag-of-words scoring.
//
//	idx := NewKeywordIndex(chunks)
//	results := idx.Search("authentication setup", 5)
type KeywordIndex struct {
	chunks []Chunk
	// termFreq[i] is a map of term -> normalised frequency within chunk i.
	termFreq []map[string]float32
	// docFreq is a map of term -> number of chunks containing the term.
	docFreq map[string]int
	// docCount is the total number of indexed chunks.
	docCount int
}

// SearchKeywords builds a temporary TF-IDF index over the provided chunks and
// returns the top-K keyword matches for the query.
//
//	results := SearchKeywords(chunks, "authentication setup", 5)
func SearchKeywords(chunks []Chunk, query string, topK int) []KeywordResult {
	return NewKeywordIndex(chunks).Search(query, topK)
}

// SearchKeywordsSeq is the iterator form of SearchKeywords.
//
//	for result := range SearchKeywordsSeq(chunks, "authentication setup", 5) { _ = result }
func SearchKeywordsSeq(chunks []Chunk, query string, topK int) iter.Seq[KeywordResult] {
	return func(yield func(KeywordResult) bool) {
		for _, result := range SearchKeywords(chunks, query, topK) {
			if !yield(result) {
				return
			}
		}
	}
}

// NewKeywordIndex builds a TF-IDF index from the given chunks.
// Tokens shorter than 3 characters are ignored.
//
//	idx := NewKeywordIndex(chunks)
func NewKeywordIndex(chunks []Chunk) *KeywordIndex {
	idx := &KeywordIndex{
		chunks:   make([]Chunk, len(chunks)),
		termFreq: make([]map[string]float32, len(chunks)),
		docFreq:  make(map[string]int, 32),
		docCount: len(chunks),
	}
	copy(idx.chunks, chunks)

	for i, chunk := range chunks {
		tokens := tokenise(chunk.Text)
		if len(tokens) == 0 {
			idx.termFreq[i] = map[string]float32{}
			continue
		}

		counts := make(map[string]int, len(tokens))
		for _, token := range tokens {
			counts[token]++
		}

		total := float32(len(tokens))
		tf := make(map[string]float32, len(counts))
		for term, count := range counts {
			tf[term] = float32(count) / total
			idx.docFreq[term]++
		}
		idx.termFreq[i] = tf
	}

	return idx
}

// Len returns the number of indexed chunks.
// n := idx.Len()
func (idx *KeywordIndex) Len() int {
	if idx == nil {
		return 0
	}
	return idx.docCount
}

// Search returns the top-K chunks matching the query ranked by TF-IDF score.
// Query tokens shorter than 3 characters are discarded. Chunks with zero
// matches are excluded from the result.
//
//	results := idx.Search("authentication setup", 5)
func (idx *KeywordIndex) Search(query string, topK int) []KeywordResult {
	if idx == nil || idx.docCount == 0 {
		return nil
	}
	queryTerms := tokenise(query)
	if len(queryTerms) == 0 {
		return nil
	}

	// Deduplicate query terms; repeated terms shouldn't inflate scores.
	seen := make(map[string]struct{}, len(queryTerms))
	uniqueTerms := make([]string, 0, len(queryTerms))
	for _, term := range queryTerms {
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		uniqueTerms = append(uniqueTerms, term)
	}

	docCountF := float64(idx.docCount)
	scores := make([]float32, idx.docCount)

	// Smoothed IDF so that a matching term always contributes a positive weight
	// even when df == docCount (tiny corpora). idf = log((N+1)/(df+1)) + 1.
	for _, term := range uniqueTerms {
		df := idx.docFreq[term]
		if df == 0 {
			continue
		}
		idf := float32(math.Log((docCountF+1.0)/float64(df+1)) + 1.0)
		for i, tf := range idx.termFreq {
			weight, ok := tf[term]
			if !ok {
				continue
			}
			scores[i] += weight * idf
		}
	}

	results := make([]KeywordResult, 0, idx.docCount)
	for i, score := range scores {
		if score <= 0 {
			continue
		}
		chunk := idx.chunks[i]
		results = append(results, KeywordResult{
			Text:       chunk.Text,
			Section:    chunk.Section,
			ChunkIndex: chunk.Index,
			Score:      score,
		})
	}

	slices.SortFunc(results, func(a, b KeywordResult) int {
		switch {
		case a.Score > b.Score:
			return -1
		case a.Score < b.Score:
			return 1
		case a.ChunkIndex < b.ChunkIndex:
			return -1
		case a.ChunkIndex > b.ChunkIndex:
			return 1
		}
		return 0
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}
	return results
}

// tokenise splits text into lowercase tokens of length >= 3.
func tokenise(text string) []string {
	var tokens []string
	current := core.NewBuilder()

	flush := func() {
		if current.Len() == 0 {
			return
		}
		token := core.Lower(current.String())
		current.Reset()
		if len([]rune(token)) >= 3 {
			tokens = append(tokens, token)
		}
	}

	for _, r := range text {
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		case r == '.' || r == ',' || r == ';' || r == ':' || r == '!' || r == '?' || r == '"' || r == '\'' || r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}' || r == '<' || r == '>':
			flush()
		default:
			current.WriteRune(r)
		}
	}
	flush()

	return tokens
}

// KeywordFilter re-ranks query results by boosting scores for results whose
// text contains one or more of the given keywords. Matching is
// case-insensitive using core.Contains. Each keyword match adds a 10%
// boost to the original score: score *= 1.0 + 0.1 * matchCount.
// Results are re-sorted by boosted score descending.
// KeywordFilter(results, []string{"kubernetes", "containers"})
func KeywordFilter(results []QueryResult, keywords []string) []QueryResult {
	if len(keywords) == 0 || len(results) == 0 {
		return results
	}

	// Normalise keywords to lowercase once and deduplicate them so repeated
	// query terms do not inflate the boost.
	lowerKeywords := make([]string, 0, len(keywords))
	seen := make(map[string]struct{}, len(keywords))
	for _, kw := range keywords {
		kw = core.Lower(kw)
		if kw == "" {
			continue
		}
		if _, ok := seen[kw]; ok {
			continue
		}
		seen[kw] = struct{}{}
		lowerKeywords = append(lowerKeywords, kw)
	}

	// Apply boost
	boosted := make([]QueryResult, len(results))
	copy(boosted, results)

	for i := range boosted {
		lowerText := core.Lower(boosted[i].Text)
		matchCount := 0
		for _, kw := range lowerKeywords {
			if kw != "" && core.Contains(lowerText, kw) {
				matchCount++
			}
		}
		if matchCount > 0 {
			boosted[i].Score *= 1.0 + 0.1*float32(matchCount)
		}
	}

	// Re-sort by boosted score descending
	slices.SortFunc(boosted, func(a, b QueryResult) int {
		if a.Score > b.Score {
			return -1
		} else if a.Score < b.Score {
			return 1
		}
		if a.Source < b.Source {
			return -1
		}
		if a.Source > b.Source {
			return 1
		}
		if a.ChunkIndex < b.ChunkIndex {
			return -1
		}
		if a.ChunkIndex > b.ChunkIndex {
			return 1
		}
		if a.Text < b.Text {
			return -1
		}
		if a.Text > b.Text {
			return 1
		}
		return 0
	})

	return boosted
}

// KeywordFilterSeq is an iterator version of KeywordFilter.
// for result := range KeywordFilterSeq(results, []string{"kubernetes"}) { _ = result }
func KeywordFilterSeq(results []QueryResult, keywords []string) iter.Seq[QueryResult] {
	return func(yield func(QueryResult) bool) {
		filtered := KeywordFilter(results, keywords)
		for _, r := range filtered {
			if !yield(r) {
				return
			}
		}
	}
}

// extractKeywords splits query text into individual keywords for filtering.
// Words shorter than 3 characters are discarded as they tend to be noise.
func extractKeywords(query string) []string {
	return slices.Collect(extractKeywordsSeq(query))
}

// extractKeywordsSeq returns an iterator that yields keywords from a query.
func extractKeywordsSeq(query string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, w := range tokenise(query) {
			if !yield(w) {
				return
			}
		}
	}
}

func fields(text string) []string {
	var words []string
	current := core.NewBuilder()

	flush := func() {
		if current.Len() == 0 {
			return
		}
		words = append(words, current.String())
		current.Reset()
	}

	for _, r := range text {
		switch r {
		case ' ', '\t', '\n', '\r':
			flush()
		default:
			current.WriteRune(r)
		}
	}
	flush()

	return words
}
