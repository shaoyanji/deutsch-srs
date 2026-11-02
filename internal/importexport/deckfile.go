package importexport

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImportCard is a format-independent card ready for the store.
type ImportCard struct {
	Deck     string
	Front    string
	Back     string
	NoteType string
	Tags     []string
}

// ImportFile is a whole deck file, already mapped to cards.
type ImportFile struct {
	Deck    string
	Cards   []ImportCard
	Skipped int
}

// LoadOptions tunes deck-file parsing. Format is "auto"/"" (sniff), "yaml", "csv" or "tsv".
type LoadOptions struct {
	DeckOverride string
	Format       string
}

var headerTokens = map[string]bool{
	"deck": true, "front": true, "back": true, "note_type": true, "tags": true,
	"word": true, "wort": true, "german": true, "deutsch": true, "englisch": true,
	"english": true, "translation": true, "uebersetzung": true, "übersetzung": true,
	"meaning": true, "definition": true, "vorderseite": true, "rückseite": true,
	"rueckseite": true, "term": true, "thema": true, "kategorie": true,
}

var (
	frontColumns = []string{"front", "wort", "word", "german", "deutsch", "vorderseite", "term", "begriff"}
	backColumns  = []string{"back", "translation", "uebersetzung", "übersetzung", "english", "englisch",
		"meaning", "definition", "bedeutung", "rückseite", "rueckseite"}
	deckColumns = []string{"deck", "stapel", "kategorie", "folder", "level"}
	tagColumns  = []string{"tags", "tag", "thema", "topic"}
	typeColumns = []string{"note_type", "type", "typ", "notetype"}
)

// LoadDeckFile reads a deck from yaml, csv or tsv, sniffing the format when asked to.
func LoadDeckFile(path string, opts LoadOptions) (ImportFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ImportFile{}, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return ImportFile{}, fmt.Errorf("%s: file is empty", path)
	}

	switch resolveFormat(path, raw, opts.Format) {
	case "yaml":
		deck, err := LoadYAML(path)
		if err != nil {
			return ImportFile{}, err
		}
		name := strings.TrimSpace(opts.DeckOverride)
		if name == "" {
			name = deck.Deck
		}
		out := ImportFile{Deck: name, Cards: make([]ImportCard, 0, len(deck.Cards))}
		for _, c := range deck.Cards {
			out.Cards = append(out.Cards, ImportCard{
				Deck:     name,
				Front:    strings.TrimSpace(c.Front),
				Back:     strings.TrimSpace(c.Back),
				NoteType: c.NoteType,
				Tags:     c.Tags,
			})
		}
		return out, nil
	case "csv", "tsv":
		delim := '\t'
		if resolveFormat(path, raw, opts.Format) != "tsv" {
			delim = sniffDelimiter(raw)
		}
		return loadDelimited(raw, path, delim, opts.DeckOverride)
	default:
		return ImportFile{}, fmt.Errorf("%s: unsupported deck file format %q", path, opts.Format)
	}
}

func loadDelimited(raw []byte, path string, delim rune, deckOverride string) (ImportFile, error) {
	r := csv.NewReader(bytes.NewReader(raw))
	r.Comma = delim
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return ImportFile{}, fmt.Errorf("parse %s: %w", path, err)
	}

	out := ImportFile{Deck: defaultDeckName(path, deckOverride)}
	start := 0
	var cols columnMap
	if len(records) > 0 {
		if m, ok := buildColumnMap(records[0]); ok {
			cols = m
			start = 1
		} else if looksLikeHeader(records[0]) {
			start = 1
		}
	}

	out.Cards = make([]ImportCard, 0, len(records)-start)
	for _, row := range records[start:] {
		card, ok := mapRow(row, cols, out.Deck)
		if !ok {
			out.Skipped++
			continue
		}
		out.Cards = append(out.Cards, card)
	}
	return out, nil
}

type columnMap struct {
	deck     int
	front    int
	back     int
	noteType int
	tags     int
	ok       bool
}

func buildColumnMap(header []string) (columnMap, bool) {
	m := columnMap{deck: -1, front: -1, back: -1, noteType: -1, tags: -1}
	for i, cell := range header {
		name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(cell), "*"))
		if name == "" {
			continue
		}
		switch {
		case m.front < 0 && containsToken(frontColumns, name):
			m.front = i
		case m.back < 0 && containsToken(backColumns, name):
			m.back = i
		case m.deck < 0 && containsToken(deckColumns, name):
			m.deck = i
		case m.noteType < 0 && containsToken(typeColumns, name):
			m.noteType = i
		case m.tags < 0 && containsToken(tagColumns, name):
			m.tags = i
		}
	}
	m.ok = m.front >= 0 && m.back >= 0
	return m, m.ok
}

