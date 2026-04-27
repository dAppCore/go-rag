package rag

import (
	"context"
	"testing"

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

func TestQdrant_NormalizeQdrantGRPCPort_Good(t *testing.T) {
	t.Run("maps REST port 6333 to gRPC port 6334", func(t *testing.T) {
		assertEqual(t, 6334, normalizeQdrantGRPCPort(6333))
	})

	t.Run("leaves other ports unchanged", func(t *testing.T) {
		assertEqual(t, 6334, normalizeQdrantGRPCPort(6334))
		assertEqual(t, 7000, normalizeQdrantGRPCPort(7000))
	})
}

// --- pointIDToString tests ---

func TestQdrant_PointIDToString_Good(t *testing.T) {
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

func TestQdrant_ValueToGo_Good(t *testing.T) {
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
		results, err := store.Search(context.Background(), "docs", []float32{0.1}, 5, nil)
		assertNoError(t, err)
		assertLen(t, results, 2)
	})

	t.Run("still accepts an optional filter map", func(t *testing.T) {
		results, err := store.Search(context.Background(), "docs", []float32{0.1}, 5, map[string]string{"source": "a.md"})
		assertNoError(t, err)
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
