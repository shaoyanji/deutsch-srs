package deckfetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleDeck = "deck: Wortschatz A1\ncards:\n  - front: Haus\n    back: house\n    note_type: vocab\n    tags:\n      - nomen\n      - a1\n"

func shaOf(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func manifestFor(t *testing.T, baseURL string, body []byte) Manifest {
	t.Helper()
	raw := fmt.Sprintf(`{
  "gist": "abc", "revision": "rev", "base_url": %q,
  "decks": [{"name": "Wortschatz A1", "level": "a1", "file": "wortschatz-a1.yaml",
             "sha256": %q, "bytes": %d, "cards": 1}]
}`, baseURL, shaOf(body), len(body))
	m, err := ParseManifest([]byte(raw))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	return m
}

func TestEmbeddedManifestIsUsable(t *testing.T) {
	m, err := Embedded()
	if err != nil {
		t.Fatalf("Embedded: %v", err)
	}
	if len(m.Decks) != 4 {
		t.Fatalf("decks = %d, want the four Wortschatz levels", len(m.Decks))
	}
	want := []string{"a1", "a2", "b1", "b2"}
	for i, d := range m.Decks {
		if d.Level != want[i] {
			t.Fatalf("deck %d level = %q, want %q", i, d.Level, want[i])
		}
		if !strings.HasPrefix(m.URL(d), "https://") {
			t.Fatalf("%s url = %q", d.File, m.URL(d))
		}
		// The raw link is pinned to a revision; without it an edit to the gist
		// would silently change what every clone downloads.
		if !strings.Contains(m.URL(d), "/raw/"+m.Revision+"/") {
			t.Fatalf("%s url is not pinned to revision %s: %s", d.File, m.Revision, m.URL(d))
		}
		if d.Cards <= 0 || d.Bytes <= 0 {
			t.Fatalf("%s has no size or card count: %+v", d.File, d)
		}
	}
}

func TestPullWritesAVerifiedCopy(t *testing.T) {
	body := []byte(sampleDeck)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "wortschatz-a1.yaml") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	results, err := Pull(context.Background(), srv.Client(), manifestFor(t, srv.URL+"/", body), Options{Dir: dir})
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if len(results) != 1 || results[0].Skipped {
		t.Fatalf("results = %+v", results)
	}
	got, err := os.ReadFile(filepath.Join(dir, "wortschatz-a1.yaml"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != sampleDeck {
		t.Fatalf("content = %q", got)
	}
	if _, err := Verify(manifestFor(t, srv.URL+"/", body), dir); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestPullSkipsACopyThatAlreadyMatches(t *testing.T) {
	body := []byte(sampleDeck)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	m := manifestFor(t, srv.URL+"/", body)
	if _, err := Pull(context.Background(), srv.Client(), m, Options{Dir: dir}); err != nil {
		t.Fatalf("first pull: %v", err)
	}
	results, err := Pull(context.Background(), srv.Client(), m, Options{Dir: dir})
	if err != nil {
		t.Fatalf("second pull: %v", err)
	}
	if len(results) != 1 || !results[0].Skipped {
		t.Fatalf("second pull = %+v, want a skip", results)
	}
	if hits != 1 {
		t.Fatalf("server hit %d times, want 1", hits)
	}

	results, err = Pull(context.Background(), srv.Client(), m, Options{Dir: dir, Force: true})
	if err != nil {
		t.Fatalf("forced pull: %v", err)
	}
	if results[0].Skipped || hits != 2 {
		t.Fatalf("force did not refetch: %+v, hits=%d", results, hits)
	}
}

func TestPullRejectsAWrongBody(t *testing.T) {
	body := []byte(sampleDeck)
	// Same length as the real deck, different content: this has to fail on the
	// checksum, not on the size guard.
	tampered := []byte(strings.Replace(sampleDeck, "Haus", "Haud", 1))
	if len(tampered) != len(body) {
		t.Fatalf("tampered body is %d bytes, want %d", len(tampered), len(body))
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(tampered)
	}))
	defer srv.Close()

	dir := t.TempDir()
	_, err := Pull(context.Background(), srv.Client(), manifestFor(t, srv.URL+"/", body), Options{Dir: dir})
	if err == nil {
		t.Fatal("a checksum mismatch must fail the fetch")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "wortschatz-a1.yaml")); !os.IsNotExist(statErr) {
		t.Fatalf("a rejected body was written to disk: %v", statErr)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		t.Fatalf("temp file left behind: %s", e.Name())
	}
}

