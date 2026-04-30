package rag

import (
	"context"
	"testing"

	core "dappco.re/go"
	"github.com/qdrant/go-client/qdrant"
)

// --- DefaultQdrantConfig tests ---

func TestQdrant_DefaultQdrantConfig_Good(t *testing.T) {
	t.Run("returns expected default values", func(t *testing.T) {
		cfg := DefaultQdrantConfig()

		assertEqual(t, "localhost", cfg.Host, "default host should be localhost")
		assertEqual(t, 6334, cfg.Port, "default gRPC port should be 6334")
		assertFalse(t, cfg.UseTLS, "TLS should be disabled by default")
		assertEmpty(t, cfg.APIKey, "API key should be empty by default")
	})
}

// --- normalizeQdrantGRPCPort tests ---

func TestQdrant_normalizeQdrantGRPCPort_Good(t *testing.T) {
	t.Run("maps REST port 6333 to gRPC port 6334", func(t *testing.T) {
		assertEqual(t, 6334, normalizeQdrantGRPCPort(6333))
	})

	t.Run("leaves other ports unchanged", func(t *testing.T) {
		assertEqual(t, 6334, normalizeQdrantGRPCPort(6334))
		assertEqual(t, 7000, normalizeQdrantGRPCPort(7000))
	})
}

// --- pointIDToString tests ---

func TestQdrant_pointIDToString_Good(t *testing.T) {
	t.Run("nil ID returns empty string", func(t *testing.T) {
		result := pointIDToString(nil)
		assertEqual(t, "", result)
	})

	t.Run("numeric ID returns string representation", func(t *testing.T) {
		id := qdrant.NewIDNum(42)
		result := pointIDToString(id)
		assertEqual(t, "42", result)
	})

	t.Run("numeric ID zero", func(t *testing.T) {
		id := qdrant.NewIDNum(0)
		result := pointIDToString(id)
		assertEqual(t, "0", result)
	})

	t.Run("large numeric ID", func(t *testing.T) {
		id := qdrant.NewIDNum(18446744073709551615) // max uint64
		result := pointIDToString(id)
		assertEqual(t, "18446744073709551615", result)
	})

	t.Run("UUID ID returns UUID string", func(t *testing.T) {
		uuid := "550e8400-e29b-41d4-a716-446655440000"
		id := qdrant.NewIDUUID(uuid)
		result := pointIDToString(id)
		assertEqual(t, uuid, result)
	})

	t.Run("empty UUID returns empty string", func(t *testing.T) {
		id := qdrant.NewIDUUID("")
		result := pointIDToString(id)
		assertEqual(t, "", result)
	})

	t.Run("string ID via NewID returns the UUID", func(t *testing.T) {
		// NewID creates a UUID-type PointId
		uuid := "abc-123-def"
		id := qdrant.NewID(uuid)
		result := pointIDToString(id)
		assertEqual(t, uuid, result)
	})
}

// --- valueToGo tests ---

