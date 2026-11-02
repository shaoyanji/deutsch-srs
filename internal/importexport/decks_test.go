package importexport

import (
	"path/filepath"
	"strings"
	"testing"
)

// Every deck shipped in decks/ must stay loadable and card-complete: a broken
// deck is content, not code, so nothing else would catch it.
func TestBundledDecksLoad(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "decks", "*.yaml"))
	if err != nil {
		t.Fatalf("glob decks: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no bundled decks found")
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			deck, err := LoadDeckFile(path, LoadOptions{})
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if strings.TrimSpace(deck.Deck) == "" {
				t.Fatal("deck name is empty")
			}
			if len(deck.Cards) == 0 {
				t.Fatal("deck has no cards")
			}

			seen := make(map[string]bool, len(deck.Cards))
			for i, card := range deck.Cards {
				if strings.TrimSpace(card.Front) == "" {
					t.Fatalf("card %d: empty front", i+1)
				}
				if strings.TrimSpace(card.Back) == "" {
					t.Fatalf("card %d (%q): empty back", i+1, card.Front)
				}
				if strings.TrimSpace(card.Deck) == "" {
					t.Fatalf("card %d (%q): empty deck", i+1, card.Front)
				}
				if len(card.Tags) == 0 {
					t.Fatalf("card %d (%q): no tags", i+1, card.Front)
				}
				key := strings.ToLower(card.Front)
				if seen[key] {
					t.Fatalf("card %d: duplicate front %q", i+1, card.Front)
				}
				seen[key] = true
			}
		})
	}
}

// The writing deck is German learning material, so it must carry real umlauts
// rather than the ae/oe/ue fallback.
func TestWritingDeckUsesRealUmlauts(t *testing.T) {
	deck, err := LoadDeckFile(filepath.Join("..", "..", "decks", "b2-schreiben.yaml"), LoadOptions{})
	if err != nil {
		t.Fatalf("load writing deck: %v", err)
	}

	// German words that genuinely contain ae/oe/ue: qu and eu spellings.
	allowed := map[string]bool{
		"Konsequenz": true, "Quelle": true, "Abenteuer": true,
		"freue": true, "freuen": true, "neue": true, "neuen": true,
	}
	for _, card := range deck.Cards {
		for _, word := range strings.Fields(strings.NewReplacer(",", " ", ".", " ", "(", " ", ")", " ", ":", " ", "-", " ", "/", " ").Replace(card.Front + " " + card.Back)) {
			word = strings.Trim(word, "\"[]!?")
			if word == "" || allowed[word] {
				continue
			}
			if strings.ContainsAny(word, "äöüÄÖÜß") {
				continue
			}
			lower := strings.ToLower(word)
			if !strings.Contains(lower, "ae") && !strings.Contains(lower, "oe") && !strings.Contains(lower, "ue") {
				continue
			}
			t.Fatalf("card %q: %q looks like a transliterated umlaut in a German text", card.Front, word)
		}
	}
}
