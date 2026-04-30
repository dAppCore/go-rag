// Package rag provides RAG (Retrieval Augmented Generation) functionality
// for storing and querying documentation in Qdrant vector database.
package rag

import (
	"context"
	"net/url"

	"dappco.re/go"
	"github.com/qdrant/go-client/qdrant"
)

// QdrantConfig holds Qdrant connection configuration.
// cfg := QdrantConfig{Host: "localhost", Port: 6334, UseTLS: false}
type QdrantConfig struct {
	// Host is the Qdrant server hostname.
	Host string
	// Port is the Qdrant gRPC port.
	Port int
	// APIKey is the optional Qdrant API key.
	APIKey string
	// UseTLS enables TLS when connecting to Qdrant.
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
// (http://localhost:6333) and normalises it to the gRPC port used by the
// Go client.
func NewQdrantStore(endpoint string) core.Result {
	cfgResult := qdrantConfigFromEndpoint(endpoint)
	if !cfgResult.OK {
		return core.Fail(core.E("rag.NewQdrantStore", "invalid Qdrant endpoint", core.NewError(cfgResult.Error())))
	}
	cfg := cfgResult.Value.(QdrantConfig)
	cfg.Port = normalizeQdrantGRPCPort(cfg.Port)
	return NewQdrantClient(cfg)
}

// normalizeQdrantGRPCPort maps Qdrant's common REST port to its gRPC port.
func normalizeQdrantGRPCPort(port int) int {
	if port == 6333 {
		return 6334
	}
	return port
}

// qdrantConfigFromEndpoint parses a host or URL into Qdrant connection settings.
func qdrantConfigFromEndpoint(endpoint string) core.Result {
	cfg := DefaultQdrantConfig()
	parsedResult := parseEndpointURL(endpoint)
	if !parsedResult.OK {
		return parsedResult
	}
	parsed := parsedResult.Value.(*url.URL)

	host := parsed.Hostname()
	if host == "" {
		host = cfg.Host
	}
	cfg.Host = host

	if portText := parsed.Port(); portText != "" {
		portResult := parseEndpointPort("rag.qdrantConfigFromEndpoint", portText)
		if !portResult.OK {
			return portResult
		}
		port := portResult.Value.(int)
		cfg.Port = port
	}

	switch parsed.Scheme {
	case "https":
		cfg.UseTLS = true
	case "http":
		cfg.UseTLS = false
	}

	return core.Ok(cfg)
}

// QdrantClient wraps the Qdrant Go client with convenience methods.
// client, _ := NewQdrantClient(DefaultQdrantConfig())
type QdrantClient struct {
	client qdrantClientAPI
	config QdrantConfig
}

type qdrantClientAPI interface {
	Close() error
	HealthCheck(ctx context.Context) (*qdrant.HealthCheckReply, error)
	ListCollections(ctx context.Context) ([]string, error)
	CollectionExists(ctx context.Context, name string) (bool, error)
	CreateCollection(ctx context.Context, request *qdrant.CreateCollection) error
	DeleteCollection(ctx context.Context, name string) error
	GetCollectionInfo(ctx context.Context, name string) (*qdrant.CollectionInfo, error)
	Upsert(ctx context.Context, request *qdrant.UpsertPoints) (*qdrant.UpdateResult, error)
	Query(ctx context.Context, request *qdrant.QueryPoints) ([]*qdrant.ScoredPoint, error)
}

func (q *QdrantClient) api() core.Result {
	if q == nil || q.client == nil {
		return core.Fail(core.E("rag.Qdrant", "client is not initialized", nil))
	}
	return core.Ok(q.client)
}

// NewQdrantClient creates a new Qdrant client.
// client, err := NewQdrantClient(DefaultQdrantConfig())
func NewQdrantClient(cfg QdrantConfig) core.Result {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return core.Fail(core.E("rag.Qdrant", core.Sprintf("port out of range: %d", cfg.Port), nil))
	}
	addr := core.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   cfg.Host,
		Port:                   cfg.Port,
		APIKey:                 cfg.APIKey,
		UseTLS:                 cfg.UseTLS,
		SkipCompatibilityCheck: true,
	})
	if err != nil {
		return core.Fail(core.E("rag.Qdrant", core.Sprintf("failed to connect to Qdrant at %s", addr), err))
	}

	return core.Ok(&QdrantClient{
		client: client,
		config: cfg,
	})
}

// Close closes the Qdrant client connection.
// defer client.Close()
func (q *QdrantClient) Close() core.Result {
	clientResult := q.api()
	if !clientResult.OK {
		return clientResult
	}
	client := clientResult.Value.(qdrantClientAPI)
	return core.ResultOf(nil, client.Close())
}