func TestQdrant_valueToGo_Good(t *testing.T) {
	t.Run("nil value returns nil", func(t *testing.T) {
		result := valueToGo(nil)
		assertNil(t, result)
	})

	t.Run("string value", func(t *testing.T) {
		v := qdrant.NewValueString("hello world")
		result := valueToGo(v)
		assertEqual(t, "hello world", result)
	})

	t.Run("empty string value", func(t *testing.T) {
		v := qdrant.NewValueString("")
		result := valueToGo(v)
		assertEqual(t, "", result)
	})

	t.Run("integer value", func(t *testing.T) {
		v := qdrant.NewValueInt(42)
		result := valueToGo(v)
		assertEqual(t, int64(42), result)
	})

	t.Run("negative integer value", func(t *testing.T) {
		v := qdrant.NewValueInt(-100)
		result := valueToGo(v)
		assertEqual(t, int64(-100), result)
	})

	t.Run("zero integer value", func(t *testing.T) {
		v := qdrant.NewValueInt(0)
		result := valueToGo(v)
		assertEqual(t, int64(0), result)
	})

	t.Run("double value", func(t *testing.T) {
		v := qdrant.NewValueDouble(3.14)
		result := valueToGo(v)
		assertEqual(t, float64(3.14), result)
	})

	t.Run("negative double value", func(t *testing.T) {
		v := qdrant.NewValueDouble(-2.718)
		result := valueToGo(v)
		assertEqual(t, float64(-2.718), result)
	})

	t.Run("bool true value", func(t *testing.T) {
		v := qdrant.NewValueBool(true)
		result := valueToGo(v)
		assertEqual(t, true, result)
	})

	t.Run("bool false value", func(t *testing.T) {
		v := qdrant.NewValueBool(false)
		result := valueToGo(v)
		assertEqual(t, false, result)
	})

	t.Run("null value returns nil", func(t *testing.T) {
		v := qdrant.NewValueNull()
		result := valueToGo(v)
		// NullValue is not handled by the switch, falls to default -> nil
		assertNil(t, result)
	})

	t.Run("list value with mixed types", func(t *testing.T) {
		v := qdrant.NewValueFromList(
			qdrant.NewValueString("alpha"),
			qdrant.NewValueInt(99),
			qdrant.NewValueBool(true),
		)
		result := valueToGo(v)

		list, ok := result.([]any)
		assertTrue(t, ok, "result should be a []any")
		assertLen(t, list, 3)
		assertEqual(t, "alpha", list[0])
		assertEqual(t, int64(99), list[1])
		assertEqual(t, true, list[2])
	})

	t.Run("empty list value", func(t *testing.T) {
		v := qdrant.NewValueList(&qdrant.ListValue{Values: []*qdrant.Value{}})
		result := valueToGo(v)

		list, ok := result.([]any)
		assertTrue(t, ok, "result should be a []any")
		assertEmpty(t, list)
	})

	t.Run("struct value with fields", func(t *testing.T) {
		v := qdrant.NewValueStruct(&qdrant.Struct{
			Fields: map[string]*qdrant.Value{
				"name": qdrant.NewValueString("test"),
				"age":  qdrant.NewValueInt(25),
			},
		})
		result := valueToGo(v)

		m, ok := result.(map[string]any)
		assertTrue(t, ok, "result should be a map[string]any")
		assertEqual(t, "test", m["name"])
		assertEqual(t, int64(25), m["age"])
	})

	t.Run("empty struct value", func(t *testing.T) {
		v := qdrant.NewValueStruct(&qdrant.Struct{
			Fields: map[string]*qdrant.Value{},
		})
		result := valueToGo(v)

		m, ok := result.(map[string]any)
		assertTrue(t, ok, "result should be a map[string]any")
		assertEmpty(t, m)
	})

	t.Run("nested list within struct", func(t *testing.T) {
		v := qdrant.NewValueStruct(&qdrant.Struct{
			Fields: map[string]*qdrant.Value{
				"tags": qdrant.NewValueFromList(
					qdrant.NewValueString("go"),
					qdrant.NewValueString("rag"),
				),
			},
		})
		result := valueToGo(v)

		m, ok := result.(map[string]any)
		assertTrue(t, ok, "result should be a map[string]any")

		tags, ok := m["tags"].([]any)
		assertTrue(t, ok, "tags should be a []any")
		assertEqual(t, []any{"go", "rag"}, tags)
	})

	t.Run("nested struct within struct", func(t *testing.T) {
		inner := qdrant.NewValueStruct(&qdrant.Struct{
			Fields: map[string]*qdrant.Value{
				"key": qdrant.NewValueString("value"),
			},
		})
		v := qdrant.NewValueStruct(&qdrant.Struct{
			Fields: map[string]*qdrant.Value{
				"nested": inner,
			},
		})
		result := valueToGo(v)

		m, ok := result.(map[string]any)
		assertTrue(t, ok)
		nested, ok := m["nested"].(map[string]any)
		assertTrue(t, ok)
		assertEqual(t, "value", nested["key"])
	})
}

// --- Point struct tests ---

func TestQdrant_Point_Good(t *testing.T) {
	t.Run("Point holds ID vector and payload", func(t *testing.T) {
		p := Point{
			ID:     "test-id-123",
			Vector: []float32{0.1, 0.2, 0.3},
			Payload: map[string]any{
				"text":   "hello",
				"source": "test.md",
			},
		}

		assertEqual(t, "test-id-123", p.ID)
		assertLen(t, p.Vector, 3)
		assertEqual(t, float32(0.1), p.Vector[0])
		assertEqual(t, "hello", p.Payload["text"])
		assertEqual(t, "test.md", p.Payload["source"])
	})

	t.Run("Point with empty payload", func(t *testing.T) {
		p := Point{
			ID:      "empty",
			Vector:  []float32{},
			Payload: map[string]any{},
		}

		assertEqual(t, "empty", p.ID)
		assertEmpty(t, p.Vector)
		assertEmpty(t, p.Payload)
	})
}

