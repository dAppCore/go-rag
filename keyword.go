package rag

import (
	"math"
	"iter"
	"slices"
	"strings"
	"unicode"
)

// KeywordResult represents a TF-IDF keyword search hit.
type KeywordResult struct {
	Text    string
	Section string
	Index   int
	Score   float32
}

type keywordDocument struct {
	chunk  Chunk
	terms  map[string]int
	length int
}

// KeywordIndex is a small TF-IDF index for fallback keyword search.
type KeywordIndex struct {
	docs      []keywordDocument
	docFreq   map[string]int
	totalDocs int
}

// NewKeywordIndex builds a keyword index from the provided chunks.
func NewKeywordIndex(chunks []Chunk) *KeywordIndex {
	index := &KeywordIndex{
		docFreq: make(map[string]int),
	}

	for _, chunk := range chunks {
		tokens := tokenizeKeywords(chunk.Text)
		if len(tokens) == 0 {
			continue
		}

		terms := make(map[string]int, len(tokens))
		seen := make(map[string]struct{}, len(tokens))
		for _, token := range tokens {
			terms[token]++
			if _, ok := seen[token]; !ok {
				index.docFreq[token]++
				seen[token] = struct{}{}
			}
		}

		index.docs = append(index.docs, keywordDocument{
			chunk:  chunk,
			terms:  terms,
			length: len(tokens),
		})
	}

	index.totalDocs = len(index.docs)
	return index
}

// Search performs a simple TF-IDF search across the indexed chunks.
func (ki *KeywordIndex) Search(query string, topK int) []KeywordResult {
	if ki == nil || ki.totalDocs == 0 || topK <= 0 {
		return nil
	}

	queryTerms := tokenizeKeywords(query)
	if len(queryTerms) == 0 {
		return nil
	}

	queryFreq := make(map[string]int, len(queryTerms))
	for _, term := range queryTerms {
		queryFreq[term]++
	}

	results := make([]KeywordResult, 0, len(ki.docs))
	for _, doc := range ki.docs {
		score := 0.0
		for term, qtf := range queryFreq {
			tf, ok := doc.terms[term]
			if !ok {
				continue
			}

			df := ki.docFreq[term]
			idf := math.Log((float64(ki.totalDocs)+1.0)/(float64(df)+1.0)) + 1.0
			tfNorm := float64(tf) / float64(doc.length)
			score += tfNorm * idf * float64(qtf)
		}

		if score <= 0 {
			continue
		}

		results = append(results, KeywordResult{
			Text:    doc.chunk.Text,
			Section: doc.chunk.Section,
			Index:   doc.chunk.Index,
			Score:   float32(score),
		})
	}

	slices.SortFunc(results, func(a, b KeywordResult) int {
		if a.Score > b.Score {
			return -1
		}
		if a.Score < b.Score {
			return 1
		}
		if a.Index < b.Index {
			return -1
		}
		if a.Index > b.Index {
			return 1
		}
		return 0
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

func tokenizeKeywords(text string) []string {
	normalized := strings.ToLower(text)
	normalized = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return ' '
	}, normalized)

	tokens := strings.Fields(normalized)
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if len(token) >= 3 {
			out = append(out, token)
		}
	}
	return out
}

// KeywordFilter re-ranks query results by boosting scores for results whose
// text contains one or more of the given keywords. Matching is
// case-insensitive using strings.Contains. Each keyword match adds a 10%
// boost to the original score: score *= 1.0 + 0.1 * matchCount.
// Results are re-sorted by boosted score descending.
func KeywordFilter(results []QueryResult, keywords []string) []QueryResult {
	if len(keywords) == 0 || len(results) == 0 {
		return results
	}

	// Normalise keywords to lowercase once
	lowerKeywords := slices.Collect(func(yield func(string) bool) {
		for _, kw := range keywords {
			if !yield(strings.ToLower(kw)) {
				return
			}
		}
	})

	// Apply boost
	boosted := make([]QueryResult, len(results))
	copy(boosted, results)

	for i := range boosted {
		lowerText := strings.ToLower(boosted[i].Text)
		matchCount := 0
		for _, kw := range lowerKeywords {
			if kw != "" && strings.Contains(lowerText, kw) {
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
		return 0
	})

	return boosted
}

// KeywordFilterSeq is an iterator version of KeywordFilter.
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
		for w := range strings.FieldsSeq(strings.ToLower(query)) {
			if len(w) >= 3 {
				if !yield(w) {
					return
				}
			}
		}
	}
}
