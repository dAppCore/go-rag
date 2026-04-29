package rag

import core "dappco.re/go"

func ExampleListCollections() {
	store := newMockVectorStore()
	store.collections["docs"] = 768
	names, err := ListCollections(core.Background(), store)
	core.Println(err == nil, names[0])
	// Output: true docs
}

func ExampleListCollectionsSeq() {
	store := newMockVectorStore()
	store.collections["docs"] = 768
	seq, err := ListCollectionsSeq(core.Background(), store)
	count := 0
	for range seq {
		count++
	}
	core.Println(err == nil, count)
	// Output: true 1
}

func ExampleDeleteCollection() {
	store := newMockVectorStore()
	store.collections["docs"] = 768
	err := DeleteCollection(core.Background(), store, "docs")
	core.Println(err == nil, len(store.collections))
	// Output: true 0
}

func ExampleCollectionStats() {
	store := newMockVectorStore()
	store.collections["docs"] = 768
	info, err := CollectionStats(core.Background(), store, "docs")
	core.Println(err == nil, info.Name, info.VectorSize)
	// Output: true docs 768
}
