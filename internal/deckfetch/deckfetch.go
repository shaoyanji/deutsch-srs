// Package deckfetch downloads the generated Wortschatz decks from the gist that
// holds them. The repository keeps the builder, the command line and the
// checksums; the ~290 KiB of YAML the builder produces stays outside the tree,
// so regenerating a deck no longer rewrites four tracked files in history.
package deckfetch

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed manifest.json
var manifestJSON []byte

// Deck is one generated deck file and the checksum that pins it.
type Deck struct {
	Name   string `json:"name"`
	Level  string `json:"level"`
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
	Cards  int    `json:"cards"`
}

// Manifest points at a gist revision. The raw URL is immutable for that
// revision, and the per-file checksum still catches a re-upload, a truncated
// response or a hand-edited copy — a gist is not a package registry.
type Manifest struct {
	Note     string `json:"note"`
	Gist     string `json:"gist"`
	Revision string `json:"revision"`
	BaseURL  string `json:"base_url"`
	Decks    []Deck `json:"decks"`
}

// Embedded is the manifest compiled into the binary, so `fetch` works from a
// directory that has no checkout in it.
func Embedded() (Manifest, error) {
	return ParseManifest(manifestJSON)
}

func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse deck manifest: %w", err)
	}
	if !strings.HasPrefix(m.BaseURL, "http") {
		return Manifest{}, fmt.Errorf("deck manifest: base_url %q is not an http url", m.BaseURL)
	}
	if !strings.HasSuffix(m.BaseURL, "/") {
		m.BaseURL += "/"
	}
	if len(m.Decks) == 0 {
		return Manifest{}, errors.New("deck manifest: no decks")
	}
	seen := make(map[string]bool, len(m.Decks))
	for i, d := range m.Decks {
		if d.File == "" || filepath.Base(d.File) != d.File {
			return Manifest{}, fmt.Errorf("deck manifest: entry %d has a bad file name %q", i, d.File)
		}
		if seen[d.File] {
			return Manifest{}, fmt.Errorf("deck manifest: %q listed twice", d.File)
		}
		seen[d.File] = true
		if len(d.SHA256) != 64 {
			return Manifest{}, fmt.Errorf("deck manifest: %q has no sha256", d.File)
		}
		if _, err := hex.DecodeString(d.SHA256); err != nil {
			return Manifest{}, fmt.Errorf("deck manifest: %q sha256 is not hex: %w", d.File, err)
		}
		if d.Bytes <= 0 {
			return Manifest{}, fmt.Errorf("deck manifest: %q has no size", d.File)
		}
	}
	return m, nil
}

// URL is the immutable raw link for one deck.
func (m Manifest) URL(d Deck) string {
	return m.BaseURL + d.File
}

// Select filters by level. An unknown level is an error rather than a silent
// no-op, so a typo cannot report a successful fetch of nothing.
func (m Manifest) Select(only []string) ([]Deck, error) {
	want := make(map[string]bool, len(only))
	for _, o := range only {
		o = strings.ToLower(strings.TrimSpace(o))
		if o == "" {
			continue
		}
		found := false
		for _, d := range m.Decks {
			if strings.EqualFold(d.Level, o) {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown level %q (want a1, a2, b1 or b2)", o)
		}
		want[o] = true
	}
	if len(want) == 0 {
		return m.Decks, nil
	}
	out := make([]Deck, 0, len(want))
	for _, d := range m.Decks {
		if want[strings.ToLower(d.Level)] {
			out = append(out, d)
		}
	}
	return out, nil
}

// DefaultDir keeps fetched decks next to the database, in the user's data
// directory, so a checkout is never dirtied by a fetch.
func DefaultDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "deutsch-srs", "decks")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "decks"
	}
	return filepath.Join(home, ".local", "share", "deutsch-srs", "decks")
}

// Options controls a fetch.
type Options struct {
	// Dir is where the files land. Empty means DefaultDir.
	Dir string
	// Force downloads even when the local copy already matches.
	Force bool
	// Only limits the fetch to these levels (a1, a2, b1, b2).
	Only []string
}

// Result is what happened to one deck.
type Result struct {
	Deck    Deck
	Path    string
	Skipped bool
}

// Pull downloads every selected deck that is missing or does not match, and
// returns one result per deck. A partial failure returns what already landed
// along with the error.
func Pull(ctx context.Context, client *http.Client, m Manifest, opts Options) ([]Result, error) {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	decks, err := m.Select(opts.Only)
	if err != nil {
		return nil, err
	}
	dir := opts.Dir
	if dir == "" {
		dir = DefaultDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create %s: %w", dir, err)
	}

	out := make([]Result, 0, len(decks))
	for _, d := range decks {
		path := filepath.Join(dir, d.File)
		if !opts.Force {
			if sum, err := fileSHA(path); err == nil && sum == d.SHA256 {
				out = append(out, Result{Deck: d, Path: path, Skipped: true})
				continue
			}
		}
		if err := fetchOne(ctx, client, m.URL(d), path, d); err != nil {
			return out, err
		}
		out = append(out, Result{Deck: d, Path: path})
	}
	return out, nil
}

func fetchOne(ctx context.Context, client *http.Client, url, path string, d Deck) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", d.File, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", d.File, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", d.File, resp.Status)
	}
	// One byte over the pinned size is enough to prove the body is not the file
	// we published, so a wrong or hostile response cannot fill the disk.
	body, err := io.ReadAll(io.LimitReader(resp.Body, d.Bytes+1))
	if err != nil {
		return fmt.Errorf("%s: %w", d.File, err)
	}
	if int64(len(body)) != d.Bytes {
		return fmt.Errorf("%s: %d bytes, want %d", d.File, len(body), d.Bytes)
	}
	sum := sha256.Sum256(body)
	if got := hex.EncodeToString(sum[:]); got != d.SHA256 {
		return fmt.Errorf("%s: checksum %s, want %s", d.File, got, d.SHA256)
	}

	// Write beside the target and rename, so an interrupted fetch never leaves a
	// half deck where a working one used to be.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".fetch-*")
	if err != nil {
		return fmt.Errorf("%s: %w", d.File, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("%s: %w", d.File, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("%s: %w", d.File, err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("%s: %w", d.File, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("%s: %w", d.File, err)
	}
	return nil
}

// Verify checks the local copies without touching the network.
func Verify(m Manifest, dir string) ([]Result, error) {
	if dir == "" {
		dir = DefaultDir()
	}
	out := make([]Result, 0, len(m.Decks))
	var problems []error
	for _, d := range m.Decks {
		path := filepath.Join(dir, d.File)
		sum, err := fileSHA(path)
		switch {
		case err != nil:
			problems = append(problems, fmt.Errorf("%s: %w", d.File, err))
		case sum != d.SHA256:
			problems = append(problems, fmt.Errorf("%s: checksum %s, want %s", d.File, sum, d.SHA256))
		default:
			out = append(out, Result{Deck: d, Path: path})
		}
	}
	return out, errors.Join(problems...)
}

func fileSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
