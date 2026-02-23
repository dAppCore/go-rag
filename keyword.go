package rag

import (
	"iter"
	"slices"
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