func TestQdrant_Search_OptionalFilter_Good(t *testing.T) {
	store := newMockVectorStore()
	store.collections["docs"] = 768
	store.points["docs"] = []Point{
		{
			ID:     "p1",
			Vector: []float32{0.1},
			Payload: map[string]any{
				"text":        "alpha",
				"source":      "a.md",
				"chunk_index": 0,
			},
		},
		{
			ID:     "p2",
			Vector: []float32{0.2},
			Payload: map[string]any{
				"text":        "beta",
				"source":      "b.md",
				"chunk_index": 1,
			},
		},
	}

	t.Run("allows RFC-style call without filter", func(t *testing.T) {
		r := store.Search(context.Background(), "docs", []float32{0.1}, 5, nil)
		results := resultValue[[]SearchResult](t, r)
		assertLen(t, results, 2)
	})

	t.Run("still accepts an optional filter map", func(t *testing.T) {
		r := store.Search(context.Background(), "docs", []float32{0.1}, 5, map[string]string{"source": "a.md"})
		results := resultValue[[]SearchResult](t, r)
		assertLen(t, results, 1)
		assertEqual(t, "a.md", results[0].Payload["source"])
	})
}

// --- SearchResult struct tests ---

func TestQdrant_SearchResult_Good(t *testing.T) {
	t.Run("SearchResult holds all fields", func(t *testing.T) {
		sr := SearchResult{
			ID:    "result-1",
			Score: 0.95,
			Payload: map[string]any{
				"text":     "some text",
				"category": "docs",
			},
		}

		assertEqual(t, "result-1", sr.ID)
		assertEqual(t, float32(0.95), sr.Score)
		assertEqual(t, "some text", sr.Payload["text"])
	})

	t.Run("prefers denormalised fields when present", func(t *testing.T) {
		sr := SearchResult{
			ID:         "result-2",
			Score:      0.9,
			Text:       "denormalised text",
			Source:     "doc.md",
			Section:    "Intro",
			Category:   "docs",
			ChunkIndex: 7,
			Payload: map[string]any{
				"text":        "payload text",
				"source":      "payload.md",
				"section":     "Payload",
				"category":    "payload",
				"chunk_index": 42,
			},
		}

		assertEqual(t, "denormalised text", sr.GetText())
		assertEqual(t, "doc.md", sr.GetSource())
		assertEqual(t, "Intro", sr.GetSection())
		assertEqual(t, "docs", sr.GetCategory())
		assertEqual(t, 7, sr.GetChunkIndex())
	})

	t.Run("falls back to payload when denormalised fields are empty", func(t *testing.T) {
		sr := SearchResult{
			Payload: map[string]any{
				"text":        "payload text",
				"source":      "payload.md",
				"section":     "Payload",
				"category":    "payload",
				"chunk_index": int64(42),
			},
		}

		assertEqual(t, "payload text", sr.GetText())
		assertEqual(t, "payload.md", sr.GetSource())
		assertEqual(t, "Payload", sr.GetSection())
		assertEqual(t, "payload", sr.GetCategory())
		assertEqual(t, 42, sr.GetChunkIndex())
	})

	t.Run("explicit zero chunk index takes precedence over payload", func(t *testing.T) {
		sr := SearchResult{
			ChunkIndex:        0,
			ChunkIndexPresent: true,
			Payload: map[string]any{
				"chunk_index": 42,
			},
		}

		assertTrue(t, sr.HasChunkIndex())
		assertEqual(t, 0, sr.GetChunkIndex())
	})

	t.Run("missing chunk index uses sentinel", func(t *testing.T) {
		sr := SearchResult{}

		assertFalse(t, sr.HasChunkIndex())
		assertEqual(t, missingChunkIndex, sr.GetChunkIndex())
	})
}

type qdrantTestAPI struct {
	closeCalled bool
	healthErr   error
	closeErr    error

	collections []string
	listErr     error
	exists      bool
	existsErr   error
	createErr   error
	deleteErr   error
	info        *qdrant.CollectionInfo
	infoErr     error
	upsertErr   error
	queryErr    error

	created *qdrant.CreateCollection
	deleted string
	upsert  *qdrant.UpsertPoints
	query   *qdrant.QueryPoints
	results []*qdrant.ScoredPoint
}

