package rag

import core "dappco.re/go"

func TestAX7_KeywordResult_GetText_Good(t *core.T) {
	result := KeywordResult{Text: "kubernetes deployment"}

	core.AssertEqual(t, "kubernetes deployment", result.GetText())
	core.AssertNotEmpty(t, result.GetText())
}

func TestAX7_KeywordResult_GetText_Bad(t *core.T) {
	result := KeywordResult{}

	core.AssertEqual(t, "", result.GetText())
	core.AssertEmpty(t, result.GetText())
}

func TestAX7_KeywordResult_GetText_Ugly(t *core.T) {
	result := KeywordResult{Text: "emoji 😀 deployment"}

	core.AssertContains(t, result.GetText(), "😀")
	core.AssertEqual(t, "emoji 😀 deployment", result.GetText())
}

func TestAX7_KeywordResult_GetScore_Good(t *core.T) {
	result := KeywordResult{Score: 0.75}

	core.AssertEqual(t, float32(0.75), result.GetScore())
	core.AssertGreater(t, result.GetScore(), float32(0))
}

func TestAX7_KeywordResult_GetScore_Bad(t *core.T) {
	result := KeywordResult{}

	core.AssertEqual(t, float32(0), result.GetScore())
	core.AssertFalse(t, result.GetScore() > 0)
}

func TestAX7_KeywordResult_GetScore_Ugly(t *core.T) {
	result := KeywordResult{Score: -1}

	core.AssertEqual(t, float32(-1), result.GetScore())
	core.AssertLess(t, result.GetScore(), float32(0))
}

func TestAX7_KeywordResult_GetSource_Good(t *core.T) {
	result := KeywordResult{Source: "docs/search.md"}

	core.AssertEqual(t, "docs/search.md", result.GetSource())
	core.AssertContains(t, result.GetSource(), "docs")
}

func TestAX7_KeywordResult_GetSource_Bad(t *core.T) {
	result := KeywordResult{}

	core.AssertEqual(t, "", result.GetSource())
	core.AssertEmpty(t, result.GetSource())
}

func TestAX7_KeywordResult_GetSource_Ugly(t *core.T) {
	result := KeywordResult{Source: "docs/space name.md"}

	core.AssertContains(t, result.GetSource(), "space name")
	core.AssertEqual(t, "docs/space name.md", result.GetSource())
}

func TestAX7_KeywordResult_HasChunkIndex_Good(t *core.T) {
	result := KeywordResult{ChunkIndex: 3}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, 3, result.GetChunkIndex())
}

func TestAX7_KeywordResult_HasChunkIndex_Bad(t *core.T) {
	result := KeywordResult{}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, 0, result.GetChunkIndex())
}

func TestAX7_KeywordResult_HasChunkIndex_Ugly(t *core.T) {
	result := KeywordResult{ChunkIndex: -9}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, -9, result.GetChunkIndex())
}

func TestAX7_KeywordResult_GetChunkIndex_Good(t *core.T) {
	result := KeywordResult{ChunkIndex: 7}

	core.AssertEqual(t, 7, result.GetChunkIndex())
	core.AssertGreater(t, result.GetChunkIndex(), 0)
}

func TestAX7_KeywordResult_GetChunkIndex_Bad(t *core.T) {
	result := KeywordResult{}

	core.AssertEqual(t, 0, result.GetChunkIndex())
	core.AssertFalse(t, result.GetChunkIndex() > 0)
}

func TestAX7_KeywordResult_GetChunkIndex_Ugly(t *core.T) {
	result := KeywordResult{ChunkIndex: -1}

	core.AssertEqual(t, -1, result.GetChunkIndex())
	core.AssertLess(t, result.GetChunkIndex(), 0)
}

func TestAX7_NewKeywordIndex_Good(t *core.T) {
	idx := NewKeywordIndex([]Chunk{{Text: "Kubernetes deployment guide", Index: 0}})

	core.AssertNotNil(t, idx)
	core.AssertEqual(t, 1, idx.Len())
}

func TestAX7_NewKeywordIndex_Bad(t *core.T) {
	idx := NewKeywordIndex(nil)

	core.AssertNotNil(t, idx)
	core.AssertEqual(t, 0, idx.Len())
}

