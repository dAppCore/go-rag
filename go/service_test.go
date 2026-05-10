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
