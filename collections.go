package rag

import (
	"context"
	"iter"
	"slices"
)

// ListCollections returns all collection names from the given vector store.
func ListCollections(ctx context.Context, store VectorStore) ([]string, error) {
	return store.ListCollections(ctx)
}

// ListCollectionsSeq returns an iterator that yields all collection names from the given vector store.
func ListCollectionsSeq(ctx context.Context, store VectorStore) (iter.Seq[string], error) {
	names, err := store.ListCollections(ctx)
	if err != nil {
		return nil, err
	}
	return slices.Values(names), nil
}

// DeleteCollection removes a collection from the given vector store.
func DeleteCollection(ctx context.Context, store VectorStore, name string) error {
	return store.DeleteCollection(ctx, name)
}

// CollectionStats returns backend-agnostic metadata about a collection.
func CollectionStats(ctx context.Context, store VectorStore, name string) (*CollectionInfo, error) {
	return store.CollectionInfo(ctx, name)
}