func TestAX7_NewKeywordIndex_Ugly(t *core.T) {
	source := []Chunk{{Text: "Mutable text", Index: 0}}
	idx := NewKeywordIndex(source)
	source[0].Text = "Changed"

	core.AssertEqual(t, 1, idx.Len())
	core.AssertEqual(t, "Mutable text", idx.Search("mutable", 1)[0].Text)
}

func TestAX7_KeywordIndex_Len_Good(t *core.T) {
	idx := NewKeywordIndex([]Chunk{{Text: "alpha"}, {Text: "beta"}})

	core.AssertEqual(t, 2, idx.Len())
	core.AssertGreater(t, idx.Len(), 1)
}

func TestAX7_KeywordIndex_Len_Bad(t *core.T) {
	var idx *KeywordIndex

	core.AssertEqual(t, 0, idx.Len())
	core.AssertFalse(t, idx.Len() > 0)
}

func TestAX7_KeywordIndex_Len_Ugly(t *core.T) {
	idx := NewKeywordIndex([]Chunk{{Text: ""}})

	core.AssertEqual(t, 1, idx.Len())
	core.AssertEmpty(t, idx.Search("missing", 5))
}

func TestAX7_KeywordIndex_Search_Good(t *core.T) {
	idx := NewKeywordIndex([]Chunk{{Text: "Kubernetes deployment", Section: "Ops", Index: 4}})
	results := idx.Search("kubernetes", 5)

	core.AssertLen(t, results, 1)
	core.AssertEqual(t, 4, results[0].ChunkIndex)
}

func TestAX7_KeywordIndex_Search_Bad(t *core.T) {
	idx := NewKeywordIndex([]Chunk{{Text: "Kubernetes deployment"}})
	results := idx.Search("zzzz", 5)

	core.AssertEmpty(t, results)
	core.AssertEqual(t, 0, len(results))
}

func TestAX7_KeywordIndex_Search_Ugly(t *core.T) {
	idx := NewKeywordIndex([]Chunk{{Text: "golang golang golang"}, {Text: "golang deployment"}})
	results := idx.Search("golang golang", 1)

	core.AssertLen(t, results, 1)
	core.AssertContains(t, results[0].Text, "golang")
}

func TestAX7_SearchKeywords_Bad(t *core.T) {
	results := SearchKeywords(nil, "kubernetes", 5)

	core.AssertEmpty(t, results)
	core.AssertEqual(t, 0, len(results))
}

func TestAX7_SearchKeywords_Ugly(t *core.T) {
	chunks := []Chunk{{Text: "alpha beta", Index: 0}, {Text: "alpha gamma", Index: 1}}
	results := SearchKeywords(chunks, "alpha alpha", 10)

	core.AssertLen(t, results, 2)
	core.AssertGreaterOrEqual(t, results[0].Score, results[1].Score)
}

func TestAX7_SearchKeywordsSeq_Good(t *core.T) {
	var results []KeywordResult
	for result := range SearchKeywordsSeq([]Chunk{{Text: "searchable deployment", Index: 0}}, "deployment", 3) {
		results = append(results, result)
	}

	core.AssertLen(t, results, 1)
	core.AssertEqual(t, 0, results[0].ChunkIndex)
}

func TestAX7_SearchKeywordsSeq_Bad(t *core.T) {
	var results []KeywordResult
	for result := range SearchKeywordsSeq(nil, "deployment", 3) {
		results = append(results, result)
	}

	core.AssertEmpty(t, results)
	core.AssertEqual(t, 0, len(results))
}

func TestAX7_SearchKeywordsSeq_Ugly(t *core.T) {
	count := 0
	for range SearchKeywordsSeq([]Chunk{{Text: "alpha beta"}, {Text: "alpha gamma"}}, "alpha", 2) {
		count++
		break
	}

	core.AssertEqual(t, 1, count)
	core.AssertTrue(t, count < 2)
}

func TestAX7_KeywordFilter_Bad(t *core.T) {
	results := []QueryResult{{Text: "alpha", Score: 0.5}}
	filtered := KeywordFilter(results, nil)

	core.AssertEqual(t, results, filtered)
	core.AssertEqual(t, float32(0.5), filtered[0].Score)
}

