package rag

import core "dappco.re/go"

func ExampleKeywordResult_GetText() {
	result := KeywordResult{Text: "kubernetes deployment"}
	core.Println(result.GetText())
	// Output: kubernetes deployment
}

func ExampleKeywordResult_GetScore() {
	result := KeywordResult{Score: 0.75}
	core.Println(result.GetScore())
	// Output: 0.75
}

func ExampleKeywordResult_GetSource() {
	result := KeywordResult{Source: "docs/search.md"}
	core.Println(result.GetSource())
	// Output: docs/search.md
}

func ExampleKeywordResult_HasChunkIndex() {
	result := KeywordResult{ChunkIndex: 3}
	core.Println(result.HasChunkIndex())
	// Output: true
}

func ExampleKeywordResult_GetChunkIndex() {
	result := KeywordResult{ChunkIndex: 3}
	core.Println(result.GetChunkIndex())
	// Output: 3
}

func ExampleSearchKeywords() {
	chunks := []Chunk{{Text: "authentication setup guide", Section: "Auth", Index: 1}}
	results := SearchKeywords(chunks, "authentication", 5)
	core.Println(len(results), results[0].Section)
	// Output: 1 Auth
}

func ExampleSearchKeywordsSeq() {
	count := 0
	for range SearchKeywordsSeq([]Chunk{{Text: "deployment guide"}}, "deployment", 5) {
		count++
	}
	core.Println(count)
	// Output: 1
}

func ExampleNewKeywordIndex() {
	idx := NewKeywordIndex([]Chunk{{Text: "deployment guide"}})
	core.Println(idx.Len())
	// Output: 1
}

func ExampleKeywordIndex_Len() {
	idx := NewKeywordIndex([]Chunk{{Text: "alpha"}, {Text: "beta"}})
	core.Println(idx.Len())
	// Output: 2
}

func ExampleKeywordIndex_Search() {
	idx := NewKeywordIndex([]Chunk{{Text: "kubernetes deployment", Index: 4}})
	results := idx.Search("kubernetes", 1)
	core.Println(results[0].ChunkIndex)
	// Output: 4
}

func ExampleKeywordFilter() {
	results := []QueryResult{{Text: "Kubernetes guide", Score: 0.8}, {Text: "Other", Score: 0.7}}
	filtered := KeywordFilter(results, []string{"kubernetes"})
	core.Println(filtered[0].Text)
	// Output: Kubernetes guide
}

func ExampleKeywordFilterSeq() {
	count := 0
	for range KeywordFilterSeq([]QueryResult{{Text: "alpha", Score: 1}}, []string{"alpha"}) {
		count++
	}
	core.Println(count)
	// Output: 1
}
