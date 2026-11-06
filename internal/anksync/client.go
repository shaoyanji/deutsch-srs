// Package anksync talks to a self-hosted anki-sync-server with the protocol
// real Anki clients speak (sync version 11: one `anki-sync` JSON header plus a
// zstd-compressed body in each direction, see rslib/src/sync/version.rs).
//
// Only the read path is implemented: login (sync/hostKey), server metadata
// (sync/meta) and a full collection download (sync/download). The incremental
// merge (start/chunk/finish) and the upload path are deliberately absent, so
// nothing in this package can overwrite the server's collection.
package anksync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

// syncVersion is the protocol version the server is guaranteed to accept
// (SYNC_VERSION_MAX upstream). Version 11 is the one that switched from
// multipart+gzip to the header+zstd framing used here.
const syncVersion = 11

// maxResponseBytes bounds a single response body. The collection of a personal
// deck set is megabytes; anything past this is a broken server, not a big file.
const maxResponseBytes = 1 << 30

var (
	zstdEncoder, _ = zstd.NewWriter(nil)
	zstdDecoder, _ = zstd.NewReader(nil)
)

// Client is a sync-protocol client for one server. It is not safe for
// concurrent use: hkey is set by HostKey and read by every later call.
type Client struct {
	BaseURL    string
	ClientVer  string
	HTTPClient *http.Client
	// SessionKey identifies one sync attempt; the server only checks it on the
	// stateful methods (start/finish), which this client never calls.
	SessionKey string

	hkey string
}

func NewClient(baseURL, clientVer string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		ClientVer:  clientVer,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
		SessionKey: uuid.NewString(),
	}
}

// header is the `anki-sync` request header. Field names are the wire names
// from rslib's SyncHeader, not free choices.
type header struct {
	Version   int    `json:"v"`
	Key       string `json:"k"`
	ClientVer string `json:"c"`
	Session   string `json:"s"`
}

// Meta is the subset of SyncMeta this client uses.
type Meta struct {
	Modified int64  `json:"mod"`
	Schema   int64  `json:"scm"`
	Usn      int32  `json:"usn"`
	Time     int64  `json:"ts"`
	Message  string `json:"msg"`
	Continue bool   `json:"cont"`
	Empty    bool   `json:"empty"`
	MediaUsn int32  `json:"media_usn"`
	HostNum  int32  `json:"hostNum"`
}

// HostKey logs in and remembers the host key every later call needs.
func (c *Client) HostKey(ctx context.Context, user, pass string) error {
	var out struct {
		Key string `json:"key"`
	}
	if err := c.call(ctx, "hostKey", map[string]string{"u": user, "p": pass}, &out); err != nil {
		return err
	}
	if out.Key == "" {
		return fmt.Errorf("sync hostKey: server returned an empty key")
	}
	c.hkey = out.Key
	return nil
}

// Meta asks the server what state its collection is in. The body is
// MetaRequest (v/cv), not empty: serde rejects a missing v before auth is even
// looked at, which is why a malformed meta answers a bare 400.
func (c *Client) Meta(ctx context.Context) (Meta, error) {
	var out Meta
	in := map[string]any{"v": syncVersion, "cv": c.ClientVer}
	if err := c.call(ctx, "meta", in, &out); err != nil {
		return Meta{}, err
	}
	return out, nil
}

// Download fetches the server's whole collection as an .anki2 SQLite file.
func (c *Client) Download(ctx context.Context) ([]byte, error) {
	return c.callRaw(ctx, "download", struct{}{})
}

func (c *Client) call(ctx context.Context, method string, in, out any) error {
	raw, err := c.callRaw(ctx, method, in)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("sync %s: decode response: %w", method, err)
	}
	return nil
}

// callRaw posts one sync method and returns the decompressed payload. Every
// body on this protocol is zstd, including the ones that carry JSON.
func (c *Client) callRaw(ctx context.Context, method string, in any) ([]byte, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("sync %s: encode request: %w", method, err)
	}
	hdr, err := json.Marshal(header{
		Version:   syncVersion,
		Key:       c.hkey,
		ClientVer: c.ClientVer,
		Session:   c.SessionKey,
	})
	if err != nil {
		return nil, fmt.Errorf("sync %s: encode header: %w", method, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/sync/"+method,
		bytes.NewReader(zstdEncoder.EncodeAll(body, nil)))
	if err != nil {
		return nil, fmt.Errorf("sync %s: build request: %w", method, err)
	}
	req.Header.Set("anki-sync", string(hdr))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sync %s: %w", method, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("sync %s: read response: %w", method, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sync %s: %s%s", method, resp.Status, bodySnippet(raw))
	}
	out, err := zstdDecoder.DecodeAll(raw, nil)
	if err != nil {
		return nil, fmt.Errorf("sync %s: decode zstd body: %w", method, err)
	}
	return out, nil
}

// bodySnippet adds the server's short error text, which is what distinguishes
// "wrong password" from "unknown method" on this protocol.
func bodySnippet(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return ""
	}
	if len(text) > 200 {
		text = text[:200] + "…"
	}
	return ": " + text
}
