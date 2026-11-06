package anksync

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveServer exercises a real anki-sync-server. It is skipped unless
// DEUTSCH_SRS_SYNC_URL is set, so `go test ./...` stays offline:
//
//	DEUTSCH_SRS_SYNC_URL=http://anki.frieren.lan \
//	DEUTSCH_SRS_SYNC_USER=devji \
//	DEUTSCH_SRS_SYNC_PASS=... go test ./internal/anksync -run Live -v
func TestLiveServer(t *testing.T) {
	baseURL := os.Getenv("DEUTSCH_SRS_SYNC_URL")
	if baseURL == "" {
		t.Skip("set DEUTSCH_SRS_SYNC_URL to test a live anki-sync-server")
	}
	user := os.Getenv("DEUTSCH_SRS_SYNC_USER")
	pass := os.Getenv("DEUTSCH_SRS_SYNC_PASS")
	if user == "" || pass == "" {
		t.Fatal("DEUTSCH_SRS_SYNC_USER and DEUTSCH_SRS_SYNC_PASS are required with a URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cfg := Config{
		URL:           baseURL,
		User:          user,
		Password:      pass,
		CachePath:     t.TempDir() + "/live.anki2",
		ClientVersion: "deutsch-srs/live-test",
	}
	be := NewBackend(cfg)
	defer be.Close()

	if err := be.Health(ctx); err != nil {
		t.Fatalf("login+download: %v", err)
	}

	decks, err := be.ListDecks(ctx)
	if err != nil {
		t.Fatalf("list decks: %v", err)
	}
	if len(decks) == 0 {
		t.Fatal("server collection has no decks")
	}
	for _, d := range decks {
		t.Logf("deck %-24s due=%-4d new=%-4d", d.Name, d.DueCount, d.NewCount)
	}

	stats, err := be.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	t.Logf("stats backend=%s total=%d due=%d new=%d reviewedToday=%d",
		stats.BackendName, stats.TotalCards, stats.DueToday, stats.NewCards, stats.ReviewsToday)

	first := decks[0].Name
	q, err := be.ReviewQueue(ctx, first, 5)
	if err != nil {
		t.Fatalf("queue %s: %v", first, err)
	}
	t.Logf("queue[%s] %d cards", first, len(q.Cards))
	for i, c := range q.Cards {
		if i >= 3 {
			break
		}
		t.Logf("  %s | %q -> %q due=%s", c.ID, c.Front, c.Back, c.DueAt.Format(time.RFC3339))
	}

	if err := be.AnswerCard(ctx, "1", 3); err != ErrReadOnly {
		t.Errorf("AnswerCard err = %v, want ErrReadOnly", err)
	}
}
