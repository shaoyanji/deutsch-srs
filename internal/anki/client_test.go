package anki

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if req["action"] != "version" {
			t.Fatalf("action=%v", req["action"])
		}
		_, _ = w.Write([]byte(`{"result":6,"error":null}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	v, err := c.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v != 6 {
		t.Fatalf("version=%d", v)
	}
}

func TestClientErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":null,"error":"boom"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.Version(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
