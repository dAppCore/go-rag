// Package rag provides RAG (Retrieval Augmented Generation) functionality
// for storing and querying documentation in Qdrant vector database.
package rag

import (
	"context"
	"strconv"

	"dappco.re/go/core"
	"github.com/qdrant/go-client/qdrant"
)

// QdrantConfig holds Qdrant connection configuration.
// cfg := QdrantConfig{Host: "localhost", Port: 6334, UseTLS: false}
type QdrantConfig struct {
	Host   string
	Port   int
	APIKey string
	UseTLS bool
}

// DefaultQdrantConfig returns default Qdrant configuration.
// Host defaults to localhost for local development.
// cfg := DefaultQdrantConfig()
func DefaultQdrantConfig() QdrantConfig {
	return QdrantConfig{
		Host:   "localhost",
		Port:   6334, // gRPC port
		UseTLS: false,
	}
}

// NewQdrantStore creates a Qdrant client from a base endpoint URL string.
// The convenience form accepts the common Qdrant REST endpoint example
// (http://localhost:6333) and normalizes it to the gRPC port used by the
// Go client.
func NewQdrantStore(endpoint string) (*QdrantClient, error) {
	cfg, err := qdrantConfigFromEndpoint(endpoint)
	if err != nil {
		return nil, core.E("rag.NewQdrantStore", "invalid Qdrant endpoint", err)
	}
	cfg.Port = normalizeQdrantGRPCPort(cfg.Port)
	return NewQdrantClient(cfg)
}

func normalizeQdrantGRPCPort(port int) int {
	if port == 6333 {
		return 6334
	}
	return port
}

func qdrantConfigFromEndpoint(endpoint string) (QdrantConfig, error) {
	cfg := DefaultQdrantConfig()
	parsed, err := parseEndpointURL(endpoint)
	if err != nil {
		return QdrantConfig{}, err
	}

	host := parsed.Hostname()
	if host == "" {
		host = cfg.Host
	}
	cfg.Host = host

	if portText := parsed.Port(); portText != "" {
		if port, err := strconv.Atoi(portText); err == nil {
			cfg.Port = port
		}
	}

	switch parsed.Scheme {
	case "https":
		cfg.UseTLS = true
	case "http":
		cfg.UseTLS = false
	}

	return cfg, nil
}

// QdrantClient wraps the Qdrant Go client with convenience methods.
// client, _ := NewQdrantClient(DefaultQdrantConfig())
type QdrantClient struct {
	client *qdrant.Client
	config QdrantConfig
}

// NewQdrantClient creates a new Qdrant client.
// client, err := NewQdrantClient(DefaultQdrantConfig())
func NewQdrantClient(cfg QdrantConfig) (*QdrantClient, error) {
	addr := core.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   cfg.Host,
		Port:   cfg.Port,
		APIKey: cfg.APIKey,
		UseTLS: cfg.UseTLS,
	})
	if err != nil {
		return nil, core.E("rag.Qdrant", core.Sprintf("failed to connect to Qdrant at %s", addr), err)
	}

	return &QdrantClient{
		client: client,
		config: cfg,
	}, nil
}

// Close closes the Qdrant client connection.
// defer client.Close()
func (q *QdrantClient) Close() error {
	return q.client.Close()
}

// HealthCheck verifies the connection to Qdrant.
// client.HealthCheck(ctx)
func (q *QdrantClient) HealthCheck(ctx context.Context) error {
	_, err := q.client.HealthCheck(ctx)
	return err
}

// ListCollections returns all collection names.
// names, _ := client.ListCollections(ctx)
func (q *QdrantClient) ListCollections(ctx context.Context) ([]string, error) {
	resp, err := q.client.ListCollections(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(resp))
	copy(names, resp)
	return names, nil
}

// CollectionExists checks if a collection exists.
// exists, _ := client.CollectionExists(ctx, "project-docs")
func (q *QdrantClient) CollectionExists(ctx context.Context, name string) (bool, error) {
	return q.client.CollectionExists(ctx, name)
}

// CreateCollection creates a new collection with cosine distance.
// client.CreateCollection(ctx, "project-docs", 768)
func (q *QdrantClient) CreateCollection(ctx context.Context, name string, vectorSize uint64) error {
	return q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})
}

// DeleteCollection deletes a collection.
// client.DeleteCollection(ctx, "project-docs")
func (q *QdrantClient) DeleteCollection(ctx context.Context, name string) error {
	return q.client.DeleteCollection(ctx, name)
}

// CollectionInfo returns backend-agnostic metadata about a collection.
// info, _ := client.CollectionInfo(ctx, "project-docs")
func (q *QdrantClient) CollectionInfo(ctx context.Context, name string) (*CollectionInfo, error) {
	info, err := q.client.GetCollectionInfo(ctx, name)
	if err != nil {
		return nil, err
	}

	pointCount := info.GetPointsCount()
	ci := &CollectionInfo{
		Name:       name,
		PointCount: pointCount,
		VectorSize: 0,
		Status:     "unknown",
	}

	// Extract vector size + index type from the Qdrant config
	if cfg := info.GetConfig(); cfg != nil {
		if params := cfg.GetParams(); params != nil {
			if vectorsCfg := params.GetVectorsConfig(); vectorsCfg != nil {
				if vecParams := vectorsCfg.GetParams(); vecParams != nil {
					ci.VectorSize = vecParams.GetSize()
				}
			}
		}
	}

	// Map Qdrant status to a simple string
	switch info.GetStatus() {
	case qdrant.CollectionStatus_Green:
		ci.Status = "green"
	case qdrant.CollectionStatus_Yellow:
		ci.Status = "yellow"
	case qdrant.CollectionStatus_Red:
		ci.Status = "red"
	default:
		ci.Status = "unknown"
	}

	return ci, nil
}