func mapRow(row []string, cols columnMap, defaultDeck string) (ImportCard, bool) {
	if len(row) > 0 {
		first := strings.TrimSpace(row[0])
		if first == "" || strings.HasPrefix(first, "#") {
			return ImportCard{}, false
		}
	}
	if !cols.ok {
		return mapPositionalRow(row, defaultDeck)
	}
	front := pick(row, cols.front)
	back := pick(row, cols.back)
	if front == "" || back == "" {
		return ImportCard{}, false
	}
	deck := pick(row, cols.deck)
	if deck == "" {
		deck = defaultDeck
	}
	return ImportCard{
		Deck:     deck,
		Front:    front,
		Back:     back,
		NoteType: pick(row, cols.noteType),
		Tags:     splitTags(pick(row, cols.tags)),
	}, true
}

// mapPositionalRow handles files with no named columns: deck,front,back[,note_type[,tags]].
// A two-column file is treated as front,back and takes the caller's deck.
func mapPositionalRow(row []string, defaultDeck string) (ImportCard, bool) {
	cells := make([]string, len(row))
	for i, c := range row {
		cells[i] = strings.TrimSpace(c)
	}
	switch {
	case len(cells) >= 5:
		if cells[0] == "" || cells[1] == "" || cells[2] == "" {
			return ImportCard{}, false
		}
		return ImportCard{
			Deck:     cells[0],
			Front:    cells[1],
			Back:     cells[2],
			NoteType: cells[3],
			Tags:     splitTags(cells[4]),
		}, true
	case len(cells) == 4:
		if cells[0] == "" || cells[1] == "" || cells[2] == "" {
			return ImportCard{}, false
		}
		return ImportCard{Deck: cells[0], Front: cells[1], Back: cells[2], NoteType: cells[3]}, true
	case len(cells) == 3:
		deck := cells[0]
		if deck == "" {
			deck = defaultDeck
		}
		if cells[1] == "" || cells[2] == "" {
			return ImportCard{}, false
		}
		return ImportCard{Deck: deck, Front: cells[1], Back: cells[2]}, true
	case len(cells) == 2:
		if cells[0] == "" || cells[1] == "" {
			return ImportCard{}, false
		}
		return ImportCard{Deck: defaultDeck, Front: cells[0], Back: cells[1]}, true
	}
	return ImportCard{}, false
}

func pick(cells []string, idx int) string {
	if idx < 0 || idx >= len(cells) {
		return ""
	}
	return strings.TrimSpace(cells[idx])
}

func splitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == '|' || r == ',' || r == ';' })
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

func containsToken(tokens []string, name string) bool {
	for _, t := range tokens {
		if t == name {
			return true
		}
	}
	return false
}

func looksLikeHeader(row []string) bool {
	if len(row) == 0 {
		return false
	}
	first := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(row[0]), "*"))
	return headerTokens[first]
}

func defaultDeckName(path, override string) string {
	if ov := strings.TrimSpace(override); ov != "" {
		return ov
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func resolveFormat(path string, raw []byte, explicit string) string {
	switch strings.ToLower(strings.TrimSpace(explicit)) {
	case "yaml", "yml":
		return "yaml"
	case "csv":
		return "csv"
	case "tsv", "tab":
		return "tsv"
	case "", "auto":
	default:
		return "unknown"
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return "yaml"
	case ".csv":
		return "csv"
	case ".tsv", ".tab":
		return "tsv"
	}
	return sniffFormat(raw)
}

func sniffFormat(raw []byte) string {
	first := firstContentLine(raw)
	low := strings.ToLower(first)
	if strings.HasPrefix(low, "deck:") || strings.HasPrefix(low, "cards:") || strings.HasPrefix(low, "- front:") {
		return "yaml"
	}
	if strings.Contains(first, "\t") {
		return "tsv"
	}
	if strings.Contains(first, ";") && !strings.Contains(first, ",") {
		return "csv"
	}
	return "csv"
}

// sniffDelimiter picks the field separator a delimited file actually uses.
// German lists are commonly semicolon-separated, exports often tab-separated.
func sniffDelimiter(raw []byte) rune {
	line := firstContentLine(raw)
	counts := []struct {
		delim rune
		n     int
	}{
		{'\t', strings.Count(line, "\t")},
		{';', strings.Count(line, ";")},
		{',', strings.Count(line, ",")},
	}
	best := ','
	bestN := 0
	for _, c := range counts {
		if c.n > bestN {
			best = c.delim
			bestN = c.n
		}
	}
	return best
}

func firstContentLine(raw []byte) string {
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}