func TestAX7_KeywordFilter_Ugly(t *core.T) {
	results := []QueryResult{{Text: "Kubernetes deployment", Score: 1}, {Text: "Other", Score: 1}}
	filtered := KeywordFilter(results, []string{"kubernetes", "kubernetes", ""})

	core.AssertEqual(t, "Kubernetes deployment", filtered[0].Text)
	core.AssertGreater(t, filtered[0].Score, filtered[1].Score)
}

func TestAX7_KeywordFilterSeq_Bad(t *core.T) {
	var results []QueryResult
	for result := range KeywordFilterSeq(nil, []string{"alpha"}) {
		results = append(results, result)
	}

	core.AssertEmpty(t, results)
	core.AssertEqual(t, 0, len(results))
}

func TestAX7_KeywordFilterSeq_Ugly(t *core.T) {
	var results []QueryResult
	input := []QueryResult{{Text: "alpha match", Score: 0.5}, {Text: "beta", Score: 0.6}}
	for result := range KeywordFilterSeq(input, []string{"alpha"}) {
		results = append(results, result)
	}

	core.AssertLen(t, results, 2)
	core.AssertGreater(t, results[0].Score, float32(0.5))
}

func TestAX7_QueryResult_GetText_Good(t *core.T) {
	result := QueryResult{Text: "answer text"}

	core.AssertEqual(t, "answer text", result.GetText())
	core.AssertNotEmpty(t, result.GetText())
}

func TestAX7_QueryResult_GetText_Bad(t *core.T) {
	result := QueryResult{}

	core.AssertEqual(t, "", result.GetText())
	core.AssertEmpty(t, result.GetText())
}

func TestAX7_QueryResult_GetText_Ugly(t *core.T) {
	result := QueryResult{Text: "<xml>&text"}

	core.AssertContains(t, result.GetText(), "&")
	core.AssertEqual(t, "<xml>&text", result.GetText())
}

func TestAX7_QueryResult_GetScore_Good(t *core.T) {
	result := QueryResult{Score: 0.8}

	core.AssertEqual(t, float32(0.8), result.GetScore())
	core.AssertGreater(t, result.GetScore(), float32(0))
}

func TestAX7_QueryResult_GetScore_Bad(t *core.T) {
	result := QueryResult{}

	core.AssertEqual(t, float32(0), result.GetScore())
	core.AssertFalse(t, result.GetScore() > 0)
}

func TestAX7_QueryResult_GetScore_Ugly(t *core.T) {
	result := QueryResult{Score: -0.2}

	core.AssertEqual(t, float32(-0.2), result.GetScore())
	core.AssertLess(t, result.GetScore(), float32(0))
}

func TestAX7_QueryResult_GetSource_Good(t *core.T) {
	result := QueryResult{Source: "docs/source.md"}

	core.AssertEqual(t, "docs/source.md", result.GetSource())
	core.AssertContains(t, result.GetSource(), "source")
}

func TestAX7_QueryResult_GetSource_Bad(t *core.T) {
	result := QueryResult{}

	core.AssertEqual(t, "", result.GetSource())
	core.AssertEmpty(t, result.GetSource())
}

func TestAX7_QueryResult_GetSource_Ugly(t *core.T) {
	result := QueryResult{Source: "docs/source with spaces.md"}

	core.AssertContains(t, result.GetSource(), "spaces")
	core.AssertEqual(t, "docs/source with spaces.md", result.GetSource())
}

func TestAX7_QueryResult_HasChunkIndex_Good(t *core.T) {
	result := QueryResult{ChunkIndex: 0, ChunkIndexPresent: true}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, 0, result.GetChunkIndex())
}

func TestAX7_QueryResult_HasChunkIndex_Bad(t *core.T) {
	result := QueryResult{}

	core.AssertFalse(t, result.HasChunkIndex())
	core.AssertEqual(t, missingChunkIndex, result.GetChunkIndex())
}

func TestAX7_QueryResult_HasChunkIndex_Ugly(t *core.T) {
	result := QueryResult{Index: 9, IndexPresent: true}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, 9, result.GetChunkIndex())
}