// Point represents a vector point with payload.
// point := Point{ID: "chunk-1", Vector: []float32{0.1, 0.2}, Payload: map[string]any{"source": "docs/go.md"}}
type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]any
}

// UpsertPoints inserts or updates points in a collection.
// client.UpsertPoints(ctx, "project-docs", points)
func (q *QdrantClient) UpsertPoints(ctx context.Context, collection string, points []Point) error {
	if len(points) == 0 {
		return nil
	}

	qdrantPoints := make([]*qdrant.PointStruct, len(points))
	for i, p := range points {
		qdrantPoints[i] = &qdrant.PointStruct{
			Id:      qdrant.NewID(p.ID),
			Vectors: qdrant.NewVectors(p.Vector...),
			Payload: qdrant.NewValueMap(p.Payload),
		}
	}

	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Points:         qdrantPoints,
	})
	return err
}

// SearchResult represents a low-level vector search hit.
// result := SearchResult{ID: "chunk-1", Score: 0.92, Payload: map[string]any{"text": "..."}}
type SearchResult struct {
	ID      string
	Score   float32
	Payload map[string]any
}

// GetText returns the text field from Payload (satisfies textResult / rankedResult).
func (r SearchResult) GetText() string {
	if r.Payload == nil {
		return ""
	}
	if v, ok := r.Payload["text"].(string); ok {
		return v
	}
	return ""
}

// GetScore returns the similarity score.
func (r SearchResult) GetScore() float32 { return r.Score }

// GetSource returns the source field from Payload, if present.
func (r SearchResult) GetSource() string {
	if r.Payload == nil {
		return ""
	}
	if v, ok := r.Payload["source"].(string); ok {
		return v
	}
	return ""
}

// GetChunkIndex returns the chunk_index field from Payload, if present.
func (r SearchResult) GetChunkIndex() int {
	if r.Payload == nil {
		return 0
	}
	switch v := r.Payload["chunk_index"].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

// GetSection returns the section field from Payload, if present.
func (r SearchResult) GetSection() string {
	if r.Payload == nil {
		return ""
	}
	if v, ok := r.Payload["section"].(string); ok {
		return v
	}
	return ""
}

// GetCategory returns the category field from Payload, if present.
func (r SearchResult) GetCategory() string {
	if r.Payload == nil {
		return ""
	}
	if v, ok := r.Payload["category"].(string); ok {
		return v
	}
	return ""
}

// Search performs a vector similarity search.
// results, _ := client.Search(ctx, "project-docs", vector, 5)
// results, _ := client.Search(ctx, "project-docs", vector, 5, map[string]string{"source": "docs"})
func (q *QdrantClient) Search(ctx context.Context, collection string, vector []float32, limit uint64, filter ...map[string]string) ([]SearchResult, error) {
	query := &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	}

	if len(filter) > 0 && len(filter[0]) > 0 {
		filterMap := filter[0]
		conditions := make([]*qdrant.Condition, 0, len(filterMap))
		for k, v := range filterMap {
			conditions = append(conditions, qdrant.NewMatch(k, v))
		}
		query.Filter = &qdrant.Filter{
			Must: conditions,
		}
	}

	resp, err := q.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(resp))
	for i, p := range resp {
		payload := make(map[string]any)
		for k, v := range p.Payload {
			payload[k] = valueToGo(v)
		}
		results[i] = SearchResult{
			ID:      pointIDToString(p.Id),
			Score:   p.Score,
			Payload: payload,
		}
	}
	return results, nil
}

// pointIDToString converts a Qdrant point ID to string.
func pointIDToString(id *qdrant.PointId) string {
	if id == nil {
		return ""
	}
	switch v := id.PointIdOptions.(type) {
	case *qdrant.PointId_Num:
		return core.Sprintf("%d", v.Num)
	case *qdrant.PointId_Uuid:
		return v.Uuid
	default:
		return ""
	}
}

// Add inserts RFC-shaped Vector payloads into the collection.
// It wraps UpsertPoints so callers can pass []Vector directly from ingestion pipelines.
//
//	err := client.Add(ctx, "docs", []Vector{{ID: "1", Values: embedding, Payload: meta}})
func (q *QdrantClient) Add(ctx context.Context, collection string, vectors []Vector) error {
	if len(vectors) == 0 {
		return nil
	}
	points := make([]Point, len(vectors))
	for i, v := range vectors {
		points[i] = Point{
			ID:      v.ID,
			Vector:  v.Values,
			Payload: v.Payload,
		}
	}
	return q.UpsertPoints(ctx, collection, points)
}

// valueToGo converts a Qdrant value to a Go value.
func valueToGo(v *qdrant.Value) any {
	if v == nil {
		return nil
	}
	switch val := v.Kind.(type) {
	case *qdrant.Value_StringValue:
		return val.StringValue
	case *qdrant.Value_IntegerValue:
		return val.IntegerValue
	case *qdrant.Value_DoubleValue:
		return val.DoubleValue
	case *qdrant.Value_BoolValue:
		return val.BoolValue
	case *qdrant.Value_ListValue:
		list := make([]any, len(val.ListValue.Values))
		for i, item := range val.ListValue.Values {
			list[i] = valueToGo(item)
		}
		return list
	case *qdrant.Value_StructValue:
		m := make(map[string]any)
		for k, item := range val.StructValue.Fields {
			m[k] = valueToGo(item)
		}
		return m
	default:
		return nil
	}
}