// HealthCheck verifies the connection to Qdrant.
// client.HealthCheck(ctx)
func (q *QdrantClient) HealthCheck(ctx context.Context) core.Result {
	clientResult := q.api()
	if !clientResult.OK {
		return clientResult
	}
	client := clientResult.Value.(qdrantClientAPI)
	_, err := client.HealthCheck(ctx)
	return core.ResultOf(nil, err)
}

// ListCollections returns all collection names.
// names, _ := client.ListCollections(ctx)
func (q *QdrantClient) ListCollections(ctx context.Context) core.Result {
	clientResult := q.api()
	if !clientResult.OK {
		return clientResult
	}
	client := clientResult.Value.(qdrantClientAPI)
	resp, err := client.ListCollections(ctx)
	if err != nil {
		return core.Fail(err)
	}
	names := make([]string, len(resp))
	copy(names, resp)
	return core.Ok(names)
}

// CollectionExists checks if a collection exists.
// exists, _ := client.CollectionExists(ctx, "project-docs")
func (q *QdrantClient) CollectionExists(ctx context.Context, name string) core.Result {
	clientResult := q.api()
	if !clientResult.OK {
		return clientResult
	}
	client := clientResult.Value.(qdrantClientAPI)
	return core.ResultOf(client.CollectionExists(ctx, name))
}

// CreateCollection creates a new collection with cosine distance.
// client.CreateCollection(ctx, "project-docs", 768)
func (q *QdrantClient) CreateCollection(ctx context.Context, name string, vectorSize uint64) core.Result {
	clientResult := q.api()
	if !clientResult.OK {
		return clientResult
	}
	client := clientResult.Value.(qdrantClientAPI)
	return core.ResultOf(nil, client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	}))
}

// DeleteCollection deletes a collection.
// client.DeleteCollection(ctx, "project-docs")
func (q *QdrantClient) DeleteCollection(ctx context.Context, name string) core.Result {
	clientResult := q.api()
	if !clientResult.OK {
		return clientResult
	}
	client := clientResult.Value.(qdrantClientAPI)
	return core.ResultOf(nil, client.DeleteCollection(ctx, name))
}

// CollectionInfo returns backend-agnostic metadata about a collection.
// info, _ := client.CollectionInfo(ctx, "project-docs")
func (q *QdrantClient) CollectionInfo(ctx context.Context, name string) core.Result {
	clientResult := q.api()
	if !clientResult.OK {
		return clientResult
	}
	client := clientResult.Value.(qdrantClientAPI)
	info, err := client.GetCollectionInfo(ctx, name)
	if err != nil {
		return core.Fail(err)
	}

	pointCount := info.GetPointsCount()
	ci := &CollectionInfo{
		Name:       name,
		Count:      pointCount,
		Vectors:    pointCount,
		PointCount: pointCount,
		VectorSize: 0,
		Index:      "unknown",
		Status:     "unknown",
	}

	// Extract vector size + index type from the Qdrant config
	if cfg := info.GetConfig(); cfg != nil {
		if params := cfg.GetParams(); params != nil {
			if vectorsCfg := params.GetVectorsConfig(); vectorsCfg != nil {
				if vecParams := vectorsCfg.GetParams(); vecParams != nil {
					ci.VectorSize = vecParams.GetSize()
				}
				ci.Index = "hnsw"
			}
		}
		if ci.Index == "unknown" && cfg.GetHnswConfig() != nil {
			ci.Index = "hnsw"
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

	return core.Ok(ci)
}

// Point represents a vector point with payload.
// point := Point{ID: "chunk-1", Vector: []float32{0.1, 0.2}, Payload: map[string]any{"source": "docs/go.md"}}
type Point struct {
	// ID is the stable vector-store identifier for the point.
	ID string
	// Vector is the embedding stored in the collection.
	Vector []float32
	// Payload stores source text and metadata alongside the vector.
	Payload map[string]any
}

// UpsertPoints inserts or updates points in a collection.
// client.UpsertPoints(ctx, "project-docs", points)
func (q *QdrantClient) UpsertPoints(ctx context.Context, collection string, points []Point) core.Result {
	if len(points) == 0 {
		return core.Ok(nil)
	}
	clientResult := q.api()
	if !clientResult.OK {
		return clientResult
	}
	client := clientResult.Value.(qdrantClientAPI)

	qdrantPoints := make([]*qdrant.PointStruct, len(points))
	for i, p := range points {
		qdrantPoints[i] = &qdrant.PointStruct{
			Id:      qdrant.NewID(p.ID),
			Vectors: qdrant.NewVectors(p.Vector...),
			Payload: qdrant.NewValueMap(p.Payload),
		}
	}

	_, err := client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Points:         qdrantPoints,
	})
	return core.ResultOf(nil, err)
}

