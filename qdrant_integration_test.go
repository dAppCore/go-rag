//go:build rag

package rag

import (
	"context"
	"crypto/md5"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCollectionName returns a unique collection name for the current test run
// to avoid conflicts between parallel runs.
func testCollectionName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test-rag-%d", time.Now().UnixNano())
}

// testPointID generates a Qdrant-compatible point ID (32-char hex hash) from
// an arbitrary label string. Qdrant's NewID() wraps the value as a UUID field,
// and Qdrant validates it — MD5 hex strings are accepted, arbitrary strings
// are not.
func testPointID(label string) string {
	h := md5.Sum([]byte(label))
	return fmt.Sprintf("%x", h)
}

// skipIfQdrantUnavailable skips the test if Qdrant is not reachable on the
// default gRPC port.
func skipIfQdrantUnavailable(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "localhost:6334", 2*time.Second)
	if err != nil {
		t.Skip("Qdrant not available on localhost:6334 — skipping integration test")
	}
	_ = conn.Close()
}

func TestQdrantIntegration(t *testing.T) {
	skipIfQdrantUnavailable(t)

	cfg := DefaultQdrantConfig()
	client, err := NewQdrantClient(cfg)
	require.NoError(t, err, "failed to create Qdrant client")

	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()

	t.Run("health check succeeds", func(t *testing.T) {
		err := client.HealthCheck(ctx)
		require.NoError(t, err, "Qdrant health check should succeed")
	})

	t.Run("create collection and verify it exists", func(t *testing.T) {
		name := testCollectionName(t)
		t.Cleanup(func() {
			_ = client.DeleteCollection(ctx, name)
		})

		err := client.CreateCollection(ctx, name, 768)
		require.NoError(t, err, "creating collection should succeed")

		exists, err := client.CollectionExists(ctx, name)
		require.NoError(t, err)
		assert.True(t, exists, "collection should exist after creation")
	})

	t.Run("collection exists returns false for non-existent collection", func(t *testing.T) {
		exists, err := client.CollectionExists(ctx, "non-existent-collection-xyz-"+fmt.Sprint(time.Now().UnixNano()))
		require.NoError(t, err)
		assert.False(t, exists, "non-existent collection should return false")
	})

	t.Run("upsert points and search", func(t *testing.T) {
		name := testCollectionName(t)
		t.Cleanup(func() {
			_ = client.DeleteCollection(ctx, name)
		})

		// Create collection with small vector size for speed
		const vectorSize = 4
		err := client.CreateCollection(ctx, name, vectorSize)
		require.NoError(t, err)

		// Upsert two points with known vectors and payloads.
		// IDs must be valid hex hashes — Qdrant's UUID parser rejects
		// arbitrary strings.
		alphaID := testPointID("alpha")
		betaID := testPointID("beta")

		points := []Point{
			{
				ID:     alphaID,
				Vector: []float32{1.0, 0.0, 0.0, 0.0},
				Payload: map[string]any{
					"text":     "Alpha document about Go programming.",
					"source":   "alpha.md",
					"section":  "Introduction",
					"category": "documentation",
				},
			},
			{
				ID:     betaID,
				Vector: []float32{0.0, 1.0, 0.0, 0.0},
				Payload: map[string]any{
					"text":     "Beta document about Rust concurrency.",
					"source":   "beta.md",
					"section":  "Concurrency",
					"category": "documentation",
				},
			},
		}

		err = client.UpsertPoints(ctx, name, points)
		require.NoError(t, err, "upserting points should succeed")

		// Allow Qdrant a moment to index — not strictly required for small data
		// but avoids flaky results on slower machines.
		time.Sleep(500 * time.Millisecond)

		// Search with a vector close to the alpha point
		queryVector := []float32{0.9, 0.1, 0.0, 0.0}
		results, err := client.Search(ctx, name, queryVector, 5, nil)
		require.NoError(t, err, "search should succeed")
		require.NotEmpty(t, results, "search should return at least one result")

		// The top result should be closest to the alpha vector
		assert.Equal(t, "Alpha document about Go programming.", results[0].Payload["text"])
		assert.Equal(t, "alpha.md", results[0].Payload["source"])
		assert.Greater(t, results[0].Score, float32(0.0), "score should be positive")
	})

	t.Run("search with filter", func(t *testing.T) {
		name := testCollectionName(t)
		t.Cleanup(func() {
			_ = client.DeleteCollection(ctx, name)
		})

		const vectorSize = 4
		err := client.CreateCollection(ctx, name, vectorSize)
		require.NoError(t, err)

		points := []Point{
			{
				ID:     testPointID("filter-arch"),
				Vector: []float32{1.0, 0.0, 0.0, 0.0},
				Payload: map[string]any{
					"text":     "Architecture overview.",
					"source":   "arch.md",
					"category": "architecture",
				},
			},
			{
				ID:     testPointID("filter-help"),
				Vector: []float32{0.9, 0.1, 0.0, 0.0},
				Payload: map[string]any{
					"text":     "Help document.",
					"source":   "help.md",
					"category": "help-doc",
				},
			},
		}

		err = client.UpsertPoints(ctx, name, points)
		require.NoError(t, err)
		time.Sleep(500 * time.Millisecond)

		// Search with filter for "architecture" category only
		filter := map[string]string{"category": "architecture"}
		results, err := client.Search(ctx, name, []float32{1.0, 0.0, 0.0, 0.0}, 5, filter)
		require.NoError(t, err)
		require.Len(t, results, 1, "filter should return only the architecture document")
		assert.Equal(t, "Architecture overview.", results[0].Payload["text"])
	})

	t.Run("upsert empty points is a no-op", func(t *testing.T) {
		name := testCollectionName(t)
		t.Cleanup(func() {
			_ = client.DeleteCollection(ctx, name)
		})

		err := client.CreateCollection(ctx, name, 4)
		require.NoError(t, err)

		// Upserting empty slice should not error
		err = client.UpsertPoints(ctx, name, []Point{})
		require.NoError(t, err)
	})

	t.Run("delete collection and verify it no longer exists", func(t *testing.T) {
		name := testCollectionName(t)

		err := client.CreateCollection(ctx, name, 128)
		require.NoError(t, err)

		exists, err := client.CollectionExists(ctx, name)
		require.NoError(t, err)
		require.True(t, exists)

		err = client.DeleteCollection(ctx, name)
		require.NoError(t, err, "deleting collection should succeed")

		exists, err = client.CollectionExists(ctx, name)
		require.NoError(t, err)
		assert.False(t, exists, "collection should not exist after deletion")
	})

	t.Run("list collections includes created collection", func(t *testing.T) {
		name := testCollectionName(t)
		t.Cleanup(func() {
			_ = client.DeleteCollection(ctx, name)
		})

		err := client.CreateCollection(ctx, name, 64)
		require.NoError(t, err)

		collections, err := client.ListCollections(ctx)
		require.NoError(t, err)
		assert.Contains(t, collections, name, "list should include the newly created collection")
	})

	t.Run("collection info returns valid data", func(t *testing.T) {
		name := testCollectionName(t)
		t.Cleanup(func() {
			_ = client.DeleteCollection(ctx, name)
		})

		err := client.CreateCollection(ctx, name, 256)
		require.NoError(t, err)

		info, err := client.CollectionInfo(ctx, name)
		require.NoError(t, err)
		require.NotNil(t, info, "collection info should not be nil")
	})

	t.Run("search returns results with valid IDs", func(t *testing.T) {
		name := testCollectionName(t)
		t.Cleanup(func() {
			_ = client.DeleteCollection(ctx, name)
		})

		const vectorSize = 4
		err := client.CreateCollection(ctx, name, vectorSize)
		require.NoError(t, err)

		pointID := testPointID("uuid-check")
		points := []Point{
			{
				ID:      pointID,
				Vector:  []float32{0.5, 0.5, 0.0, 0.0},
				Payload: map[string]any{"text": "Test point."},
			},
		}
		err = client.UpsertPoints(ctx, name, points)
		require.NoError(t, err)
		time.Sleep(500 * time.Millisecond)

		results, err := client.Search(ctx, name, []float32{0.5, 0.5, 0.0, 0.0}, 1, nil)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.NotEmpty(t, results[0].ID, "result ID should not be empty")
	})

	t.Run("upsert overwrites existing point", func(t *testing.T) {
		name := testCollectionName(t)
		t.Cleanup(func() {
			_ = client.DeleteCollection(ctx, name)
		})

		const vectorSize = 4
		err := client.CreateCollection(ctx, name, vectorSize)
		require.NoError(t, err)

		id := testPointID("upsert-overwrite")

		// Insert original point
		original := []Point{
			{
				ID:      id,
				Vector:  []float32{1.0, 0.0, 0.0, 0.0},
				Payload: map[string]any{"text": "original content"},
			},
		}
		err = client.UpsertPoints(ctx, name, original)
		require.NoError(t, err)

		// Upsert same ID with different content
		updated := []Point{
			{
				ID:      id,
				Vector:  []float32{0.0, 1.0, 0.0, 0.0},
				Payload: map[string]any{"text": "updated content"},
			},
		}
		err = client.UpsertPoints(ctx, name, updated)
		require.NoError(t, err)
		time.Sleep(500 * time.Millisecond)

		// Search should find the updated content
		results, err := client.Search(ctx, name, []float32{0.0, 1.0, 0.0, 0.0}, 1, nil)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "updated content", results[0].Payload["text"],
			"upsert should overwrite the previous point payload")
	})
}
