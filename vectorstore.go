package rag

import "context"

// VectorStore defines the interface for vector storage and search.
// QdrantClient satisfies this interface.
type VectorStore interface {
	// CreateCollection creates a new vector collection with the given
	// name and vector dimensionality.
	CreateCollection(ctx context.Context, name string, vectorSize uint64) error

	// CollectionExists checks whether a collection with the given name exists.
	CollectionExists(ctx context.Context, name string) (bool, error)

	// DeleteCollection deletes the collection with the given name.
	DeleteCollection(ctx context.Context, name string) error

	// UpsertPoints inserts or updates points in the named collection.
	UpsertPoints(ctx context.Context, collection string, points []Point) error

	// Search performs a vector similarity search, returning up to limit results.
	// An optional filter map restricts results by payload field values.
	Search(ctx context.Context, collection string, vector []float32, limit uint64, filter map[string]string) ([]SearchResult, error)
}
