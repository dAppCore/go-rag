package rag

import (
	"context"
	"testing"

	"dappco.re/go/core"
)

// --- ListCollections tests ---

func TestCollections_ListCollections_Good(t *testing.T) {
	t.Run("returns collection names from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["alpha"] = 768
		store.collections["bravo"] = 384

		names, err := ListCollections(context.Background(), store)

		assertNoError(t, err)
		assertLen(t, names, 2)
		assertContains(t, names, "alpha")
		assertContains(t, names, "bravo")
	})

	t.Run("empty store returns empty list", func(t *testing.T) {
		store := newMockVectorStore()

		names, err := ListCollections(context.Background(), store)

		assertNoError(t, err)
		assertEmpty(t, names)
	})

	t.Run("error from store propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.listErr = core.E("mock.collections.list", "connection lost", nil)

		_, err := ListCollections(context.Background(), store)

		assertError(t, err)
		assertContains(t, err.Error(), "connection lost")
	})
}

// --- ListCollectionsSeq tests ---

func TestCollections_ListCollectionsSeq_Good(t *testing.T) {
	t.Run("yields collection names from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["alpha"] = 768
		store.collections["bravo"] = 384

		it, err := ListCollectionsSeq(context.Background(), store)

		assertNoError(t, err)
		assertNotNil(t, it)

		var names []string
		for name := range it {
			names = append(names, name)
		}
		assertLen(t, names, 2)
		assertContains(t, names, "alpha")
		assertContains(t, names, "bravo")
	})

	t.Run("empty store yields nothing", func(t *testing.T) {
		store := newMockVectorStore()

		it, err := ListCollectionsSeq(context.Background(), store)

		assertNoError(t, err)

		count := 0
		for range it {
			count++
		}
		assertEqual(t, 0, count)
	})

	t.Run("error from store returns nil iterator", func(t *testing.T) {
		store := newMockVectorStore()
		store.listErr = core.E("mock.collections.list", "connection lost", nil)

		it, err := ListCollectionsSeq(context.Background(), store)

		assertError(t, err)
		assertNil(t, it)
	})
}

// --- DeleteCollection tests ---

func TestCollections_DeleteCollection_Good(t *testing.T) {
	t.Run("deletes collection from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["to-delete"] = 768
		store.points["to-delete"] = []Point{{ID: "p1"}}

		err := DeleteCollection(context.Background(), store, "to-delete")

		assertNoError(t, err)
		_, exists := store.collections["to-delete"]
		assertFalse(t, exists)
		assertEmpty(t, store.points["to-delete"])
	})

	t.Run("error from store propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.deleteErr = core.E("mock.collections.delete", "permission denied", nil)

		err := DeleteCollection(context.Background(), store, "any")

		assertError(t, err)
		assertContains(t, err.Error(), "permission denied")
	})
}

// --- CollectionStats tests ---

func TestCollections_CollectionStats_Good(t *testing.T) {
	t.Run("returns info for existing collection", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["my-col"] = 768
		store.points["my-col"] = []Point{
			{ID: "p1", Vector: []float32{0.1}},
			{ID: "p2", Vector: []float32{0.2}},
			{ID: "p3", Vector: []float32{0.3}},
		}

		info, err := CollectionStats(context.Background(), store, "my-col")

		assertNoError(t, err)
		assertNotNil(t, info)
		assertEqual(t, "my-col", info.Name)
		assertEqual(t, uint64(3), info.Count)
		assertEqual(t, uint64(3), info.Vectors)
		assertEqual(t, uint64(3), info.PointCount)
		assertEqual(t, uint64(768), info.VectorSize)
		assertEqual(t, "hnsw", info.Index)
		assertEqual(t, "green", info.Status)
	})

	t.Run("nonexistent collection returns error", func(t *testing.T) {
		store := newMockVectorStore()

		_, err := CollectionStats(context.Background(), store, "missing")

		assertError(t, err)
		assertContains(t, err.Error(), "not found")
	})

	t.Run("error from store propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["err-col"] = 768
		store.infoErr = core.E("mock.collections.info", "internal error", nil)

		_, err := CollectionStats(context.Background(), store, "err-col")

		assertError(t, err)
		assertContains(t, err.Error(), "internal error")
	})

	t.Run("empty collection has zero point count", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["empty-col"] = 384

		info, err := CollectionStats(context.Background(), store, "empty-col")

		assertNoError(t, err)
		assertEqual(t, uint64(0), info.Count)
		assertEqual(t, uint64(0), info.Vectors)
		assertEqual(t, uint64(0), info.PointCount)
		assertEqual(t, uint64(384), info.VectorSize)
		assertEqual(t, "hnsw", info.Index)
	})
}
