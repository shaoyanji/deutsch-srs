package anksync

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shaoyanji/deutsch-srs/internal/backend"
	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

// ErrReadOnly is what every write returns. The backend pulls the collection and
// reads it; pushing changes back needs the incremental merge protocol
// (start/chunk/finish), which this package does not implement. Failing loudly
// is the point: a grade that only existed locally would be erased by the next
// download.
var ErrReadOnly = errors.New("anki-sync is read-only: reviews are not written back to the server")

// Config is what the TUI passes in.
type Config struct {
	URL           string // e.g. http://anki.frieren.lan
	User          string
	Password      string
	CachePath     string // empty means ~/.cache/deutsch-srs/<host>.anki2
	ClientVersion string
}

// Backend serves reviews from a snapshot of the sync server's collection.
// The download happens once, on first use, and is kept for the session.
type Backend struct {
	cfg Config
	now func() time.Time

	mu   sync.Mutex
	col  *Collection
	meta Meta
}

func NewBackend(cfg Config) *Backend {
	return &Backend{cfg: cfg, now: time.Now}
}

func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.col == nil {
		return nil
	}
	err := b.col.Close()
	b.col = nil
	return err
}

// Health is the reachability gate: it performs the download if it has not
// happened yet, so a bad URL or password surfaces before the review screen.
func (b *Backend) Health(ctx context.Context) error {
	_, err := b.snapshot(ctx)
	return err
}

func (b *Backend) snapshot(ctx context.Context) (*Collection, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.col != nil {
		return b.col, nil
	}

	client := NewClient(b.cfg.URL, b.cfg.ClientVersion)
	if err := client.HostKey(ctx, b.cfg.User, b.cfg.Password); err != nil {
		return nil, err
	}
	meta, err := client.Meta(ctx)
	if err != nil {
		return nil, err
	}
	data, err := client.Download(ctx)
	if err != nil {
		return nil, err
	}
	cachePath, err := b.cachePath()
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(cachePath, data); err != nil {
		return nil, err
	}
	col, err := OpenCollection(cachePath)
	if err != nil {
		return nil, err
	}
	b.col, b.meta = col, meta
	return col, nil
}

func (b *Backend) cachePath() (string, error) {
	if b.cfg.CachePath != "" {
		return b.cfg.CachePath, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate cache dir: %w", err)
	}
	host := "anki-sync"
	if u, err := url.Parse(b.cfg.URL); err == nil && u.Hostname() != "" {
		host = strings.NewReplacer(":", "_", "/", "_").Replace(u.Hostname())
	}
	return filepath.Join(dir, "deutsch-srs", host+".anki2"), nil
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ankisync-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp collection: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write collection: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("install collection: %w", err)
	}
	return nil
}

func (b *Backend) ListDecks(ctx context.Context) ([]srs.Deck, error) {
	col, err := b.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return col.Decks(ctx, b.now())
}

func (b *Backend) DueCards(ctx context.Context, deck string, limit int) ([]srs.Card, error) {
	col, err := b.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	return col.DueCards(ctx, deck, limit, b.now())
}

// ReviewQueue hands out due reviews first and then new cards, which is the
// order Anki's own scheduler would use. The server owns the daily allowance, so
// NewPerDay is reported but not enforced here.
func (b *Backend) ReviewQueue(ctx context.Context, deck string, limit int) (backend.Queue, error) {
	col, err := b.snapshot(ctx)
	if err != nil {
		return backend.Queue{}, err
	}
	now := b.now()
	if limit <= 0 {
		limit = 50
	}
	due, err := col.DueCards(ctx, deck, limit, now)
	if err != nil {
		return backend.Queue{}, err
	}
	q := backend.Queue{Cards: due}
	if room := limit - len(due); room > 0 {
		fresh, err := col.NewCards(deck, room, now)
		if err != nil {
			return backend.Queue{}, err
		}
		q.Cards = append(q.Cards, fresh...)
		q.NewRemaining = room - len(fresh)
	}
	return q, nil
}

// SearchCards matches the query against front, back, deck and tags. The
// collection is small enough to filter in memory, which avoids reimplementing
// Anki's search grammar badly.
func (b *Backend) SearchCards(ctx context.Context, query string, limit int) ([]srs.Card, error) {
	col, err := b.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	now := b.now()
	needle := strings.ToLower(strings.TrimSpace(query))
	cards, err := col.DueCards(ctx, "", 0, now)
	if err != nil {
		return nil, err
	}
	newCards, err := col.NewCards("", 0, now)
	if err != nil {
		return nil, err
	}
	out := make([]srs.Card, 0, len(cards)+len(newCards))
	for _, c := range append(cards, newCards...) {
		if needle == "" || cardMatches(c, needle) {
			out = append(out, c)
		}
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func cardMatches(c srs.Card, needle string) bool {
	if strings.Contains(strings.ToLower(c.Front), needle) ||
		strings.Contains(strings.ToLower(c.Back), needle) ||
		strings.Contains(strings.ToLower(c.Deck), needle) {
		return true
	}
	for _, t := range c.Tags {
		if strings.Contains(strings.ToLower(t), needle) {
			return true
		}
	}
	return false
}

func (b *Backend) Stats(ctx context.Context) (backend.Stats, error) {
	col, err := b.snapshot(ctx)
	if err != nil {
		return backend.Stats{BackendName: "anki-sync", HealthErrorText: err.Error()}, nil
	}
	now := b.now()
	decks, err := col.Decks(ctx, now)
	if err != nil {
		return backend.Stats{}, err
	}
	stats := backend.Stats{BackendName: "anki-sync", BackendHealthy: true}
	for _, d := range decks {
		stats.DueToday += d.DueCount
		stats.NewCards += d.NewCount
	}
	stats.TotalCards = stats.DueToday + stats.NewCards
	reviews, err := col.ReviewsToday(now)
	if err != nil {
		return backend.Stats{}, err
	}
	stats.ReviewsToday = reviews
	return stats, nil
}

func (b *Backend) AddCard(context.Context, backend.AddCardInput) (srs.Card, error) {
	return srs.Card{}, ErrReadOnly
}

func (b *Backend) AnswerCard(context.Context, string, srs.ReviewGrade) error {
	return ErrReadOnly
}

func (b *Backend) SuspendCard(context.Context, string) error   { return ErrReadOnly }
func (b *Backend) UnsuspendCard(context.Context, string) error { return ErrReadOnly }
func (b *Backend) BuryCard(context.Context, string) error      { return ErrReadOnly }
func (b *Backend) UndoReview(context.Context, backend.ReviewUndo) error {
	return ErrReadOnly
}
