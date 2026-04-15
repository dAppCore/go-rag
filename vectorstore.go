package rag

import "context"

// VectorStore defines the interface for vector storage and search.
// var store VectorStore = qdrantClient
type VectorStore interface {
	// CreateCollection creates a new vector collection with the given
	// name and vector dimensionality.
	CreateCollection(ctx context.Context, name string, vectorSize uint64) error

	// CollectionExists checks whether a collection with the given name exists.
	CollectionExists(ctx context.Context, name string) (bool, error)

	// DeleteCollection deletes the collection with the given name.
	DeleteCollection(ctx context.Context, name string) error

	// ListCollections returns all collection names in the store.
	ListCollections(ctx context.Context) ([]string, error)

	// CollectionInfo returns metadata about a collection. Implementations
	// should populate at least PointCount and VectorSize in the returned
	// CollectionInfo struct.
	CollectionInfo(ctx context.Context, name string) (*CollectionInfo, error)

	// UpsertPoints inserts or updates points in the named collection.
	UpsertPoints(ctx context.Context, collection string, points []Point) error

	// Search performs a vector similarity search, returning up to limit results.
	// The optional filter map restricts results by payload field values.
	Search(ctx context.Context, collection string, vector []float32, limit uint64, filter map[string]string) ([]SearchResult, error)
}

// Vector represents an RFC-compatible vector payload for storage.
// It is equivalent to Point, but uses the Values field name from the spec.
type Vector struct {
	ID      string
	Values  []float32
	Payload map[string]any
}

// CollectionInfo holds backend-agnostic metadata about a collection.
// info := CollectionInfo{Name: "project-docs", PointCount: 42, VectorSize: 768, Count: 42, Vectors: 42, Index: "hnsw"}
type CollectionInfo struct {
	Name       string
	PointCount uint64
	VectorSize uint64
	Status     string // e.g. "green", "yellow", "red", "unknown"
	Count      uint64 // RFC alias for PointCount
	Vectors    uint64 // RFC: vector count in collection
	Index      string // RFC: index type (e.g. "hnsw")
}
