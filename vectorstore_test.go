package rag

import core "dappco.re/go"

func TestVectorstore_VectorStore_Good(t *core.T) {
	var store VectorStore = newMockVectorStore()

	core.AssertNotNil(t, store)
	core.AssertFalse(t, store == nil)
}

func TestVectorstore_VectorStore_Bad(t *core.T) {
	var store VectorStore

	core.AssertNil(t, store)
	core.AssertTrue(t, store == nil)
}

func TestVectorstore_VectorStore_Ugly(t *core.T) {
	store := newMockVectorStore()
	var _ VectorStore = store

	err := store.CreateCollection(core.Background(), "docs", 2)
	core.AssertNoError(t, err)
	core.AssertEqual(t, uint64(2), store.collections["docs"])
}

func TestVectorstore_Vector_Good(t *core.T) {
	vector := Vector{ID: "chunk-1", Values: []float32{0.1, 0.2}, Payload: map[string]any{"source": "guide.md"}}

	core.AssertEqual(t, "chunk-1", vector.ID)
	core.AssertLen(t, vector.Values, 2)
	core.AssertEqual(t, "guide.md", vector.Payload["source"])
}

func TestVectorstore_Vector_Bad(t *core.T) {
	vector := Vector{}

	core.AssertEqual(t, "", vector.ID)
	core.AssertEmpty(t, vector.Values)
	core.AssertNil(t, vector.Payload)
}

func TestVectorstore_Vector_Ugly(t *core.T) {
	vector := Vector{Payload: map[string]any{}}
	vector.Payload["chunk_index"] = 0

	core.AssertEqual(t, 0, vector.Payload["chunk_index"])
	core.AssertEmpty(t, vector.Values)
}

func TestVectorstore_CollectionInfo_Good(t *core.T) {
	info := CollectionInfo{Name: "docs", Count: 2, Vectors: 2, PointCount: 2, VectorSize: 768, Index: "hnsw", Status: "green"}

	core.AssertEqual(t, "docs", info.Name)
	core.AssertEqual(t, uint64(768), info.VectorSize)
	core.AssertEqual(t, "green", info.Status)
}

func TestVectorstore_CollectionInfo_Bad(t *core.T) {
	info := CollectionInfo{}

	core.AssertEqual(t, "", info.Name)
	core.AssertEqual(t, uint64(0), info.PointCount)
	core.AssertEqual(t, "", info.Status)
}

func TestVectorstore_CollectionInfo_Ugly(t *core.T) {
	info := CollectionInfo{Name: "empty", Count: 0, Vectors: 0, PointCount: 0, VectorSize: 384, Status: "yellow"}

	core.AssertEqual(t, uint64(0), info.Count)
	core.AssertEqual(t, uint64(384), info.VectorSize)
	core.AssertEqual(t, "yellow", info.Status)
}