func (f *qdrantTestAPI) Close() error {
	f.closeCalled = true
	return f.closeErr
}

func (f *qdrantTestAPI) HealthCheck(core.Context) (*qdrant.HealthCheckReply, error) {
	return &qdrant.HealthCheckReply{Title: "qdrant"}, f.healthErr
}

func (f *qdrantTestAPI) ListCollections(core.Context) ([]string, error) {
	return f.collections, f.listErr
}

func (f *qdrantTestAPI) CollectionExists(core.Context, string) (bool, error) {
	return f.exists, f.existsErr
}

func (f *qdrantTestAPI) CreateCollection(_ core.Context, request *qdrant.CreateCollection) error {
	f.created = request
	return f.createErr
}

func (f *qdrantTestAPI) DeleteCollection(_ core.Context, name string) error {
	f.deleted = name
	return f.deleteErr
}

func (f *qdrantTestAPI) GetCollectionInfo(core.Context, string) (*qdrant.CollectionInfo, error) {
	return f.info, f.infoErr
}

func (f *qdrantTestAPI) Upsert(_ core.Context, request *qdrant.UpsertPoints) (*qdrant.UpdateResult, error) {
	f.upsert = request
	return &qdrant.UpdateResult{}, f.upsertErr
}

func (f *qdrantTestAPI) Query(_ core.Context, request *qdrant.QueryPoints) ([]*qdrant.ScoredPoint, error) {
	f.query = request
	return f.results, f.queryErr
}

func qdrantTestCollectionInfo(points uint64, size uint64, status qdrant.CollectionStatus) *qdrant.CollectionInfo {
	return &qdrant.CollectionInfo{
		Status:      status,
		PointsCount: qdrant.PtrOf(points),
		Config: &qdrant.CollectionConfig{
			Params: &qdrant.CollectionParams{
				VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{Size: size}),
			},
		},
	}
}

func TestQdrant_DefaultQdrantConfig_Bad(t *core.T) {
	cfg := DefaultQdrantConfig()

	core.AssertNotEqual(t, "", cfg.Host)
	core.AssertNotEqual(t, 6333, cfg.Port)
}

func TestQdrant_DefaultQdrantConfig_Ugly(t *core.T) {
	cfg := DefaultQdrantConfig()
	cfg.Host = "mutated"

	core.AssertEqual(t, "localhost", DefaultQdrantConfig().Host)
	core.AssertEqual(t, "mutated", cfg.Host)
}

func TestQdrant_NewQdrantClient_Good(t *core.T) {
	r := NewQdrantClient(DefaultQdrantConfig())
	client := r.Value.(*QdrantClient)
	defer func() {
		if client != nil {
			_ = client.Close()
		}
	}()

	core.AssertTrue(t, r.OK)
	core.AssertNotNil(t, client)
}

func TestQdrant_NewQdrantClient_Bad(t *core.T) {
	r := NewQdrantClient(QdrantConfig{Host: "bad host", Port: -1})

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "port out of range")
}

func TestQdrant_NewQdrantClient_Ugly(t *core.T) {
	r := NewQdrantClient(QdrantConfig{Host: "localhost", Port: 6334, APIKey: "token", UseTLS: true})
	client := r.Value.(*QdrantClient)
	defer func() {
		if client != nil {
			_ = client.Close()
		}
	}()

	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, client.config.UseTLS)
}

func TestQdrant_NewQdrantStore_Good(t *core.T) {
	r := NewQdrantStore("http://localhost:6333")
	client := r.Value.(*QdrantClient)
	defer func() {
		if client != nil {
			_ = client.Close()
		}
	}()

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, 6334, client.config.Port)
}

func TestQdrant_NewQdrantStore_Bad(t *core.T) {
	r := NewQdrantStore("http://[::1")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "invalid Qdrant endpoint")
}

func TestQdrant_NewQdrantStore_Ugly(t *core.T) {
	r := NewQdrantStore("")
	client := r.Value.(*QdrantClient)
	defer func() {
		if client != nil {
			_ = client.Close()
		}
	}()

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "localhost", client.config.Host)
}

func TestQdrant_QdrantClient_Close_Good(t *core.T) {
	fake := &qdrantTestAPI{}
	r := (&QdrantClient{client: fake}).Close()

	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, fake.closeCalled)
}

