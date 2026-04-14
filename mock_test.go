package rag

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// mockEmbedder is a test-only Embedder that returns deterministic vectors.
// It tracks all calls for verification in tests.
type mockEmbedder struct {
	mu         sync.Mutex
	dimension  uint64
	embedCalls []string   // texts passed to Embed
	batchCalls [][]string // text slices passed to EmbedBatch
	embedErr   error      // if set, Embed returns this error
	batchErr   error      // if set, EmbedBatch returns this error

	// embedFunc allows per-test custom embedding behaviour.
	// If nil, the default deterministic vector is returned.
	embedFunc func(text string) ([]float32, error)
}

func newMockEmbedder(dimension uint64) *mockEmbedder {
	return &mockEmbedder{dimension: dimension}
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	m.mu.Lock()
	m.embedCalls = append(m.embedCalls, text)
	m.mu.Unlock()

	if m.embedErr != nil {
		return nil, m.embedErr
	}
	if m.embedFunc != nil {
		return m.embedFunc(text)
	}

	// Return a deterministic vector: all 0.1 values of the configured dimension.
	vec := make([]float32, m.dimension)
	for i := range vec {
		vec[i] = 0.1
	}
	return vec, nil
}

func (m *mockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	m.mu.Lock()
	m.batchCalls = append(m.batchCalls, texts)
	m.mu.Unlock()

	if m.batchErr != nil {
		return nil, m.batchErr
	}

	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := m.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = vec
	}
	return results, nil
}

func (m *mockEmbedder) EmbedDimension() uint64 {
	return m.dimension
}

// embedCallCount returns the number of times Embed was called.
func (m *mockEmbedder) embedCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.embedCalls)
}

// --- mockVectorStore ---

// mockVectorStore is a test-only VectorStore backed by in-memory maps.
// It tracks all calls for verification in tests.
type mockVectorStore struct {
	mu          sync.Mutex
	collections map[string]uint64  // collection name -> vector size
	points      map[string][]Point // collection name -> stored points
	searchFunc  func(collection string, vector []float32, limit uint64, filter map[string]string) ([]SearchResult, error)

	// Call tracking
	createCalls []createCollectionCall
	existsCalls []string
	deleteCalls []string
	listCalls   int
	infoCalls   []string
	upsertCalls []upsertCall
	searchCalls []searchCall

	// Error injection
	createErr error
	existsErr error
	deleteErr error
	listErr   error
	infoErr   error
	upsertErr error
	searchErr error
}

type createCollectionCall struct {
	Name       string
	VectorSize uint64
}

type upsertCall struct {
	Collection string
	Points     []Point
}

type searchCall struct {
	Collection string
	Vector     []float32
	Limit      uint64
	Filter     map[string]string
}

func newMockVectorStore() *mockVectorStore {
	return &mockVectorStore{
		collections: make(map[string]uint64),
		points:      make(map[string][]Point),
	}
}

func (m *mockVectorStore) CreateCollection(ctx context.Context, name string, vectorSize uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createCalls = append(m.createCalls, createCollectionCall{Name: name, VectorSize: vectorSize})

	if m.createErr != nil {
		return m.createErr
	}

	m.collections[name] = vectorSize
	return nil
}

func (m *mockVectorStore) CollectionExists(ctx context.Context, name string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.existsCalls = append(m.existsCalls, name)

	if m.existsErr != nil {
		return false, m.existsErr
	}

	_, exists := m.collections[name]
	return exists, nil
}

func (m *mockVectorStore) DeleteCollection(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls = append(m.deleteCalls, name)

	if m.deleteErr != nil {
		return m.deleteErr
	}

	delete(m.collections, name)
	delete(m.points, name)
	return nil
}

func (m *mockVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.listCalls++

	if m.listErr != nil {
		return nil, m.listErr
	}

	names := make([]string, 0, len(m.collections))
	for name := range m.collections {
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}

func (m *mockVectorStore) CollectionInfo(ctx context.Context, name string) (*CollectionInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.infoCalls = append(m.infoCalls, name)

	if m.infoErr != nil {
		return nil, m.infoErr
	}

	vectorSize, exists := m.collections[name]
	if !exists {
		return nil, fmt.Errorf("collection %q not found", name)
	}

	pointCount := uint64(len(m.points[name]))

	return &CollectionInfo{
		Name:       name,
		PointCount: pointCount,
		VectorSize: vectorSize,
		Status:     "green",
		Count:      pointCount,
		Vectors:    pointCount,
		Index:      "hnsw",
	}, nil
}

func (m *mockVectorStore) UpsertPoints(ctx context.Context, collection string, points []Point) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.upsertCalls = append(m.upsertCalls, upsertCall{Collection: collection, Points: points})

	if m.upsertErr != nil {
		return m.upsertErr
	}

	m.points[collection] = append(m.points[collection], points...)
	return nil
}

func (m *mockVectorStore) Add(ctx context.Context, collection string, vectors []Vector) error {
	points := make([]Point, len(vectors))
	for i, v := range vectors {
		points[i] = Point{
			ID:      v.ID,
			Vector:  v.Values,
			Payload: v.Payload,
		}
	}
	return m.UpsertPoints(ctx, collection, points)
}

func (m *mockVectorStore) Search(ctx context.Context, collection string, vector []float32, limit uint64, filter ...map[string]string) ([]SearchResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mergedFilter := mergeStringFilters(filter...)
	m.searchCalls = append(m.searchCalls, searchCall{
		Collection: collection,
		Vector:     vector,
		Limit:      limit,
		Filter:     mergedFilter,
	})

	if m.searchErr != nil {
		return nil, m.searchErr
	}

	if m.searchFunc != nil {
		return m.searchFunc(collection, vector, limit, mergedFilter)
	}

	// Default: return stored points as search results, sorted by a fake
	// descending score (1.0, 0.9, 0.8, ...), limited to `limit`.
	stored := m.points[collection]
	var results []SearchResult

	for i, p := range stored {
		// Apply filter if provided
		if len(mergedFilter) > 0 {
			match := true
			for k, v := range mergedFilter {
				if pv, ok := p.Payload[k]; !ok || fmt.Sprintf("%v", pv) != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		results = append(results, SearchResult{
			ID:      p.ID,
			Score:   1.0 - float32(i)*0.1,
			Payload: p.Payload,
		})
	}

	// Sort by score descending
	slices.SortFunc(results, func(a, b SearchResult) int {
		if a.Score > b.Score {
			return -1
		} else if a.Score < b.Score {
			return 1
		}
		return 0
	})

	// Apply limit
	if uint64(len(results)) > limit {
		results = results[:limit]
	}

	return results, nil
}

// allPoints returns all points stored across all collections.
func (m *mockVectorStore) allPoints(collection string) []Point {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.points[collection]
}

// upsertCallCount returns the total number of upsert calls.
func (m *mockVectorStore) upsertCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.upsertCalls)
}

// searchCallCount returns the total number of search calls.
func (m *mockVectorStore) searchCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.searchCalls)
}
