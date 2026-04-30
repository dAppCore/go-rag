package rag

import (
	"context"

	"dappco.re/go"
)

// VectorStore defines the interface for vector storage and search.
// var store VectorStore = qdrantClient
type VectorStore interface {
	// CreateCollection creates a new vector collection with the given
	// name and vector dimensionality.
	CreateCollection(ctx context.Context, name string, vectorSize uint64) core.Result

	// CollectionExists checks whether a collection with the given name exists.
	CollectionExists(ctx context.Context, name string) core.Result

	// DeleteCollection deletes the collection with the given name.
	DeleteCollection(ctx context.Context, name string) core.Result

	// ListCollections returns all collection names in the store.
	ListCollections(ctx context.Context) core.Result

	// CollectionInfo returns metadata about a collection. Implementations
	// should populate at least PointCount and VectorSize in the returned
	// CollectionInfo struct.
	CollectionInfo(ctx context.Context, name string) core.Result

	// UpsertPoints inserts or updates points in the named collection.
	UpsertPoints(ctx context.Context, collection string, points []Point) core.Result

	// Search performs a vector similarity search, returning up to limit results.
	// The filter map restricts results by payload field values when non-nil.
	Search(ctx context.Context, collection string, vector []float32, limit uint64, filter map[string]string) core.Result
}

// Vector represents an RFC-compatible vector payload for storage.
// It is equivalent to Point, but uses the Values field name from the spec.
type Vector struct {
	// ID is the stable vector identifier.
	ID string
	// Values is the embedding vector payload.
	Values []float32
	// Payload stores arbitrary metadata alongside the vector.
	Payload map[string]any
}

// CollectionInfo holds backend-agnostic metadata about a collection.
// info := CollectionInfo{Name: "project-docs", Count: 42, Vectors: 42, PointCount: 42, VectorSize: 768, Status: "green"}
type CollectionInfo struct {
	// Name is the collection name.
	Name string
	// Count is the backend-reported point count.
	Count uint64
	// Vectors is the backend-reported vector count.
	Vectors uint64
	// Index names the index implementation when known.
	Index string
	// PointCount is the number of stored points.
	PointCount uint64
	// VectorSize is the configured embedding dimension.
	VectorSize uint64
	// Status is the backend health state, e.g. "green", "yellow", "red", or "unknown".
	Status string
}