func TestQdrant_QdrantClient_Close_Bad(t *core.T) {
	r := (&QdrantClient{}).Close()

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not initialized")
}

func TestQdrant_QdrantClient_Close_Ugly(t *core.T) {
	fake := &qdrantTestAPI{closeErr: core.NewError("close failed")}
	r := (&QdrantClient{client: fake}).Close()

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "close failed")
}

func TestQdrant_QdrantClient_HealthCheck_Good(t *core.T) {
	r := (&QdrantClient{client: &qdrantTestAPI{}}).HealthCheck(core.Background())

	core.AssertTrue(t, r.OK)
	core.AssertNil(t, r.Value)
}

func TestQdrant_QdrantClient_HealthCheck_Bad(t *core.T) {
	r := (&QdrantClient{}).HealthCheck(core.Background())

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not initialized")
}

func TestQdrant_QdrantClient_HealthCheck_Ugly(t *core.T) {
	r := (&QdrantClient{client: &qdrantTestAPI{healthErr: core.NewError("down")}}).HealthCheck(core.Background())

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "down")
}

func TestQdrant_QdrantClient_ListCollections_Good(t *core.T) {
	fake := &qdrantTestAPI{collections: []string{"alpha", "bravo"}}
	r := (&QdrantClient{client: fake}).ListCollections(core.Background())
	names := r.Value.([]string)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, []string{"alpha", "bravo"}, names)
}

func TestQdrant_QdrantClient_ListCollections_Bad(t *core.T) {
	r := (&QdrantClient{}).ListCollections(core.Background())

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not initialized")
}

func TestQdrant_QdrantClient_ListCollections_Ugly(t *core.T) {
	fake := &qdrantTestAPI{listErr: core.NewError("list failed")}
	r := (&QdrantClient{client: fake}).ListCollections(core.Background())

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "list failed")
}

func TestQdrant_QdrantClient_CollectionExists_Good(t *core.T) {
	r := (&QdrantClient{client: &qdrantTestAPI{exists: true}}).CollectionExists(core.Background(), "docs")
	exists := r.Value.(bool)

	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, exists)
}

func TestQdrant_QdrantClient_CollectionExists_Bad(t *core.T) {
	r := (&QdrantClient{}).CollectionExists(core.Background(), "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not initialized")
}

func TestQdrant_QdrantClient_CollectionExists_Ugly(t *core.T) {
	r := (&QdrantClient{client: &qdrantTestAPI{existsErr: core.NewError("exists failed")}}).CollectionExists(core.Background(), "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "exists failed")
}

func TestQdrant_QdrantClient_CreateCollection_Good(t *core.T) {
	fake := &qdrantTestAPI{}
	r := (&QdrantClient{client: fake}).CreateCollection(core.Background(), "docs", 768)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, uint64(768), fake.created.GetVectorsConfig().GetParams().GetSize())
}

func TestQdrant_QdrantClient_CreateCollection_Bad(t *core.T) {
	r := (&QdrantClient{}).CreateCollection(core.Background(), "docs", 768)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not initialized")
}

func TestQdrant_QdrantClient_CreateCollection_Ugly(t *core.T) {
	r := (&QdrantClient{client: &qdrantTestAPI{createErr: core.NewError("create failed")}}).CreateCollection(core.Background(), "", 0)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "create failed")
}

func TestQdrant_QdrantClient_DeleteCollection_Good(t *core.T) {
	fake := &qdrantTestAPI{}
	r := (&QdrantClient{client: fake}).DeleteCollection(core.Background(), "docs")

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "docs", fake.deleted)
}

func TestQdrant_QdrantClient_DeleteCollection_Bad(t *core.T) {
	r := (&QdrantClient{}).DeleteCollection(core.Background(), "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not initialized")
}

func TestQdrant_QdrantClient_DeleteCollection_Ugly(t *core.T) {
	r := (&QdrantClient{client: &qdrantTestAPI{deleteErr: core.NewError("delete failed")}}).DeleteCollection(core.Background(), "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "delete failed")
}

func TestQdrant_QdrantClient_CollectionInfo_Good(t *core.T) {
	r := (&QdrantClient{client: &qdrantTestAPI{info: qdrantTestCollectionInfo(3, 768, qdrant.CollectionStatus_Green)}}).CollectionInfo(core.Background(), "docs")
	info := r.Value.(*CollectionInfo)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, uint64(768), info.VectorSize)
}

