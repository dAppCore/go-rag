// SPDX-License-Identifier: EUPL-1.2

package rag

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestNewService_NilBackends_RegistersWithErrorOnUse — happy path for the
// no-backend lazy-injection shape. Service registers cleanly even without
// Embedder + Store; methods that need them return a configuration error.
func TestNewService_NilBackends_RegistersWithErrorOnUse(t *testing.T) {
	c := core.New(core.WithService(NewService(Options{})))
	r := c.Service("rag")
	if !r.OK {
		t.Fatal("rag service not registered via NewService(Options{})")
	}
	svc := r.Value.(*Service)

	ctx := context.Background()
	if q := svc.Query(ctx, "test", "coll", 5); q.OK {
		t.Fatal("expected nil-backend Query to fail")
	}
	if q := svc.QueryContext(ctx, "test", "coll", 5); q.OK {
		t.Fatal("expected nil-backend QueryContext to fail")
	}
	if i := svc.IngestDir(ctx, "/dir", "coll", false); i.OK {
		t.Fatal("expected nil-backend IngestDir to fail")
	}
	if i := svc.IngestFile(ctx, "/file", "coll"); i.OK {
		t.Fatal("expected nil-backend IngestFile to fail")
	}
}

// TestService_WiredBackends_Good — happy-path delegation through wired
// Embedder + Store. Every Service method forwards to the package-level
// *With function and returns its OK result.
func TestService_WiredBackends_Good(t *testing.T) {
	store := newMockVectorStore()
	embedder := newMockEmbedder(8)
	c := core.New(core.WithService(NewService(Options{Embedder: embedder, Store: store})))
	svc := core.MustServiceFor[*Service](c, "rag")

	ctx := context.Background()

	// IngestFile populates the collection so the subsequent query has data.
	dir := t.TempDir()
	path := core.JoinPath(dir, "doc.md")
	writeFile(t, path, "## Section\n\nHello wired world.\n")
	if r := svc.IngestFile(ctx, path, "wired-col"); !r.OK {
		t.Fatalf("wired IngestFile failed: %s", r.Error())
	}

	// IngestDir over the same directory.
	if r := svc.IngestDir(ctx, dir, "wired-dir", false); !r.OK {
		t.Fatalf("wired IngestDir failed: %s", r.Error())
	}

	// Query delegates to QueryWith and returns matches.
	if r := svc.Query(ctx, "hello", "wired-col", 5); !r.OK {
		t.Fatalf("wired Query failed: %s", r.Error())
	}

	// QueryContext delegates to QueryContextWith.
	if r := svc.QueryContext(ctx, "hello", "wired-col", 5); !r.OK {
		t.Fatalf("wired QueryContext failed: %s", r.Error())
	}

	if store.upsertCallCount() == 0 {
		t.Fatal("expected upserts through wired Store")
	}
	if store.searchCallCount() == 0 {
		t.Fatal("expected searches through wired Store")
	}
}

// TestService_PartialBackends_Ugly — only one backend wired still fails
// closed on every method (both Embedder and Store are required).
func TestService_PartialBackends_Ugly(t *testing.T) {
	ctx := context.Background()

	embedderOnly := &Service{Embedder: newMockEmbedder(4)}
	if r := embedderOnly.Query(ctx, "q", "c", 5); r.OK {
		t.Fatal("expected store-less Query to fail")
	}
	if r := embedderOnly.IngestFile(ctx, "/f", "c"); r.OK {
		t.Fatal("expected store-less IngestFile to fail")
	}

	storeOnly := &Service{Store: newMockVectorStore()}
	if r := storeOnly.QueryContext(ctx, "q", "c", 5); r.OK {
		t.Fatal("expected embedder-less QueryContext to fail")
	}
	if r := storeOnly.IngestDir(ctx, "/d", "c", false); r.OK {
		t.Fatal("expected embedder-less IngestDir to fail")
	}
}

// TestRegister_DefaultsRegistersService — imperative Register(c) shorthand.
func TestRegister_DefaultsRegistersService(t *testing.T) {
	c := core.New(core.WithService(Register))
	r := c.Service("rag")
	if !r.OK {
		t.Fatalf("rag service not registered via Register, got %#v", r.Value)
	}
}

// TestService_NilReceiver_GuardsMethods — defensive nil-receiver guards on
// every Service method don't panic.
func TestService_NilReceiver_GuardsMethods(t *testing.T) {
	var svc *Service
	ctx := context.Background()
	if r := svc.Query(ctx, "q", "c", 5); r.OK {
		t.Fatal("expected nil-receiver Query to fail")
	}
	if r := svc.QueryContext(ctx, "q", "c", 5); r.OK {
		t.Fatal("expected nil-receiver QueryContext to fail")
	}
	if r := svc.IngestDir(ctx, "/d", "c", false); r.OK {
		t.Fatal("expected nil-receiver IngestDir to fail")
	}
	if r := svc.IngestFile(ctx, "/f", "c"); r.OK {
		t.Fatal("expected nil-receiver IngestFile to fail")
	}
}
