// Package rag provides RAG (Retrieval Augmented Generation) functionality
// for storing and querying documentation in Qdrant vector database.
package rag

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"dappco.re/go/core/log"
	"github.com/qdrant/go-client/qdrant"
)

// QdrantConfig holds Qdrant connection configuration.
type QdrantConfig struct {
	Host   string
	Port   int
	APIKey string
	UseTLS bool
}

// DefaultQdrantConfig returns default Qdrant configuration.
// Host defaults to localhost for local development.
func DefaultQdrantConfig() QdrantConfig {
	return QdrantConfig{
		Host:   "localhost",
		Port:   6334, // gRPC port
		UseTLS: false,
	}
}

// QdrantClient wraps the Qdrant Go client with convenience methods.
type QdrantClient struct {
	client *qdrant.Client
	config QdrantConfig
}

// NewQdrantClient creates a new Qdrant client.
func NewQdrantClient(cfg QdrantConfig) (*QdrantClient, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   cfg.Host,
		Port:   cfg.Port,
		APIKey: cfg.APIKey,
		UseTLS: cfg.UseTLS,
	})
	if err != nil {
		return nil, log.E("rag.Qdrant", fmt.Sprintf("failed to connect to Qdrant at %s", addr), err)
	}

	return &QdrantClient{
		client: client,
		config: cfg,
	}, nil
}

// NewQdrantStore creates a Qdrant client from an endpoint string.
//
// The endpoint may be a bare host:port, http://host:port, or https://host:port.
// The scheme only affects TLS selection; the underlying client remains gRPC.
func NewQdrantStore(endpoint string) (*QdrantClient, error) {
	cfg, err := qdrantConfigFromEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	return NewQdrantClient(cfg)
}

func qdrantConfigFromEndpoint(endpoint string) (QdrantConfig, error) {
	if endpoint == "" {
		return DefaultQdrantConfig(), nil
	}

	parsed, err := parseEndpointURL(endpoint)
	if err != nil {
		return QdrantConfig{}, err
	}

	host := parsed.Hostname()
	if host == "" {
		host = "localhost"
	}

	port := 6334
	if p := parsed.Port(); p != "" {
		if parsedPort, err := strconv.Atoi(p); err == nil {
			port = parsedPort
		}
	}

	return QdrantConfig{
		Host:   host,
		Port:   port,
		UseTLS: parsed.Scheme == "https",
	}, nil
}

// Close closes the Qdrant client connection.
func (q *QdrantClient) Close() error {
	return q.client.Close()
}

// HealthCheck verifies the connection to Qdrant.
func (q *QdrantClient) HealthCheck(ctx context.Context) error {
	_, err := q.client.HealthCheck(ctx)
	return err
}

// ListCollections returns all collection names.
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
func (q *QdrantClient) CollectionExists(ctx context.Context, name string) (bool, error) {
	return q.client.CollectionExists(ctx, name)
}

// CreateCollection creates a new collection with cosine distance.
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
func (q *QdrantClient) DeleteCollection(ctx context.Context, name string) error {
	return q.client.DeleteCollection(ctx, name)
}

// CollectionInfo returns backend-agnostic metadata about a collection.
func (q *QdrantClient) CollectionInfo(ctx context.Context, name string) (*CollectionInfo, error) {
	info, err := q.client.GetCollectionInfo(ctx, name)
	if err != nil {
		return nil, err
	}

	pointCount := info.GetPointsCount()
	ci := &CollectionInfo{
		Name:       name,
		PointCount: pointCount,
		Count:      pointCount,
		Vectors:    info.GetIndexedVectorsCount(),
		Index:      "unknown",
	}

	// Extract vector size from the Qdrant config
	if cfg := info.GetConfig(); cfg != nil {
		if params := cfg.GetParams(); params != nil {
			if vectorsCfg := params.GetVectorsConfig(); vectorsCfg != nil {
				if vecParams := vectorsCfg.GetParams(); vecParams != nil {
					ci.VectorSize = vecParams.GetSize()
				}
			}
		}
		if cfg.GetHnswConfig() != nil {
			ci.Index = "hnsw"
		}
	}

	if ci.Vectors == 0 {
		ci.Vectors = pointCount
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
type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]any
}

// UpsertPoints inserts or updates points in a collection.
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

// SearchResult represents a search result with score.
type SearchResult struct {
	ID       string
	Score    float32
	Text     string
	Source   string
	Section  string
	Category string
	Index    int
	Payload  map[string]any
}

func (r SearchResult) GetText() string {
	return r.Text
}

func (r SearchResult) GetScore() float32 {
	return r.Score
}

func (r SearchResult) GetSource() string {
	return r.Source
}

func (r SearchResult) GetChunkIndex() int {
	return r.Index
}

// Add inserts or updates vectors in a collection using the RFC-compatible
// Vector representation.
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

// Search performs a vector similarity search.
func (q *QdrantClient) Search(ctx context.Context, collection string, vector []float32, limit uint64, filter ...map[string]string) ([]SearchResult, error) {
	query := &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	}

	// Add filter if provided
	mergedFilter := mergeStringFilters(filter...)
	if len(mergedFilter) > 0 {
		conditions := make([]*qdrant.Condition, 0, len(mergedFilter))
		for k, v := range mergedFilter {
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

		searchResult := SearchResult{
			ID:      pointIDToString(p.Id),
			Score:   p.Score,
			Payload: payload,
		}
		if text, ok := payload["text"].(string); ok {
			searchResult.Text = text
		}
		if source, ok := payload["source"].(string); ok {
			searchResult.Source = source
		}
		if section, ok := payload["section"].(string); ok {
			searchResult.Section = section
		}
		if category, ok := payload["category"].(string); ok {
			searchResult.Category = category
		}
		switch idx := payload["chunk_index"].(type) {
		case int:
			searchResult.Index = idx
		case int32:
			searchResult.Index = int(idx)
		case int64:
			searchResult.Index = int(idx)
		case float32:
			searchResult.Index = int(idx)
		case float64:
			searchResult.Index = int(idx)
		case uint64:
			searchResult.Index = int(idx)
		case string:
			if parsed, err := strconv.Atoi(idx); err == nil {
				searchResult.Index = parsed
			}
		}
		results[i] = searchResult
	}
	return results, nil
}

func mergeStringFilters(filters ...map[string]string) map[string]string {
	if len(filters) == 0 {
		return nil
	}

	merged := make(map[string]string)
	for _, filter := range filters {
		for k, v := range filter {
			merged[k] = v
		}
	}

	if len(merged) == 0 {
		return nil
	}
	return merged
}

func parseEndpointURL(endpoint string) (*url.URL, error) {
	raw := endpoint
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

// pointIDToString converts a Qdrant point ID to string.
func pointIDToString(id *qdrant.PointId) string {
	if id == nil {
		return ""
	}
	switch v := id.PointIdOptions.(type) {
	case *qdrant.PointId_Num:
		return fmt.Sprintf("%d", v.Num)
	case *qdrant.PointId_Uuid:
		return v.Uuid
	default:
		return ""
	}
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