func TestQdrant_QdrantClient_CollectionInfo_Bad(t *core.T) {
	r := (&QdrantClient{}).CollectionInfo(core.Background(), "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not initialized")
}

func TestQdrant_QdrantClient_CollectionInfo_Ugly(t *core.T) {
	r := (&QdrantClient{client: &qdrantTestAPI{infoErr: core.NewError("info failed")}}).CollectionInfo(core.Background(), "docs")

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "info failed")
}

func TestQdrant_QdrantClient_UpsertPoints_Good(t *core.T) {
	fake := &qdrantTestAPI{}
	r := (&QdrantClient{client: fake}).UpsertPoints(core.Background(), "docs", []Point{{ID: "id", Vector: []float32{0.1}, Payload: map[string]any{"text": "alpha"}}})

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "docs", fake.upsert.GetCollectionName())
}

func TestQdrant_QdrantClient_UpsertPoints_Bad(t *core.T) {
	r := (&QdrantClient{}).UpsertPoints(core.Background(), "docs", []Point{{ID: "id", Vector: []float32{0.1}}})

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not initialized")
}

func TestQdrant_QdrantClient_UpsertPoints_Ugly(t *core.T) {
	r := (&QdrantClient{}).UpsertPoints(core.Background(), "docs", nil)

	core.AssertTrue(t, r.OK)
	core.AssertNil(t, r.Value)
}

func TestQdrant_QdrantClient_Search_Good(t *core.T) {
	fake := &qdrantTestAPI{results: []*qdrant.ScoredPoint{{
		Id:    qdrant.NewID("point-1"),
		Score: 0.9,
		Payload: map[string]*qdrant.Value{
			"text":        qdrant.NewValueString("alpha"),
			"source":      qdrant.NewValueString("a.md"),
			"section":     qdrant.NewValueString("Intro"),
			"category":    qdrant.NewValueString("docs"),
			"chunk_index": qdrant.NewValueInt(2),
		},
	}}}
	r := (&QdrantClient{client: fake}).Search(core.Background(), "docs", []float32{0.1}, 5, map[string]string{"category": "docs"})
	results := r.Value.([]SearchResult)

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "alpha", results[0].Text)
}

func TestQdrant_QdrantClient_Search_Bad(t *core.T) {
	r := (&QdrantClient{}).Search(core.Background(), "docs", []float32{0.1}, 5, nil)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not initialized")
}

func TestQdrant_QdrantClient_Search_Ugly(t *core.T) {
	r := (&QdrantClient{client: &qdrantTestAPI{queryErr: core.NewError("query failed")}}).Search(core.Background(), "docs", nil, 0, nil)

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "query failed")
}

func TestQdrant_QdrantClient_Add_Good(t *core.T) {
	fake := &qdrantTestAPI{}
	r := (&QdrantClient{client: fake}).Add(core.Background(), "docs", []Vector{{ID: "id", Values: []float32{0.2}, Payload: map[string]any{"text": "alpha"}}})

	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "docs", fake.upsert.GetCollectionName())
}

func TestQdrant_QdrantClient_Add_Bad(t *core.T) {
	r := (&QdrantClient{}).Add(core.Background(), "docs", []Vector{{ID: "id", Values: []float32{0.2}}})

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not initialized")
}

func TestQdrant_QdrantClient_Add_Ugly(t *core.T) {
	r := (&QdrantClient{}).Add(core.Background(), "docs", nil)

	core.AssertTrue(t, r.OK)
	core.AssertNil(t, r.Value)
}

func TestQdrant_SearchResult_GetText_Good(t *core.T) {
	result := SearchResult{Text: "direct text", Payload: map[string]any{"text": "payload text"}}

	core.AssertEqual(t, "direct text", result.GetText())
	core.AssertNotEmpty(t, result.GetText())
}

func TestQdrant_SearchResult_GetText_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, "", result.GetText())
	core.AssertEmpty(t, result.GetText())
}

func TestQdrant_SearchResult_GetText_Ugly(t *core.T) {
	result := SearchResult{Payload: map[string]any{"text": "payload text"}}

	core.AssertEqual(t, "payload text", result.GetText())
	core.AssertContains(t, result.GetText(), "payload")
}