func TestPullRejectsATruncatedBody(t *testing.T) {
	body := []byte(sampleDeck)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body[:len(body)/2])
	}))
	defer srv.Close()

	dir := t.TempDir()
	_, err := Pull(context.Background(), srv.Client(), manifestFor(t, srv.URL+"/", body), Options{Dir: dir})
	if err == nil {
		t.Fatal("a short body must fail the fetch")
	}
	if !strings.Contains(err.Error(), "bytes") {
		t.Fatalf("error = %v", err)
	}
}

func TestPullRejectsAServerError(t *testing.T) {
	body := []byte(sampleDeck)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := Pull(context.Background(), srv.Client(), manifestFor(t, srv.URL+"/", body), Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("a 404 must fail the fetch")
	}
}

func TestVerifyReportsAMissingOrEditedCopy(t *testing.T) {
	body := []byte(sampleDeck)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body) }))
	defer srv.Close()
	m := manifestFor(t, srv.URL+"/", body)

	dir := t.TempDir()
	if _, err := Verify(m, dir); err == nil {
		t.Fatal("Verify accepted an empty directory")
	}

	if _, err := Pull(context.Background(), srv.Client(), m, Options{Dir: dir}); err != nil {
		t.Fatalf("pull: %v", err)
	}
	path := filepath.Join(dir, "wortschatz-a1.yaml")
	if err := os.WriteFile(path, []byte("deck: edited\ncards: []\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Verify(m, dir); err == nil {
		t.Fatal("Verify accepted an edited deck")
	}
}

func TestSelectRejectsAnUnknownLevel(t *testing.T) {
	m, err := Embedded()
	if err != nil {
		t.Fatalf("Embedded: %v", err)
	}
	if _, err := m.Select([]string{"a3"}); err == nil {
		t.Fatal("a typo must not select zero decks silently")
	}
	got, err := m.Select([]string{"A1", "b2"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(got) != 2 || got[0].Level != "a1" || got[1].Level != "b2" {
		t.Fatalf("Select = %+v", got)
	}
}

func TestParseManifestRejectsBadEntries(t *testing.T) {
	good := `{"base_url": "https://example.test/", "decks": [{"file": "a.yaml", "sha256": "%s", "bytes": 1}]}`
	sum := shaOf([]byte("x"))
	cases := map[string]string{
		"no base url":    fmt.Sprintf(`{"decks": [{"file": "a.yaml", "sha256": %q, "bytes": 1}]}`, sum),
		"path in file":   fmt.Sprintf(`{"base_url": "https://e.t/", "decks": [{"file": "../a.yaml", "sha256": %q, "bytes": 1}]}`, sum),
		"no checksum":    `{"base_url": "https://e.t/", "decks": [{"file": "a.yaml", "bytes": 1}]}`,
		"not hex":        `{"base_url": "https://e.t/", "decks": [{"file": "a.yaml", "sha256": "zzzz", "bytes": 1}]}`,
		"no size":        fmt.Sprintf(`{"base_url": "https://e.t/", "decks": [{"file": "a.yaml", "sha256": %q}]}`, sum),
		"no decks":       `{"base_url": "https://e.t/", "decks": []}`,
		"duplicate file": fmt.Sprintf(`{"base_url": "https://e.t/", "decks": [{"file": "a.yaml", "sha256": %q, "bytes": 1}, {"file": "a.yaml", "sha256": %q, "bytes": 1}]}`, sum, sum),
	}
	for name, raw := range cases {
		if _, err := ParseManifest([]byte(raw)); err == nil {
			t.Fatalf("%s: ParseManifest accepted %s", name, raw)
		}
	}
	if _, err := ParseManifest([]byte(fmt.Sprintf(good, sum))); err != nil {
		t.Fatalf("the well-formed manifest was rejected: %v", err)
	}
}