func TestAX7_QueryResult_GetChunkIndex_Good(t *core.T) {
	result := QueryResult{ChunkIndex: 5, ChunkIndexPresent: true}

	core.AssertEqual(t, 5, result.GetChunkIndex())
	core.AssertTrue(t, result.HasChunkIndex())
}

func TestAX7_QueryResult_GetChunkIndex_Bad(t *core.T) {
	result := QueryResult{}

	core.AssertEqual(t, missingChunkIndex, result.GetChunkIndex())
	core.AssertFalse(t, result.HasChunkIndex())
}

func TestAX7_QueryResult_GetChunkIndex_Ugly(t *core.T) {
	result := QueryResult{Index: 0, IndexPresent: true}

	core.AssertEqual(t, 0, result.GetChunkIndex())
	core.AssertTrue(t, result.HasChunkIndex())
}

func TestAX7_DefaultQueryConfig_Bad(t *core.T) {
	cfg := DefaultQueryConfig()

	core.AssertNotEqual(t, "", cfg.Collection)
	core.AssertNotEqual(t, uint64(0), cfg.Limit)
}

func TestAX7_DefaultQueryConfig_Ugly(t *core.T) {
	cfg := DefaultQueryConfig()
	cfg.Collection = "mutated"

	core.AssertEqual(t, "hostuk-docs", DefaultQueryConfig().Collection)
	core.AssertEqual(t, "mutated", cfg.Collection)
}

func TestAX7_Rank_Good(t *core.T) {
	results := []QueryResult{{Text: "low", Score: 0.1}, {Text: "high", Score: 0.9}}
	ranked := Rank(results, 1)

	core.AssertLen(t, ranked, 1)
	core.AssertEqual(t, "high", ranked[0].Text)
}

func TestAX7_Rank_Bad(t *core.T) {
	ranked := Rank([]QueryResult{{Text: "ignored", Score: 1}}, 0)

	core.AssertEmpty(t, ranked)
	core.AssertEqual(t, 0, len(ranked))
}

func TestAX7_Rank_Ugly(t *core.T) {
	results := []QueryResult{{Text: "dup", Source: "a.md", ChunkIndex: 1, ChunkIndexPresent: true, Score: 0.9}, {Text: "dup", Source: "a.md", ChunkIndex: 1, ChunkIndexPresent: true, Score: 0.8}}
	ranked := Rank(results, 5)

	core.AssertLen(t, ranked, 1)
	core.AssertEqual(t, float32(0.9), ranked[0].Score)
}

func TestAX7_Query_Bad(t *core.T) {
	store := newMockVectorStore()
	store.searchErr = core.NewError("search failed")
	_, err := Query(core.Background(), store, newMockEmbedder(2), "query", DefaultQueryConfig())

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "error searching")
}

func TestAX7_Query_Ugly(t *core.T) {
	store := newMockVectorStore()
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 0.1, Payload: map[string]any{"text": "low"}}}, nil
	}
	results, err := Query(core.Background(), store, newMockEmbedder(2), "query", QueryConfig{Collection: "docs", Limit: 5, Threshold: 0.9})

	core.AssertNoError(t, err)
	core.AssertEmpty(t, results)
}

func TestAX7_QuerySeq_Good(t *core.T) {
	store := newMockVectorStore()
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 0.9, Payload: map[string]any{"text": "hit", "source": "a.md", "chunk_index": 0}}}, nil
	}
	seq, err := QuerySeq(core.Background(), store, newMockEmbedder(2), "query", QueryConfig{Collection: "docs", Limit: 5, Threshold: 0.1})

	var results []QueryResult
	for result := range seq {
		results = append(results, result)
	}
	core.AssertNoError(t, err)
	core.AssertLen(t, results, 1)
}

func TestAX7_QuerySeq_Bad(t *core.T) {
	embedder := newMockEmbedder(2)
	embedder.embedErr = core.NewError("embed failed")
	seq, err := QuerySeq(core.Background(), newMockVectorStore(), embedder, "query", DefaultQueryConfig())

	core.AssertError(t, err)
	core.AssertNil(t, seq)
}

