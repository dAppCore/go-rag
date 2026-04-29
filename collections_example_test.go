package rag

import (
	"iter"

	core "dappco.re/go"
)

func ExampleListCollections() {
	store := newMockVectorStore()
	store.collections["docs"] = 768
	r := ListCollections(core.Background(), store)
	names := r.Value.([]string)
	core.Println(r.OK, names[0])
	// Output: true docs
}

func ExampleListCollectionsSeq() {
	store := newMockVectorStore()
	store.collections["docs"] = 768
	r := ListCollectionsSeq(core.Background(), store)
	seq := r.Value.(iter.Seq[string])
	count := 0
	for range seq {
		count++
	}
	core.Println(r.OK, count)
	// Output: true 1
}

func ExampleDeleteCollection() {
	store := newMockVectorStore()
	store.collections["docs"] = 768
	r := DeleteCollection(core.Background(), store, "docs")
	core.Println(r.OK, len(store.collections))
	// Output: true 0
}

func ExampleCollectionStats() {
	store := newMockVectorStore()
	store.collections["docs"] = 768
	r := CollectionStats(core.Background(), store, "docs")
	info := r.Value.(*CollectionInfo)
	core.Println(r.OK, info.Name, info.VectorSize)
	// Output: true docs 768
}
