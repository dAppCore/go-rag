package rag

import (
	"context"
	"iter"
	"testing"

	"dappco.re/go"
)

// --- ListCollections tests ---

func TestCollections_ListCollections_Good(t *testing.T) {
	t.Run("returns collection names from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["alpha"] = 768
		store.collections["bravo"] = 384

		r := ListCollections(context.Background(), store)
		names := resultValue[[]string](t, r)

		assertLen(t, names, 2)
		assertContains(t, names, "alpha")
		assertContains(t, names, "bravo")
	})

	t.Run("empty store returns empty list", func(t *testing.T) {
		store := newMockVectorStore()

		r := ListCollections(context.Background(), store)
		names := resultValue[[]string](t, r)

		assertEmpty(t, names)
	})

	t.Run("error from store propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.listErr = core.E("mock.collections.list", "connection lost", nil)

		r := ListCollections(context.Background(), store)

		assertError(t, r)
		assertContains(t, r.Error(), "connection lost")
	})
}

// --- ListCollectionsSeq tests ---

func TestCollections_ListCollectionsSeq_Good(t *testing.T) {
	t.Run("yields collection names from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["alpha"] = 768
		store.collections["bravo"] = 384

		r := ListCollectionsSeq(context.Background(), store)
		it := resultValue[iter.Seq[string]](t, r)

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

		r := ListCollectionsSeq(context.Background(), store)
		it := resultValue[iter.Seq[string]](t, r)

		count := 0
		for range it {
			count++
		}
		assertEqual(t, 0, count)
	})

	t.Run("error from store returns nil iterator", func(t *testing.T) {
		store := newMockVectorStore()
		store.listErr = core.E("mock.collections.list", "connection lost", nil)

		r := ListCollectionsSeq(context.Background(), store)

		assertError(t, r)
		assertContains(t, r.Error(), "connection lost")
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

		r := CollectionStats(context.Background(), store, "my-col")
		info := resultValue[*CollectionInfo](t, r)

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

		r := CollectionStats(context.Background(), store, "missing")

		assertError(t, r)
		assertContains(t, r.Error(), "not found")
	})

	t.Run("error from store propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["err-col"] = 768
		store.infoErr = core.E("mock.collections.info", "internal error", nil)

		r := CollectionStats(context.Background(), store, "err-col")

		assertError(t, r)
		assertContains(t, r.Error(), "internal error")
	})

	t.Run("empty collection has zero point count", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["empty-col"] = 384

		r := CollectionStats(context.Background(), store, "empty-col")
		info := resultValue[*CollectionInfo](t, r)

		assertEqual(t, uint64(0), info.Count)
		assertEqual(t, uint64(0), info.Vectors)
		assertEqual(t, uint64(0), info.PointCount)
		assertEqual(t, uint64(384), info.VectorSize)
		assertEqual(t, "hnsw", info.Index)
	})
}

func TestCollections_ListCollections_Bad(t *core.T) {
	store := newMockVectorStore()
	store.listErr = core.NewError("list failed")
	r := ListCollections(core.Background(), store)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "list failed")
}

func TestCollections_ListCollections_Ugly(t *core.T) {
	store := newMockVectorStore()
	r := ListCollections(core.Background(), store)
	names := r.Value.([]string)

	core.AssertTrue(t, r.OK)
	core.AssertEmpty(t, names)
}

func TestCollections_ListCollectionsSeq_Bad(t *core.T) {
	store := newMockVectorStore()
	store.listErr = core.NewError("list failed")
	r := ListCollectionsSeq(core.Background(), store)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "list failed")
}

func TestCollections_ListCollectionsSeq_Ugly(t *core.T) {
	store := newMockVectorStore()
	r := ListCollectionsSeq(core.Background(), store)
	seq := r.Value.(iter.Seq[string])
	count := 0
	for range seq {
		count++
	}

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, 0, count)
}

func TestCollections_DeleteCollection_Bad(t *core.T) {
	store := newMockVectorStore()
	store.deleteErr = core.NewError("delete failed")
	r := DeleteCollection(core.Background(), store, "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "delete failed")
}

func TestCollections_DeleteCollection_Ugly(t *core.T) {
	store := newMockVectorStore()
	r := DeleteCollection(core.Background(), store, "")

	core.AssertTrue(t, r.OK)
	core.AssertLen(t, store.deleteCalls, 1)
}

func TestCollections_CollectionStats_Bad(t *core.T) {
	store := newMockVectorStore()
	store.infoErr = core.NewError("info failed")
	r := CollectionStats(core.Background(), store, "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "info failed")
}

func TestCollections_CollectionStats_Ugly(t *core.T) {
	store := newMockVectorStore()
	store.collections["empty"] = 384
	r := CollectionStats(core.Background(), store, "empty")
	info := r.Value.(*CollectionInfo)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, uint64(0), info.PointCount)
}
