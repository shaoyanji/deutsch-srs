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

func ReadCSV(r io.Reader) ([]CSVCard, error) {
	cr := csv.NewReader(r)
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}

	out := make([]CSVCard, 0, len(records))
	for _, row := range records {
		if len(row) < 3 {
			continue
		}
		card := CSVCard{Deck: row[0], Front: row[1], Back: row[2]}
		if len(row) > 3 {
			card.NoteType = row[3]
		}
		if len(row) > 4 && row[4] != "" {
			card.Tags = strings.Split(row[4], "|")
		}
		out = append(out, card)
	}
	return out, nil
}

func WriteCSV(w io.Writer, cards []CSVCard) error {
	cw := csv.NewWriter(w)
	for _, c := range cards {
		if err := cw.Write([]string{c.Deck, c.Front, c.Back, c.NoteType, strings.Join(c.Tags, "|")}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
