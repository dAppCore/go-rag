package rag

import "context"

// ListCollections returns all collection names from the given vector store.
func ListCollections(ctx context.Context, store VectorStore) ([]string, error) {
	return store.ListCollections(ctx)
}

// DeleteCollection removes a collection from the given vector store.
func DeleteCollection(ctx context.Context, store VectorStore, name string) error {
	return store.DeleteCollection(ctx, name)
}

// CollectionStats returns backend-agnostic metadata about a collection.
func CollectionStats(ctx context.Context, store VectorStore, name string) (*CollectionInfo, error) {
	return store.CollectionInfo(ctx, name)
}
