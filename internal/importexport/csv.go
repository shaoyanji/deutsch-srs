package importexport

import (
	"encoding/csv"
	"io"
	"strings"
)

type CSVCard struct {
	Deck     string
	Front    string
	Back     string
	NoteType string
	Tags     []string
}

var csvHeader = []string{"deck", "front", "back", "note_type", "tags"}

// ReadCSV reads the app's own column order and tolerates a header row.
func ReadCSV(r io.Reader) ([]CSVCard, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}

	start := 0
	if len(records) > 0 && looksLikeHeader(records[0]) {
		start = 1
	}

	out := make([]CSVCard, 0, len(records)-start)
	for _, row := range records[start:] {
		card, ok := mapPositionalRow(row, "")
		if !ok {
			continue
		}
		out = append(out, CSVCard{
			Deck:     card.Deck,
			Front:    card.Front,
			Back:     card.Back,
			NoteType: card.NoteType,
			Tags:     card.Tags,
		})
	}
	return out, nil
}

// WriteCSV writes a header plus one row per card, so a round-trip is self-describing.
func WriteCSV(w io.Writer, cards []CSVCard) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(csvHeader); err != nil {
		return err
	}
	for _, c := range cards {
		if err := cw.Write([]string{c.Deck, c.Front, c.Back, c.NoteType, strings.Join(c.Tags, "|")}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