func TestAX7_QuerySeq_Ugly(t *core.T) {
	store := newMockVectorStore()
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 1, Payload: map[string]any{"text": "Kubernetes"}}, {Score: 0.95, Payload: map[string]any{"text": "Other"}}}, nil
	}
	seq, err := QuerySeq(core.Background(), store, newMockEmbedder(2), "kubernetes", QueryConfig{Collection: "docs", Limit: 5, Threshold: 0.1, Keywords: true})

	var results []QueryResult
	for result := range seq {
		results = append(results, result)
	}
	core.AssertNoError(t, err)
	core.AssertEqual(t, "Kubernetes", results[0].Text)
}

func TestAX7_QueryWith_Bad(t *core.T) {
	embedder := newMockEmbedder(2)
	embedder.embedErr = core.NewError("embed failed")
	_, err := QueryWith(core.Background(), newMockVectorStore(), embedder, "query", "docs", 3)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "embedding")
}

func TestAX7_QueryWith_Ugly(t *core.T) {
	store := newMockVectorStore()
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 1, Payload: map[string]any{"text": "hit"}}}, nil
	}
	results, err := QueryWith(core.Background(), store, newMockEmbedder(2), "query", "docs", -1)

	core.AssertNoError(t, err)
	core.AssertEmpty(t, results)
}

func TestAX7_QueryContextWith_Bad(t *core.T) {
	embedder := newMockEmbedder(2)
	embedder.embedErr = core.NewError("embed failed")
	text, err := QueryContextWith(core.Background(), newMockVectorStore(), embedder, "query", "docs", 3)

	core.AssertError(t, err)
	core.AssertEqual(t, "", text)
}

func TestAX7_QueryContextWith_Ugly(t *core.T) {
	store := newMockVectorStore()
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return nil, nil
	}
	text, err := QueryContextWith(core.Background(), store, newMockEmbedder(2), "query", "docs", 3)

	core.AssertNoError(t, err)
	core.AssertEqual(t, "", text)
}

func TestAX7_FormatResultsText_Bad(t *core.T) {
	text := FormatResultsText(nil)

	core.AssertEqual(t, "No results found.", text)
	core.AssertNotContains(t, text, "Result 1")
}

func TestAX7_FormatResultsText_Ugly(t *core.T) {
	text := FormatResultsText([]QueryResult{{Text: "", Source: "", Category: "", Score: 0}})

	core.AssertContains(t, text, "score: 0.00")
	core.AssertContains(t, text, "Category:")
}

func TestAX7_FormatResultsContext_Bad(t *core.T) {
	context := FormatResultsContext(nil)

	core.AssertEqual(t, "", context)
	core.AssertEmpty(t, context)
}

func TestAX7_FormatResultsContext_Ugly(t *core.T) {
	context := FormatResultsContext([]QueryResult{{Text: "<tag>&", Source: "a&b.md", Section: "\"sec\""}})

	core.AssertContains(t, context, "&lt;tag&gt;&amp;")
	core.AssertContains(t, context, "a&amp;b.md")
}

func TestAX7_FormatResultsJSON_Bad(t *core.T) {
	json := FormatResultsJSON(nil)

	core.AssertEqual(t, "[]", json)
	core.AssertLen(t, json, 2)
}

func TestAX7_FormatResultsJSON_Ugly(t *core.T) {
	json := FormatResultsJSON([]QueryResult{{Text: "line\nquote\"", Source: "a.md", Score: 0.123456}})

	core.AssertContains(t, json, "0.1235")
	core.AssertContains(t, json, `quote\"`)
}

func TestAX7_JoinResults_Bad(t *core.T) {
	output := JoinResults[QueryResult](nil)

	core.AssertEqual(t, "", output)
	core.AssertEmpty(t, output)
}

func TestAX7_JoinResults_Ugly(t *core.T) {
	output := JoinResults([]QueryResult{{Text: "  Alpha  "}, {Text: ""}, {Text: "\nBeta\n"}})

	core.AssertEqual(t, "Alpha\n\nBeta", output)
	core.AssertNotContains(t, output, "\n\n\n")
}
