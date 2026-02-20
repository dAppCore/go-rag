package rag

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DefaultQueryConfig tests ---

func TestDefaultQueryConfig(t *testing.T) {
	t.Run("returns expected default values", func(t *testing.T) {
		cfg := DefaultQueryConfig()

		assert.Equal(t, "hostuk-docs", cfg.Collection, "default collection should be hostuk-docs")
		assert.Equal(t, uint64(5), cfg.Limit, "default limit should be 5")
		assert.Equal(t, float32(0.5), cfg.Threshold, "default threshold should be 0.5")
		assert.Empty(t, cfg.Category, "default category should be empty")
	})
}

// --- FormatResultsText tests ---

func TestFormatResultsText(t *testing.T) {
	t.Run("empty results returns no-results message", func(t *testing.T) {
		result := FormatResultsText(nil)
		assert.Equal(t, "No results found.", result)
	})

	t.Run("empty slice returns no-results message", func(t *testing.T) {
		result := FormatResultsText([]QueryResult{})
		assert.Equal(t, "No results found.", result)
	})

	t.Run("single result with all fields", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     "Some relevant text about Go.",
				Source:   "docs/go-intro.md",
				Section:  "Introduction",
				Category: "documentation",
				Score:    0.95,
			},
		}

		output := FormatResultsText(results)

		assert.Contains(t, output, "Result 1")
		assert.Contains(t, output, "score: 0.95")
		assert.Contains(t, output, "Source: docs/go-intro.md")
		assert.Contains(t, output, "Section: Introduction")
		assert.Contains(t, output, "Category: documentation")
		assert.Contains(t, output, "Some relevant text about Go.")
	})

	t.Run("section omitted when empty", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     "No section here.",
				Source:   "test.md",
				Section:  "",
				Category: "docs",
				Score:    0.80,
			},
		}

		output := FormatResultsText(results)

		assert.NotContains(t, output, "Section:")
	})

	t.Run("multiple results numbered correctly", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First result.", Source: "a.md", Category: "docs", Score: 0.90},
			{Text: "Second result.", Source: "b.md", Category: "docs", Score: 0.85},
			{Text: "Third result.", Source: "c.md", Category: "docs", Score: 0.80},
		}

		output := FormatResultsText(results)

		assert.Contains(t, output, "Result 1")
		assert.Contains(t, output, "Result 2")
		assert.Contains(t, output, "Result 3")
		// Verify ordering: first result appears before second
		idx1 := strings.Index(output, "Result 1")
		idx2 := strings.Index(output, "Result 2")
		idx3 := strings.Index(output, "Result 3")
		assert.Less(t, idx1, idx2)
		assert.Less(t, idx2, idx3)
	})

	t.Run("score formatted to two decimal places", func(t *testing.T) {
		results := []QueryResult{
			{Text: "Test.", Source: "s.md", Category: "c", Score: 0.123456},
		}

		output := FormatResultsText(results)

		assert.Contains(t, output, "score: 0.12")
	})
}

// --- FormatResultsContext tests ---

