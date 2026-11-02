package importexport

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

func TestLoadDeckFileTwoColumnSemicolon(t *testing.T) {
	p := writeFile(t, "liste.csv", "Hund;dog\nKatze;cat\n")

	got, err := LoadDeckFile(p, LoadOptions{DeckOverride: "Wortschatz"})
	if err != nil {
		t.Fatalf("LoadDeckFile: %v", err)
	}
	if len(got.Cards) != 2 {
		t.Fatalf("cards=%d want=2", len(got.Cards))
	}
	first := got.Cards[0]
	if first.Deck != "Wortschatz" || first.Front != "Hund" || first.Back != "dog" {
		t.Fatalf("unexpected card: %+v", first)
	}
}

func TestLoadDeckFileNamedGermanHeader(t *testing.T) {
	p := writeFile(t, "b2.csv", "wort;übersetzung;thema\nHund;dog;tiere\n")

	got, err := LoadDeckFile(p, LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDeckFile: %v", err)
	}
	if len(got.Cards) != 1 {
		t.Fatalf("cards=%d want=1", len(got.Cards))
	}
	card := got.Cards[0]
	if card.Front != "Hund" || card.Back != "dog" {
		t.Fatalf("header not mapped: %+v", card)
	}
	if len(card.Tags) != 1 || card.Tags[0] != "tiere" {
		t.Fatalf("tags=%v", card.Tags)
	}
}

func TestLoadDeckFileSkipsHeaderCommentsAndBadRows(t *testing.T) {
	p := writeFile(t, "mixed.csv", "deck,front,back\nA,eins,one\n# not a card\nA,,missing front\n")
	got, err := LoadDeckFile(p, LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDeckFile: %v", err)
	}
	if len(got.Cards) != 1 {
		t.Fatalf("cards=%d want=1 (%+v)", len(got.Cards), got.Cards)
	}
	if got.Skipped != 2 {
		t.Fatalf("skipped=%d want=2", got.Skipped)
	}
}

func TestLoadDeckFileTabSeparated(t *testing.T) {
	p := writeFile(t, "liste.tsv", "Hund\tdog\n")
	got, err := LoadDeckFile(p, LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDeckFile: %v", err)
	}
	if len(got.Cards) != 1 || got.Cards[0].Front != "Hund" || got.Cards[0].Back != "dog" {
		t.Fatalf("unexpected cards: %+v", got.Cards)
	}
}

func TestLoadDeckFileYAML(t *testing.T) {
	p := writeFile(t, "deck.yaml", "deck: Basics\ncards:\n  - front: Hallo\n    back: hello\n    tags: [greeting]\n")
	got, err := LoadDeckFile(p, LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDeckFile: %v", err)
	}
	if got.Deck != "Basics" || len(got.Cards) != 1 || got.Cards[0].Back != "hello" {
		t.Fatalf("unexpected deck: %+v", got)
	}

	renamed, err := LoadDeckFile(p, LoadOptions{DeckOverride: "B2 Redemittel"})
	if err != nil {
		t.Fatalf("LoadDeckFile override: %v", err)
	}
	if renamed.Cards[0].Deck != "B2 Redemittel" {
		t.Fatalf("deck override ignored: %+v", renamed.Cards[0])
	}
}

func TestLoadDeckFileExplicitFormatBeatsExtension(t *testing.T) {
	p := writeFile(t, "liste.csv", "Hund;dog\n")
	got, err := LoadDeckFile(p, LoadOptions{Format: "csv", DeckOverride: "D"})
	if err != nil {
		t.Fatalf("LoadDeckFile: %v", err)
	}
	if len(got.Cards) != 1 {
		t.Fatalf("cards=%d want=1", len(got.Cards))
	}
}
