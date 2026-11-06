package anksync

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func zstdBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatalf("zstd writer: %v", err)
	}
	return enc.EncodeAll(data, nil)
}

func unzstd(t *testing.T, data []byte) []byte {
	t.Helper()
	dec, err := zstd.NewReader(nil)
	if err != nil {
		t.Fatalf("zstd reader: %v", err)
	}
	out, err := dec.DecodeAll(data, nil)
	if err != nil {
		t.Fatalf("decode zstd body: %v", err)
	}
	return out
}

// TestHostKeyFraming pins the wire format: one anki-sync header carrying the
// version/key/client/session fields, and a zstd body. Getting either wrong makes
// the server answer a bare 400 with no explanation.
func TestHostKeyFraming(t *testing.T) {
	var gotHeader header
	var gotBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sync/hostKey" {
			t.Errorf("path = %q, want /sync/hostKey", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/octet-stream" {
			t.Errorf("content-type = %q, want application/octet-stream", ct)
		}
		if err := json.Unmarshal([]byte(r.Header.Get("anki-sync")), &gotHeader); err != nil {
			t.Fatalf("anki-sync header not JSON: %v", err)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(unzstd(t, raw), &gotBody); err != nil {
			t.Fatalf("body not zstd JSON: %v", err)
		}
		w.Header().Set("anki-original-size", "15")
		w.Write(zstdBytes(t, []byte(`{"key":"hkey-1"}`)))
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/", "deutsch-srs/test")
	if err := c.HostKey(context.Background(), "devji", "s3cret"); err != nil {
		t.Fatalf("HostKey: %v", err)
	}
	if c.hkey != "hkey-1" {
		t.Errorf("hkey = %q, want hkey-1", c.hkey)
	}
	if gotHeader.Version != syncVersion {
		t.Errorf("header v = %d, want %d", gotHeader.Version, syncVersion)
	}
	if gotHeader.Key != "" {
		t.Errorf("hostKey carries the previous key %q; login must send an empty k", gotHeader.Key)
	}
	if gotHeader.ClientVer != "deutsch-srs/test" {
		t.Errorf("header c = %q", gotHeader.ClientVer)
	}
	if gotBody["u"] != "devji" || gotBody["p"] != "s3cret" {
		t.Errorf("body = %v, want u/p", gotBody)
	}
}

// TestCallsCarryHostKey checks the key from the login is reused, which is what
// separates an authenticated call from a 400.
func TestCallsCarryHostKey(t *testing.T) {
	var keys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var h header
		json.Unmarshal([]byte(r.Header.Get("anki-sync")), &h)
		keys = append(keys, r.URL.Path+"="+h.Key)
		switch r.URL.Path {
		case "/sync/hostKey":
			w.Write(zstdBytes(t, []byte(`{"key":"K"}`)))
		case "/sync/meta":
			w.Write(zstdBytes(t, []byte(`{"mod":1700000000000,"scm":1,"usn":42,"ts":1700000000}`)))
		case "/sync/download":
			w.Write(zstdBytes(t, []byte("SQLite format 3\x00")))
		default:
			t.Errorf("unexpected method %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	c := NewClient(srv.URL, "test")
	if err := c.HostKey(ctx, "u", "p"); err != nil {
		t.Fatal(err)
	}
	meta, err := c.Meta(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Usn != 42 {
		t.Errorf("usn = %d, want 42", meta.Usn)
	}
	col, err := c.Download(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(col) != "SQLite format 3\x00" {
		t.Errorf("download = %q", col)
	}
	want := []string{"/sync/hostKey=", "/sync/meta=K", "/sync/download=K"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Errorf("keys = %v, want %v", keys, want)
	}
}

// TestErrorTextSurvives makes sure a rejected login reports what the server
// said instead of a bare status.
func TestErrorTextSurvives(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403"))
	}))
	defer srv.Close()

	err := NewClient(srv.URL, "test").HostKey(context.Background(), "u", "wrong")
	if err == nil {
		t.Fatal("expected an error for a 403")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error = %v, want the server's 403 text", err)
	}
}
