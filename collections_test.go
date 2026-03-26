package rag

import (
	"context"
	"testing"

	"dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ListCollections tests ---

func TestCollections_ListCollections_Good(t *testing.T) {
	t.Run("returns collection names from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["alpha"] = 768
		store.collections["bravo"] = 384

		names, err := ListCollections(context.Background(), store)

		require.NoError(t, err)
		assert.Len(t, names, 2)
		assert.Contains(t, names, "alpha")
		assert.Contains(t, names, "bravo")
	})

	t.Run("empty store returns empty list", func(t *testing.T) {
		store := newMockVectorStore()

		names, err := ListCollections(context.Background(), store)

		require.NoError(t, err)
		assert.Empty(t, names)
	})

	t.Run("error from store propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.listErr = core.E("mock.collections.list", "connection lost", nil)

		_, err := ListCollections(context.Background(), store)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection lost")
	})
}

// --- ListCollectionsSeq tests ---

func TestCollections_ListCollectionsSeq_Good(t *testing.T) {
	t.Run("yields collection names from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["alpha"] = 768
		store.collections["bravo"] = 384

		it, err := ListCollectionsSeq(context.Background(), store)

		require.NoError(t, err)
		require.NotNil(t, it)

		var names []string
		for name := range it {
			names = append(names, name)
		}
		assert.Len(t, names, 2)
		assert.Contains(t, names, "alpha")
		assert.Contains(t, names, "bravo")
	})

	t.Run("empty store yields nothing", func(t *testing.T) {
		store := newMockVectorStore()

		it, err := ListCollectionsSeq(context.Background(), store)

		require.NoError(t, err)

		count := 0
		for range it {
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("error from store returns nil iterator", func(t *testing.T) {
		store := newMockVectorStore()
		store.listErr = core.E("mock.collections.list", "connection lost", nil)

		it, err := ListCollectionsSeq(context.Background(), store)

		assert.Error(t, err)
		assert.Nil(t, it)
	})
}

// --- DeleteCollection tests ---

func TestCollections_DeleteCollection_Good(t *testing.T) {
	t.Run("deletes collection from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["to-delete"] = 768
		store.points["to-delete"] = []Point{{ID: "p1"}}

		err := DeleteCollection(context.Background(), store, "to-delete")

		require.NoError(t, err)
		_, exists := store.collections["to-delete"]
		assert.False(t, exists)
		assert.Empty(t, store.points["to-delete"])
	})

	t.Run("error from store propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.deleteErr = core.E("mock.collections.delete", "permission denied", nil)

		err := DeleteCollection(context.Background(), store, "any")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
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

		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, "my-col", info.Name)
		assert.Equal(t, uint64(3), info.PointCount)
		assert.Equal(t, uint64(768), info.VectorSize)
		assert.Equal(t, "green", info.Status)
	})

	t.Run("nonexistent collection returns error", func(t *testing.T) {
		store := newMockVectorStore()

		_, err := CollectionStats(context.Background(), store, "missing")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("error from store propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["err-col"] = 768
		store.infoErr = core.E("mock.collections.info", "internal error", nil)

		_, err := CollectionStats(context.Background(), store, "err-col")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "internal error")
	})

	t.Run("empty collection has zero point count", func(t *testing.T) {
		store := newMockVectorStore()
		store.collections["empty-col"] = 384

		info, err := CollectionStats(context.Background(), store, "empty-col")

		require.NoError(t, err)
		assert.Equal(t, uint64(0), info.PointCount)
		assert.Equal(t, uint64(384), info.VectorSize)
	})
}