func TestFormatResultsContext(t *testing.T) {
	t.Run("empty results returns empty string", func(t *testing.T) {
		result := FormatResultsContext(nil)
		assert.Equal(t, "", result)
	})

	t.Run("empty slice returns empty string", func(t *testing.T) {
		result := FormatResultsContext([]QueryResult{})
		assert.Equal(t, "", result)
	})

	t.Run("wraps output in retrieved_context tags", func(t *testing.T) {
		results := []QueryResult{
			{Text: "Hello world.", Source: "test.md", Section: "Intro", Category: "docs", Score: 0.9},
		}

		output := FormatResultsContext(results)

		assert.True(t, strings.HasPrefix(output, "<retrieved_context>\n"),
			"output should start with <retrieved_context> tag")
		assert.True(t, strings.HasSuffix(output, "</retrieved_context>"),
			"output should end with </retrieved_context> tag")
	})

	t.Run("wraps each result in document tags with attributes", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     "Content here.",
				Source:   "file.md",
				Section:  "My Section",
				Category: "documentation",
				Score:    0.88,
			},
		}

		output := FormatResultsContext(results)

		assert.Contains(t, output, `source="file.md"`)
		assert.Contains(t, output, `section="My Section"`)
		assert.Contains(t, output, `category="documentation"`)
		assert.Contains(t, output, "Content here.")
		assert.Contains(t, output, "</document>")
	})

	t.Run("escapes XML special characters in attributes and text", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     `Text with <tags> & "quotes" in it.`,
				Source:   `path/with<special>&chars.md`,
				Section:  `Section "One"`,
				Category: "docs",
				Score:    0.75,
			},
		}

		output := FormatResultsContext(results)

		// The html.EscapeString function escapes <, >, &, " and '
		assert.Contains(t, output, "&lt;tags&gt;")
		assert.Contains(t, output, "&amp;")
		assert.Contains(t, output, "&#34;quotes&#34;")
		// Source attribute should also be escaped
		assert.Contains(t, output, "path/with&lt;special&gt;&amp;chars.md")
	})

	t.Run("multiple results each wrapped in document tags", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First.", Source: "a.md", Section: "", Category: "docs", Score: 0.9},
			{Text: "Second.", Source: "b.md", Section: "", Category: "docs", Score: 0.8},
		}

		output := FormatResultsContext(results)

		// Count document tags
		assert.Equal(t, 2, strings.Count(output, "<document "))
		assert.Equal(t, 2, strings.Count(output, "</document>"))
	})
}

// --- FormatResultsJSON tests ---

func TestFormatResultsJSON(t *testing.T) {
	t.Run("empty results returns empty JSON array", func(t *testing.T) {
		result := FormatResultsJSON(nil)
		assert.Equal(t, "[]", result)
	})

	t.Run("empty slice returns empty JSON array", func(t *testing.T) {
		result := FormatResultsJSON([]QueryResult{})
		assert.Equal(t, "[]", result)
	})

	t.Run("single result produces valid JSON", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     "Test content.",
				Source:   "test.md",
				Section:  "Intro",
				Category: "docs",
				Score:    0.9234,
			},
		}

		output := FormatResultsJSON(results)

		// Verify it parses as valid JSON
		var parsed []map[string]any
		err := json.Unmarshal([]byte(output), &parsed)
		require.NoError(t, err, "output should be valid JSON")
		require.Len(t, parsed, 1)

		assert.Equal(t, "test.md", parsed[0]["source"])
		assert.Equal(t, "Intro", parsed[0]["section"])
		assert.Equal(t, "docs", parsed[0]["category"])
		assert.Equal(t, "Test content.", parsed[0]["text"])
		// Score is formatted to 4 decimal places
		assert.InDelta(t, 0.9234, parsed[0]["score"], 0.0001)
	})

	t.Run("multiple results produce valid JSON array", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First.", Source: "a.md", Section: "A", Category: "docs", Score: 0.95},
			{Text: "Second.", Source: "b.md", Section: "B", Category: "code", Score: 0.80},
			{Text: "Third.", Source: "c.md", Section: "C", Category: "task", Score: 0.70},
		}

		output := FormatResultsJSON(results)

		var parsed []map[string]any
		err := json.Unmarshal([]byte(output), &parsed)
		require.NoError(t, err, "output should be valid JSON")
		require.Len(t, parsed, 3)

		assert.Equal(t, "First.", parsed[0]["text"])
		assert.Equal(t, "Second.", parsed[1]["text"])
		assert.Equal(t, "Third.", parsed[2]["text"])
	})

	t.Run("special characters are JSON-escaped in text", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     "Line one\nLine two\twith tab and \"quotes\"",
				Source:   "special.md",
				Section:  "",
				Category: "docs",
				Score:    0.5,
			},
		}

		output := FormatResultsJSON(results)

		var parsed []map[string]any
		err := json.Unmarshal([]byte(output), &parsed)
		require.NoError(t, err, "output should be valid JSON even with special characters")
		assert.Equal(t, "Line one\nLine two\twith tab and \"quotes\"", parsed[0]["text"])
	})

	t.Run("score formatted to four decimal places", func(t *testing.T) {
		results := []QueryResult{
			{Text: "T.", Source: "s.md", Section: "", Category: "c", Score: 0.123456789},
		}

		output := FormatResultsJSON(results)

		assert.Contains(t, output, "0.1235")
	})
}
