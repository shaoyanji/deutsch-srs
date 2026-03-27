package importexport

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestYAMLLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "starter.yaml")
	y := `deck: Test
cards:
  - front: Hallo
    back: hello
`
	if err := os.WriteFile(p, []byte(y), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	deck, err := LoadYAML(p)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	if deck.Deck != "Test" {
		t.Fatalf("deck=%s", deck.Deck)
	}
	if len(deck.Cards) != 1 || deck.Cards[0].Front != "Hallo" {
		t.Fatalf("cards not parsed: %+v", deck.Cards)
	}
}

func TestCSVReadWrite(t *testing.T) {
	cards := []CSVCard{{Deck: "A", Front: "eins", Back: "one", Tags: []string{"num", "basic"}}}

	var b bytes.Buffer
	if err := WriteCSV(&b, cards); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	got, err := ReadCSV(&b)
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Front != "eins" || got[0].Back != "one" {
		t.Fatalf("unexpected card: %+v", got[0])
	}
	if len(got[0].Tags) != 2 {
		t.Fatalf("tags=%v", got[0].Tags)
	}
}