// SearchResult represents a low-level vector search hit.
// result := SearchResult{ID: "chunk-1", Score: 0.92, Text: "...", Source: "docs.md"}
type SearchResult struct {
	// ID is the vector-store point identifier.
	ID string
	// Score is the similarity score returned by the vector store.
	Score float32
	// Text is the denormalised chunk text, when present.
	Text string
	// Source is the denormalised source path, when present.
	Source string
	// Section is the denormalised Markdown section, when present.
	Section string
	// Category is the denormalised document category, when present.
	Category string
	// ChunkIndex is the denormalised source chunk index.
	ChunkIndex int
	// Index is a compatibility alias for ChunkIndex.
	Index int
	// ChunkIndexPresent distinguishes an explicit zero chunk index from missing metadata.
	ChunkIndexPresent bool
	// IndexPresent distinguishes an explicit zero index from missing metadata.
	IndexPresent bool
	// Payload is the decoded raw vector-store payload.
	Payload map[string]any
}

// GetText returns the text field from Payload (satisfies textResult / rankedResult).
func (r SearchResult) GetText() string {
	if r.Text != "" {
		return r.Text
	}
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
	if r.Source != "" {
		return r.Source
	}
	if r.Payload == nil {
		return ""
	}
	if v, ok := r.Payload["source"].(string); ok {
		return v
	}
	return ""
}

// HasChunkIndex reports whether this result carries explicit chunk metadata.
func (r SearchResult) HasChunkIndex() bool {
	if r.ChunkIndexPresent || r.IndexPresent {
		return true
	}
	if (r.ChunkIndex != 0 && r.ChunkIndex != missingChunkIndex) || (r.Index != 0 && r.Index != missingChunkIndex) {
		return true
	}
	_, ok := payloadChunkIndex(r.Payload)
	return ok
}

// GetChunkIndex returns the explicit chunk index, or the payload value.
func (r SearchResult) GetChunkIndex() int {
	if r.ChunkIndexPresent || (r.ChunkIndex != 0 && r.ChunkIndex != missingChunkIndex) {
		return r.ChunkIndex
	}
	if r.IndexPresent || (r.Index != 0 && r.Index != missingChunkIndex) {
		return r.Index
	}
	if index, ok := payloadChunkIndex(r.Payload); ok {
		return index
	}
	return missingChunkIndex
}

// GetSection returns the section field from Payload, if present.
func (r SearchResult) GetSection() string {
	if r.Section != "" {
		return r.Section
	}
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
	if r.Category != "" {
		return r.Category
	}
	if r.Payload == nil {
		return ""
	}
	if v, ok := r.Payload["category"].(string); ok {
		return v
	}
	return ""
}

// Search performs a vector similarity search.
// results, _ := client.Search(ctx, "project-docs", vector, 5, nil)
// results, _ := client.Search(ctx, "project-docs", vector, 5, map[string]string{"source": "docs"})
func (q *QdrantClient) Search(ctx context.Context, collection string, vector []float32, limit uint64, filter map[string]string) core.Result {
	clientResult := q.api()
	if !clientResult.OK {
		return clientResult
	}
	client := clientResult.Value.(qdrantClientAPI)
	query := &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	}

	if len(filter) > 0 {
		conditions := make([]*qdrant.Condition, 0, len(filter))
		for k, v := range filter {
			conditions = append(conditions, qdrant.NewMatch(k, v))
		}
		query.Filter = &qdrant.Filter{
			Must: conditions,
		}
	}

	resp, err := client.Query(ctx, query)
	if err != nil {
		return core.Fail(err)
	}

	results := make([]SearchResult, len(resp))
	for i, p := range resp {
		payload := make(map[string]any)
		for k, v := range p.Payload {
			payload[k] = valueToGo(v)
		}
		text, _ := payload["text"].(string)
		source, _ := payload["source"].(string)
		section, _ := payload["section"].(string)
		category, _ := payload["category"].(string)
		chunkIndex, chunkIndexPresent := payloadChunkIndex(payload)
		results[i] = SearchResult{
			ID:                pointIDToString(p.Id),
			Score:             p.Score,
			Text:              text,
			Source:            source,
			Section:           section,
			Category:          category,
			ChunkIndex:        chunkIndex,
			Index:             chunkIndex,
			ChunkIndexPresent: chunkIndexPresent,
			IndexPresent:      chunkIndexPresent,
			Payload:           payload,
		}
	}
	return core.Ok(results)
}

// payloadChunkIndex extracts chunk_index from decoded Qdrant payload values.
func payloadChunkIndex(payload map[string]any) (int, bool) {
	if payload == nil {
		return 0, false
	}
	switch v := payload["chunk_index"].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
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
//	r := client.Add(ctx, "docs", []Vector{{ID: "1", Values: embedding, Payload: meta}})
func (q *QdrantClient) Add(ctx context.Context, collection string, vectors []Vector) core.Result {
	if len(vectors) == 0 {
		return core.Ok(nil)
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
