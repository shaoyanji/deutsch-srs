package backend

import (
	"context"
	"os"
	"strings"

	"github.com/shaoyanji/deutsch-srs/internal/importexport"
	"github.com/shaoyanji/deutsch-srs/internal/store"
)

// ImportOptions controls a deck-file import.
type ImportOptions struct {
	DeckOverride string
	Format       string
	SkipExisting bool
	DryRun       bool
}

// ImportResult reports what one file contributed.
type ImportResult struct {
	File    string
	Deck    string
	Added   int
	Skipped int
	Invalid int
	DryRun  bool
}

// ImportFile loads a yaml, csv or tsv deck file and writes it in one transaction.
func (b *LocalBackend) ImportFile(ctx context.Context, path string, opts ImportOptions) (ImportResult, error) {
	deckFile, err := importexport.LoadDeckFile(path, importexport.LoadOptions{
		DeckOverride: opts.DeckOverride,
		Format:       opts.Format,
	})
	if err != nil {
		return ImportResult{File: path}, err
	}

	inputs := make([]store.AddCardInput, 0, len(deckFile.Cards))
	for _, c := range deckFile.Cards {
		deck := strings.TrimSpace(c.Deck)
		if deck == "" {
			deck = deckFile.Deck
		}
		inputs = append(inputs, store.AddCardInput{
			Deck:     deck,
			Front:    c.Front,
			Back:     c.Back,
			NoteType: c.NoteType,
			Tags:     c.Tags,
		})
	}

	bulk, err := b.store.AddCardsBulk(ctx, inputs, b.now().UTC(), store.BulkOptions{
		SkipExisting: opts.SkipExisting,
		DryRun:       opts.DryRun,
	})
	if err != nil {
		return ImportResult{File: path, Deck: deckFile.Deck}, err
	}

	return ImportResult{
		File:    path,
		Deck:    deckFile.Deck,
		Added:   bulk.Added,
		Skipped: bulk.Skipped,
		Invalid: bulk.Invalid + deckFile.Skipped,
		DryRun:  opts.DryRun,
	}, nil
}

// ImportYAML imports a deck file, skipping cards already present.
func (b *LocalBackend) ImportYAML(ctx context.Context, path string) (int, error) {
	res, err := b.ImportFile(ctx, path, ImportOptions{SkipExisting: true})
	return res.Added, err
}

// ImportCSV imports a deck file, skipping cards already present.
func (b *LocalBackend) ImportCSV(ctx context.Context, path string) (int, error) {
	res, err := b.ImportFile(ctx, path, ImportOptions{SkipExisting: true})
	return res.Added, err
}

func (b *LocalBackend) ExportCSV(ctx context.Context, path string) (int, error) {
	cards, err := b.store.AllCards(ctx)
	if err != nil {
		return 0, err
	}
	out := make([]importexport.CSVCard, 0, len(cards))
	for _, c := range cards {
		out = append(out, importexport.CSVCard{
			Deck:     c.Deck,
			Front:    c.Front,
			Back:     c.Back,
			NoteType: c.NoteType,
			Tags:     c.Tags,
		})
	}
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if err := importexport.WriteCSV(f, out); err != nil {
		return 0, err
	}
	return len(out), nil
}
