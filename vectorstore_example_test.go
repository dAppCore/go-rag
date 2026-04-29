package rag

import core "dappco.re/go"

func ExampleVectorStore() {
	var store VectorStore = newMockVectorStore()
	err := store.CreateCollection(core.Background(), "docs", 768)
	core.Println(err == nil)
	// Output: true
}

func ExampleVector() {
	vector := Vector{ID: "chunk-1", Values: []float32{0.1}, Payload: map[string]any{"source": "guide.md"}}
	core.Println(vector.ID, len(vector.Values), vector.Payload["source"])
	// Output: chunk-1 1 guide.md
}

func ExampleCollectionInfo() {
	info := CollectionInfo{Name: "docs", PointCount: 2, VectorSize: 768, Status: "green"}
	core.Println(info.Name, info.PointCount, info.VectorSize, info.Status)
	// Output: docs 2 768 green
}
