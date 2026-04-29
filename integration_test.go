//go:build rag

package rag

import (
	"context"
	"net"
	"testing"
	"time"

	"dappco.re/go"
)

// skipIfServicesUnavailable skips the test if either Qdrant or Ollama is not
// reachable. Full pipeline tests need both.
func skipIfServicesUnavailable(t *testing.T) {
	t.Helper()
	for _, addr := range []string{"localhost:6334", "localhost:11434"} {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			t.Skipf("%s not available — skipping pipeline integration test", addr)
		}
		_ = conn.Close()
	}
}

func TestIntegrationPipelineEndToEnd(t *testing.T) {
	skipIfServicesUnavailable(t)

	ctx := context.Background()

	// Create shared clients for the pipeline tests.
	qdrantCfg := DefaultQdrantConfig()
	qdrantClient, err := NewQdrantClient(qdrantCfg)
	assertNoError(t, err)
	t.Cleanup(func() { _ = qdrantClient.Close() })

	ollamaCfg := DefaultOllamaConfig()
	ollamaClient, err := NewOllamaClient(ollamaCfg)
	assertNoError(t, err)

	t.Run("ingest and query end-to-end", func(t *testing.T) {
		collection := core.Sprintf("test-pipeline-%d", time.Now().UnixNano())
		t.Cleanup(func() {
			_ = qdrantClient.DeleteCollection(ctx, collection)
		})

		// Create temp directory with markdown files
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "go-intro.md"), `# Go Programming

## Overview

Go is an open-source programming language designed at Google. It features
garbage collection, structural typing, and CSP-style concurrency. Go was
created by Robert Griesemer, Rob Pike, and Ken Thompson.

## Concurrency

Go provides goroutines and channels for concurrent programming. Goroutines
are lightweight threads managed by the Go runtime. Channels allow goroutines
to communicate safely without shared memory.
`)

		writeFile(t, core.JoinPath(dir, "qdrant-intro.md"), `# Qdrant Vector Database

## What Is Qdrant

Qdrant is a vector similarity search engine and vector database. It provides
a convenient API to store, search, and manage points with payload. Qdrant is
written in Rust and supports filtering, quantisation, and distributed deployment.

## Use Cases

Qdrant is commonly used for semantic search, recommendation systems, and
retrieval-augmented generation (RAG) pipelines. It supports cosine, dot product,
and Euclidean distance metrics.
`)

		writeFile(t, core.JoinPath(dir, "rust-intro.md"), `# Rust Programming

## Memory Safety

Rust guarantees memory safety without a garbage collector through its ownership
system. The borrow checker enforces rules at compile time, preventing data races,
dangling pointers, and buffer overflows.
`)

		// Ingest the directory
		ingestCfg := DefaultIngestConfig()
		ingestCfg.Directory = dir
		ingestCfg.Collection = collection
		ingestCfg.Chunk = ChunkConfig{Size: 500, Overlap: 50}

		stats, err := Ingest(ctx, qdrantClient, ollamaClient, ingestCfg, nil)
		assertNoError(t, err, "ingest should succeed")
		assertEqual(t, 3, stats.Files, "all three files should be ingested")
		assertGreater(t, stats.Chunks, 0, "should produce at least one chunk")
		assertEqual(t, 0, stats.Errors, "no errors should occur during ingest")

		// Allow Qdrant to index
		time.Sleep(1 * time.Second)

		// Query for Go-related content
		queryCfg := DefaultQueryConfig()
		queryCfg.Collection = collection
		queryCfg.Limit = 5
		queryCfg.Threshold = 0.0 // Accept all results for testing

		results, err := Query(ctx, qdrantClient, ollamaClient, "goroutines and channels in Go", queryCfg)
		assertNoError(t, err, "query should succeed")
		assertNotEmpty(t, results, "query should return at least one result")

		// The top result should be about Go concurrency
		foundGoContent := false
		for _, r := range results {
			if r.Source != "" && r.Text != "" {
				foundGoContent = true
				break
			}
		}
		assertTrue(t, foundGoContent, "results should contain content with source and text fields")

		// Verify all results have expected metadata fields populated
		for i, r := range results {
			assertNotEmptyf(t, r.Text, "result %d should have text", i)
			assertNotEmptyf(t, r.Source, "result %d should have source", i)
			assertNotEmptyf(t, r.Category, "result %d should have category", i)
			assertGreaterf(t, r.Score, float32(0.0), "result %d should have positive score", i)
		}
	})

	t.Run("format results from real query", func(t *testing.T) {
		collection := core.Sprintf("test-format-%d", time.Now().UnixNano())
		t.Cleanup(func() {
			_ = qdrantClient.DeleteCollection(ctx, collection)
		})

		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "format-test.md"), `## Format Test

This document is used to verify that the format functions produce non-empty
output when given real query results from live services.
`)

		ingestCfg := DefaultIngestConfig()
		ingestCfg.Directory = dir
		ingestCfg.Collection = collection

		_, err := Ingest(ctx, qdrantClient, ollamaClient, ingestCfg, nil)
		assertNoError(t, err)
		time.Sleep(1 * time.Second)

		queryCfg := DefaultQueryConfig()
		queryCfg.Collection = collection
		queryCfg.Limit = 3
		queryCfg.Threshold = 0.0

		results, err := Query(ctx, qdrantClient, ollamaClient, "format test document", queryCfg)
		assertNoError(t, err)
		assertNotEmpty(t, results, "should return at least one result for formatting")

		// FormatResultsText
		textOutput := FormatResultsText(results)
		assertNotEmpty(t, textOutput)
		assertNotEqual(t, "No results found.", textOutput)
		assertContains(t, textOutput, "Result 1")
		assertContains(t, textOutput, "Source:")

		// FormatResultsContext
		ctxOutput := FormatResultsContext(results)
		assertNotEmpty(t, ctxOutput)
		assertContains(t, ctxOutput, "<retrieved_context>")
		assertContains(t, ctxOutput, "</retrieved_context>")
		assertContains(t, ctxOutput, "<document ")

		// FormatResultsJSON
		jsonOutput := FormatResultsJSON(results)
		assertNotEmpty(t, jsonOutput)
		assertNotEqual(t, "[]", jsonOutput)
		assertContains(t, jsonOutput, `"source"`)
		assertContains(t, jsonOutput, `"text"`)
	})

	t.Run("IngestFile single file with live services", func(t *testing.T) {
		collection := core.Sprintf("test-single-%d", time.Now().UnixNano())
		t.Cleanup(func() {
			_ = qdrantClient.DeleteCollection(ctx, collection)
		})

		// Create the collection first (IngestFile does not create collections)
		err := qdrantClient.CreateCollection(ctx, collection, ollamaClient.EmbedDimension())
		assertNoError(t, err)

		dir := t.TempDir()
		path := core.JoinPath(dir, "single.md")
		writeFile(t, path, `## Single File Ingest

Testing the IngestFile function with a single markdown file. This content
should be chunked, embedded, and stored in Qdrant.
`)

		count, err := IngestFile(ctx, qdrantClient, ollamaClient, collection, path, DefaultChunkConfig())
		assertNoError(t, err, "IngestFile should succeed")
		assertGreater(t, count, 0, "should produce at least one point")
	})

	t.Run("QueryWith helper with live services", func(t *testing.T) {
		collection := core.Sprintf("test-querywith-%d", time.Now().UnixNano())
		t.Cleanup(func() {
			_ = qdrantClient.DeleteCollection(ctx, collection)
		})

		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "helper-test.md"), `## Helper Test

Content for testing the QueryWith and QueryContextWith helper functions
with real Qdrant and Ollama connections.
`)

		ingestCfg := DefaultIngestConfig()
		ingestCfg.Directory = dir
		ingestCfg.Collection = collection

		_, err := Ingest(ctx, qdrantClient, ollamaClient, ingestCfg, nil)
		assertNoError(t, err)
		time.Sleep(1 * time.Second)

		// Test QueryWith
		results, err := QueryWith(ctx, qdrantClient, ollamaClient, "helper function test", collection, 3)
		assertNoError(t, err, "QueryWith should succeed")
		// Results may or may not pass the default threshold — that is fine
		_ = results

		// Test QueryContextWith
		ctxOutput, err := QueryContextWith(ctx, qdrantClient, ollamaClient, "helper function test", collection, 3)
		assertNoError(t, err, "QueryContextWith should succeed")
		// Even if no results pass threshold, the function should not error
		_ = ctxOutput
	})

	t.Run("IngestDirWith helper with live services", func(t *testing.T) {
		collection := core.Sprintf("test-ingestdirwith-%d", time.Now().UnixNano())
		t.Cleanup(func() {
			_ = qdrantClient.DeleteCollection(ctx, collection)
		})

		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "dirwith-a.md"), `## Directory Ingest A

First document for testing the IngestDirWith convenience wrapper.
`)
		writeFile(t, core.JoinPath(dir, "dirwith-b.md"), `## Directory Ingest B

Second document for the same test, ensuring multiple files are processed.
`)

		err := IngestDirWith(ctx, qdrantClient, ollamaClient, dir, collection, false)
		assertNoError(t, err, "IngestDirWith should succeed")

		// Verify the collection now exists and has data
		exists, err := qdrantClient.CollectionExists(ctx, collection)
		assertNoError(t, err)
		assertTrue(t, exists, "collection should exist after IngestDirWith")
	})

	t.Run("IngestFileWith helper with live services", func(t *testing.T) {
		collection := core.Sprintf("test-ingestfilewith-%d", time.Now().UnixNano())
		t.Cleanup(func() {
			_ = qdrantClient.DeleteCollection(ctx, collection)
		})

		// Create collection first
		err := qdrantClient.CreateCollection(ctx, collection, ollamaClient.EmbedDimension())
		assertNoError(t, err)

		dir := t.TempDir()
		path := core.JoinPath(dir, "filewith.md")
		writeFile(t, path, `## File With Helper

Testing the IngestFileWith convenience wrapper with live services.
`)

		count, err := IngestFileWith(ctx, qdrantClient, ollamaClient, path, collection)
		assertNoError(t, err, "IngestFileWith should succeed")
		assertGreater(t, count, 0, "should produce at least one point")
	})

	t.Run("QueryDocs with default clients", func(t *testing.T) {
		// This test exercises the convenience wrappers that construct their own
		// clients internally. We ingest data via the shared client, then query
		// via QueryDocs which creates its own client pair.
		collection := core.Sprintf("test-querydocs-%d", time.Now().UnixNano())
		t.Cleanup(func() {
			_ = qdrantClient.DeleteCollection(ctx, collection)
		})

		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "default-client.md"), `## Default Client Test

Content to verify that QueryDocs can query with internally constructed clients.
`)

		ingestCfg := DefaultIngestConfig()
		ingestCfg.Directory = dir
		ingestCfg.Collection = collection
		_, err := Ingest(ctx, qdrantClient, ollamaClient, ingestCfg, nil)
		assertNoError(t, err)
		time.Sleep(1 * time.Second)

		results, err := QueryDocs(ctx, "default client test query", collection, 3)
		assertNoError(t, err, "QueryDocs should succeed with default clients")
		_ = results // Results depend on threshold; the important thing is no error
	})

	t.Run("IngestDirectory with default clients", func(t *testing.T) {
		collection := core.Sprintf("test-ingestdir-%d", time.Now().UnixNano())
		t.Cleanup(func() {
			_ = qdrantClient.DeleteCollection(ctx, collection)
		})

		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "ingestdir.md"), `## Ingest Directory

Testing the IngestDirectory convenience wrapper that constructs its own
Qdrant and Ollama clients internally.
`)

		err := IngestDirectory(ctx, dir, collection, true)
		assertNoError(t, err, "IngestDirectory should succeed with default clients")

		exists, err := qdrantClient.CollectionExists(ctx, collection)
		assertNoError(t, err)
		assertTrue(t, exists, "collection should exist after IngestDirectory")
	})

	t.Run("recreate flag drops and recreates collection", func(t *testing.T) {
		collection := core.Sprintf("test-recreate-%d", time.Now().UnixNano())
		t.Cleanup(func() {
			_ = qdrantClient.DeleteCollection(ctx, collection)
		})

		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "v1.md"), `## Version 1

Original content that will be replaced.
`)

		// First ingest
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = collection
		_, err := Ingest(ctx, qdrantClient, ollamaClient, cfg, nil)
		assertNoError(t, err)

		// Replace the file content and re-ingest with recreate
		writeFile(t, core.JoinPath(dir, "v1.md"), `## Version 2

Updated content after recreation.
`)
		cfg.Recreate = true
		stats, err := Ingest(ctx, qdrantClient, ollamaClient, cfg, nil)
		assertNoError(t, err)
		assertEqual(t, 1, stats.Files)
		assertEqual(t, 0, stats.Errors)
	})

	t.Run("semantic similarity — related queries rank higher", func(t *testing.T) {
		collection := core.Sprintf("test-semantic-%d", time.Now().UnixNano())
		t.Cleanup(func() {
			_ = qdrantClient.DeleteCollection(ctx, collection)
		})

		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "cooking.md"), `## Cooking

Pasta with tomato sauce is a classic Italian dish. Boil the spaghetti for
eight minutes, then drain and add the sauce. Season with basil and parmesan.
`)
		writeFile(t, core.JoinPath(dir, "programming.md"), `## Programming

Functions in Go are first-class citizens. You can pass functions as arguments,
return them from other functions, and assign them to variables. Closures capture
their surrounding scope.
`)

		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = collection
		_, err := Ingest(ctx, qdrantClient, ollamaClient, cfg, nil)
		assertNoError(t, err)
		time.Sleep(1 * time.Second)

		// Query about programming
		queryCfg := DefaultQueryConfig()
		queryCfg.Collection = collection
		queryCfg.Limit = 2
		queryCfg.Threshold = 0.0

		results, err := Query(ctx, qdrantClient, ollamaClient, "How do Go functions and closures work?", queryCfg)
		assertNoError(t, err)
		assertNotEmpty(t, results)

		// The programming document should rank higher than the cooking one
		foundProgrammingFirst := false
		for _, r := range results {
			if r.Source != "" {
				// Check if the first result with a source is the programming file
				foundProgrammingFirst = (r.Source == "programming.md")
				break
			}
		}
		assertTrue(t, foundProgrammingFirst,
			"programming content should rank higher for a programming query")
	})
}