func TestQdrant_SearchResult_GetScore_Good(t *core.T) {
	result := SearchResult{Score: 0.6}

	core.AssertEqual(t, float32(0.6), result.GetScore())
	core.AssertGreater(t, result.GetScore(), float32(0))
}

func TestQdrant_SearchResult_GetScore_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, float32(0), result.GetScore())
	core.AssertFalse(t, result.GetScore() > 0)
}

func TestQdrant_SearchResult_GetScore_Ugly(t *core.T) {
	result := SearchResult{Score: -0.5}

	core.AssertEqual(t, float32(-0.5), result.GetScore())
	core.AssertLess(t, result.GetScore(), float32(0))
}

func TestQdrant_SearchResult_GetSource_Good(t *core.T) {
	result := SearchResult{Source: "direct.md", Payload: map[string]any{"source": "payload.md"}}

	core.AssertEqual(t, "direct.md", result.GetSource())
	core.AssertNotEmpty(t, result.GetSource())
}

func TestQdrant_SearchResult_GetSource_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, "", result.GetSource())
	core.AssertEmpty(t, result.GetSource())
}

func TestQdrant_SearchResult_GetSource_Ugly(t *core.T) {
	result := SearchResult{Payload: map[string]any{"source": "payload.md"}}

	core.AssertEqual(t, "payload.md", result.GetSource())
	core.AssertContains(t, result.GetSource(), "payload")
}

func TestQdrant_SearchResult_GetSection_Good(t *core.T) {
	result := SearchResult{Section: "Intro", Payload: map[string]any{"section": "Payload"}}

	core.AssertEqual(t, "Intro", result.GetSection())
	core.AssertNotEmpty(t, result.GetSection())
}

func TestQdrant_SearchResult_GetSection_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, "", result.GetSection())
	core.AssertEmpty(t, result.GetSection())
}

func TestQdrant_SearchResult_GetSection_Ugly(t *core.T) {
	result := SearchResult{Payload: map[string]any{"section": "Payload"}}

	core.AssertEqual(t, "Payload", result.GetSection())
	core.AssertContains(t, result.GetSection(), "Payload")
}

func TestQdrant_SearchResult_GetCategory_Good(t *core.T) {
	result := SearchResult{Category: "docs", Payload: map[string]any{"category": "payload"}}

	core.AssertEqual(t, "docs", result.GetCategory())
	core.AssertNotEmpty(t, result.GetCategory())
}

func TestQdrant_SearchResult_GetCategory_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, "", result.GetCategory())
	core.AssertEmpty(t, result.GetCategory())
}

func TestQdrant_SearchResult_GetCategory_Ugly(t *core.T) {
	result := SearchResult{Payload: map[string]any{"category": "payload"}}

	core.AssertEqual(t, "payload", result.GetCategory())
	core.AssertContains(t, result.GetCategory(), "payload")
}

func TestQdrant_SearchResult_HasChunkIndex_Good(t *core.T) {
	result := SearchResult{ChunkIndex: 0, ChunkIndexPresent: true}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, 0, result.GetChunkIndex())
}

func TestQdrant_SearchResult_HasChunkIndex_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertFalse(t, result.HasChunkIndex())
	core.AssertEqual(t, missingChunkIndex, result.GetChunkIndex())
}

func TestQdrant_SearchResult_HasChunkIndex_Ugly(t *core.T) {
	result := SearchResult{Payload: map[string]any{"chunk_index": float64(7)}}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, 7, result.GetChunkIndex())
}

func TestQdrant_SearchResult_GetChunkIndex_Good(t *core.T) {
	result := SearchResult{ChunkIndex: 5, ChunkIndexPresent: true}

	core.AssertEqual(t, 5, result.GetChunkIndex())
	core.AssertTrue(t, result.HasChunkIndex())
}

func TestQdrant_SearchResult_GetChunkIndex_Bad(t *core.T) {
	result := SearchResult{}

	core.AssertEqual(t, missingChunkIndex, result.GetChunkIndex())
	core.AssertFalse(t, result.HasChunkIndex())
}

func TestQdrant_SearchResult_GetChunkIndex_Ugly(t *core.T) {
	result := SearchResult{Index: 3, IndexPresent: true, Payload: map[string]any{"chunk_index": 8}}

	core.AssertEqual(t, 3, result.GetChunkIndex())
	core.AssertTrue(t, result.HasChunkIndex())
}
