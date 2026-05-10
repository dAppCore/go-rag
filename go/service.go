// SPDX-License-Identifier: EUPL-1.2

// Service registration for the rag package — exposes the canonical
// `NewService(opts)` + `Register(c)` shape per Mantis #1336, holding a
// pre-wired Embedder + VectorStore pair that consumers can use through
// a typed handle:
//
//	c, _ := core.New(
//	    core.WithService(rag.NewService(rag.Options{
//	        Embedder: emb,  // any rag.Embedder implementation
//	        Store:    vs,   // any rag.VectorStore implementation
//	    })),
//	)
//	svc := core.MustServiceFor[*rag.Service](c, "rag")
//	r := svc.Query(ctx, "what's a memvid?", "docs", 5)
//
// The package-level functions (Ingest, Query, IngestDirWith, etc.)
// remain the source of truth — Service methods delegate to them with
// the wired backends, so no behaviour duplication.

package rag

import (
	"context"

	core "dappco.re/go"
)

// Options configures the rag service. Embedder and Store are optional —
// nil values mean callers must use the package-level *With variants
// (QueryWith, IngestWith) and supply their own backends. Pre-wiring
// them here lets the Service expose the convenience methods.
type Options struct {
	// Embedder is the embedding-vector source for ingest + query.
	// Common construction: rag.NewOllamaEmbedder(endpoint, model).
	Embedder Embedder
	// Store is the vector storage backend.
	// Common construction: rag.NewQdrantStore(endpoint).
	Store VectorStore
}

// Service is the registerable handle for the rag package — embeds
// *core.ServiceRuntime[Options] for typed options access and exposes
// pre-wired Embedder + Store. Service methods delegate to the
// package-level *With functions.
//
// Usage example: `svc := core.MustServiceFor[*rag.Service](c, "rag"); r := svc.Query(ctx, "q", "coll", 5)`
type Service struct {
	*core.ServiceRuntime[Options]
	// Embedder is the wired embedding source. May be nil if the consumer
	// constructed Options without one — Service methods will return a
	// "not configured" error in that case.
	Embedder Embedder
	// Store is the wired vector store. Same nil semantics as Embedder.
	Store VectorStore
}

// NewService returns a factory that constructs a *Service holding the
// supplied Embedder + Store and registers it under "rag" via
// core.WithService.
//
//	core.WithService(rag.NewService(rag.Options{Embedder: emb, Store: vs}))
//
// Both Embedder and Store may be nil — Service methods return a
// configuration error rather than panic when called against unwired
// backends. Callers that don't need the typed handle can use the
// package-level *With functions directly.
func NewService(opts Options) func(*core.Core) core.Result {
	return func(c *core.Core) core.Result {
		return core.Ok(&Service{
			ServiceRuntime: core.NewServiceRuntime(c, opts),
			Embedder:       opts.Embedder,
			Store:          opts.Store,
		})
	}
}

// Register wires the rag service into the Core with empty Options —
// the imperative-style alternative to NewService. The resulting
// *Service holds nil Embedder + Store, so consumers must inject
// backends via the package-level *With functions or by mutating
// svc.Embedder / svc.Store directly before calling Service methods.
//
//	c := core.New()
//	if r := rag.Register(c); !r.OK { return r }
//	svc := core.MustServiceFor[*rag.Service](c, "rag")
//	svc.Embedder = myEmbedder
//	svc.Store = myStore
func Register(c *core.Core) core.Result {
	return NewService(Options{})(c)
}

// Query runs a vector query through the wired Embedder + Store. Returns
// a "rag.Service not configured" error if either backend is nil.
//
//	r := svc.Query(ctx, "what's memvid?", "docs", 5)
func (s *Service) Query(ctx context.Context, question, collectionName string, topK int) core.Result {
	if s == nil || s.Embedder == nil || s.Store == nil {
		return core.Fail(core.E("rag.Service.Query", "embedder + store not configured", nil))
	}
	return QueryWith(ctx, s.Store, s.Embedder, question, collectionName, topK)
}

// QueryContext runs a context-only query (returns assembled context
// rather than raw matches). Same nil-backend semantics as Query.
//
//	r := svc.QueryContext(ctx, "what's memvid?", "docs", 5)
func (s *Service) QueryContext(ctx context.Context, question, collectionName string, topK int) core.Result {
	if s == nil || s.Embedder == nil || s.Store == nil {
		return core.Fail(core.E("rag.Service.QueryContext", "embedder + store not configured", nil))
	}
	return QueryContextWith(ctx, s.Store, s.Embedder, question, collectionName, topK)
}

// IngestDir ingests every supported file in a directory into a named
// collection. Same nil-backend semantics as Query.
//
//	r := svc.IngestDir(ctx, "/docs", "docs", false)
func (s *Service) IngestDir(ctx context.Context, directory, collectionName string, recreateCollection bool) core.Result {
	if s == nil || s.Embedder == nil || s.Store == nil {
		return core.Fail(core.E("rag.Service.IngestDir", "embedder + store not configured", nil))
	}
	return IngestDirWith(ctx, s.Store, s.Embedder, directory, collectionName, recreateCollection)
}

// IngestFile ingests a single file. Same nil-backend semantics as Query.
//
//	r := svc.IngestFile(ctx, "/docs/memvid.md", "docs")
func (s *Service) IngestFile(ctx context.Context, filePath, collectionName string) core.Result {
	if s == nil || s.Embedder == nil || s.Store == nil {
		return core.Fail(core.E("rag.Service.IngestFile", "embedder + store not configured", nil))
	}
	return IngestFileWith(ctx, s.Store, s.Embedder, filePath, collectionName)
}
