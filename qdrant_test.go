package rag

import (
	"testing"

	"github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/assert"
)

// --- DefaultQdrantConfig tests ---

func TestQdrant_DefaultQdrantConfig_Good(t *testing.T) {
	t.Run("returns expected default values", func(t *testing.T) {
		cfg := DefaultQdrantConfig()

		assert.Equal(t, "localhost", cfg.Host, "default host should be localhost")
		assert.Equal(t, 6334, cfg.Port, "default gRPC port should be 6334")
		assert.False(t, cfg.UseTLS, "TLS should be disabled by default")
		assert.Empty(t, cfg.APIKey, "API key should be empty by default")
	})
}

// --- normalizeQdrantGRPCPort tests ---

func TestQdrant_NormalizeQdrantGRPCPort_Good(t *testing.T) {
	t.Run("maps REST port 6333 to gRPC port 6334", func(t *testing.T) {
		assert.Equal(t, 6334, normalizeQdrantGRPCPort(6333))
	})

	t.Run("leaves other ports unchanged", func(t *testing.T) {
		assert.Equal(t, 6334, normalizeQdrantGRPCPort(6334))
		assert.Equal(t, 7000, normalizeQdrantGRPCPort(7000))
	})
}

// --- pointIDToString tests ---

func TestQdrant_PointIDToString_Good(t *testing.T) {
	t.Run("nil ID returns empty string", func(t *testing.T) {
		result := pointIDToString(nil)
		assert.Equal(t, "", result)
	})

	t.Run("numeric ID returns string representation", func(t *testing.T) {
		id := qdrant.NewIDNum(42)
		result := pointIDToString(id)
		assert.Equal(t, "42", result)
	})

	t.Run("numeric ID zero", func(t *testing.T) {
		id := qdrant.NewIDNum(0)
		result := pointIDToString(id)
		assert.Equal(t, "0", result)
	})

	t.Run("large numeric ID", func(t *testing.T) {
		id := qdrant.NewIDNum(18446744073709551615) // max uint64
		result := pointIDToString(id)
		assert.Equal(t, "18446744073709551615", result)
	})

	t.Run("UUID ID returns UUID string", func(t *testing.T) {
		uuid := "550e8400-e29b-41d4-a716-446655440000"
		id := qdrant.NewIDUUID(uuid)
		result := pointIDToString(id)
		assert.Equal(t, uuid, result)
	})

	t.Run("empty UUID returns empty string", func(t *testing.T) {
		id := qdrant.NewIDUUID("")
		result := pointIDToString(id)
		assert.Equal(t, "", result)
	})

	t.Run("string ID via NewID returns the UUID", func(t *testing.T) {
		// NewID creates a UUID-type PointId
		uuid := "abc-123-def"
		id := qdrant.NewID(uuid)
		result := pointIDToString(id)
		assert.Equal(t, uuid, result)
	})
}

// --- valueToGo tests ---

func TestQdrant_ValueToGo_Good(t *testing.T) {
	t.Run("nil value returns nil", func(t *testing.T) {
		result := valueToGo(nil)
		assert.Nil(t, result)
	})

	t.Run("string value", func(t *testing.T) {
		v := qdrant.NewValueString("hello world")
		result := valueToGo(v)
		assert.Equal(t, "hello world", result)
	})

	t.Run("empty string value", func(t *testing.T) {
		v := qdrant.NewValueString("")
		result := valueToGo(v)
		assert.Equal(t, "", result)
	})

	t.Run("integer value", func(t *testing.T) {
		v := qdrant.NewValueInt(42)
		result := valueToGo(v)
		assert.Equal(t, int64(42), result)
	})

	t.Run("negative integer value", func(t *testing.T) {
		v := qdrant.NewValueInt(-100)
		result := valueToGo(v)
		assert.Equal(t, int64(-100), result)
	})

	t.Run("zero integer value", func(t *testing.T) {
		v := qdrant.NewValueInt(0)
		result := valueToGo(v)
		assert.Equal(t, int64(0), result)
	})

	t.Run("double value", func(t *testing.T) {
		v := qdrant.NewValueDouble(3.14)
		result := valueToGo(v)
		assert.Equal(t, float64(3.14), result)
	})

	t.Run("negative double value", func(t *testing.T) {
		v := qdrant.NewValueDouble(-2.718)
		result := valueToGo(v)
		assert.Equal(t, float64(-2.718), result)
	})

	t.Run("bool true value", func(t *testing.T) {
		v := qdrant.NewValueBool(true)
		result := valueToGo(v)
		assert.Equal(t, true, result)
	})

	t.Run("bool false value", func(t *testing.T) {
		v := qdrant.NewValueBool(false)
		result := valueToGo(v)
		assert.Equal(t, false, result)
	})

	t.Run("null value returns nil", func(t *testing.T) {
		v := qdrant.NewValueNull()
		result := valueToGo(v)
		// NullValue is not handled by the switch, falls to default -> nil
		assert.Nil(t, result)
	})

	t.Run("list value with mixed types", func(t *testing.T) {
		v := qdrant.NewValueFromList(
			qdrant.NewValueString("alpha"),
			qdrant.NewValueInt(99),
			qdrant.NewValueBool(true),
		)
		result := valueToGo(v)

		list, ok := result.([]any)
		assert.True(t, ok, "result should be a []any")
		assert.Len(t, list, 3)
		assert.Equal(t, "alpha", list[0])
		assert.Equal(t, int64(99), list[1])
		assert.Equal(t, true, list[2])
	})

	t.Run("empty list value", func(t *testing.T) {
		v := qdrant.NewValueList(&qdrant.ListValue{Values: []*qdrant.Value{}})
		result := valueToGo(v)

		list, ok := result.([]any)
		assert.True(t, ok, "result should be a []any")
		assert.Empty(t, list)
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
		assert.True(t, ok, "result should be a map[string]any")
		assert.Equal(t, "test", m["name"])
		assert.Equal(t, int64(25), m["age"])
	})

	t.Run("empty struct value", func(t *testing.T) {
		v := qdrant.NewValueStruct(&qdrant.Struct{
			Fields: map[string]*qdrant.Value{},
		})
		result := valueToGo(v)

		m, ok := result.(map[string]any)
		assert.True(t, ok, "result should be a map[string]any")
		assert.Empty(t, m)
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
		assert.True(t, ok, "result should be a map[string]any")

		tags, ok := m["tags"].([]any)
		assert.True(t, ok, "tags should be a []any")
		assert.Equal(t, []any{"go", "rag"}, tags)
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
		assert.True(t, ok)
		nested, ok := m["nested"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "value", nested["key"])
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

		assert.Equal(t, "test-id-123", p.ID)
		assert.Len(t, p.Vector, 3)
		assert.Equal(t, float32(0.1), p.Vector[0])
		assert.Equal(t, "hello", p.Payload["text"])
		assert.Equal(t, "test.md", p.Payload["source"])
	})

	t.Run("Point with empty payload", func(t *testing.T) {
		p := Point{
			ID:      "empty",
			Vector:  []float32{},
			Payload: map[string]any{},
		}

		assert.Equal(t, "empty", p.ID)
		assert.Empty(t, p.Vector)
		assert.Empty(t, p.Payload)
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

		assert.Equal(t, "result-1", sr.ID)
		assert.Equal(t, float32(0.95), sr.Score)
		assert.Equal(t, "some text", sr.Payload["text"])
	})
}
