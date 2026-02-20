package rag

import (
	"sort"
	"strings"
)

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
	lowerKeywords := make([]string, len(keywords))
	for i, kw := range keywords {
		lowerKeywords[i] = strings.ToLower(kw)
	}

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
	sort.Slice(boosted, func(i, j int) bool {
		return boosted[i].Score > boosted[j].Score
	})

	return boosted
}

// extractKeywords splits query text into individual keywords for filtering.
// Words shorter than 3 characters are discarded as they tend to be noise.
func extractKeywords(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	var keywords []string
	for _, w := range words {
		if len(w) >= 3 {
			keywords = append(keywords, w)
		}
	}
	return keywords
}
