package rag

import (
	"context"
	"slices"

	"dappco.re/go"
)

// ListCollections returns all collection names from the given vector store.
// ListCollections(ctx, store)
func ListCollections(ctx context.Context, store VectorStore) core.Result {
	return store.ListCollections(ctx)
}

// ListCollectionsSeq returns an iterator that yields all collection names from the given vector store.
// it, _ := ListCollectionsSeq(ctx, store)
func ListCollectionsSeq(ctx context.Context, store VectorStore) core.Result {
	namesResult := store.ListCollections(ctx)
	if !namesResult.OK {
		return namesResult
	}
	names := namesResult.Value.([]string)
	return core.Ok(slices.Values(names))
}

// DeleteCollection removes a collection from the given vector store.
// DeleteCollection(ctx, store, "project-docs")
func DeleteCollection(ctx context.Context, store VectorStore, name string) core.Result {
	return store.DeleteCollection(ctx, name)
}

// CollectionStats returns backend-agnostic metadata about a collection.
// info, _ := CollectionStats(ctx, store, "project-docs")
func CollectionStats(ctx context.Context, store VectorStore, name string) core.Result {
	return store.CollectionInfo(ctx, name)
}
