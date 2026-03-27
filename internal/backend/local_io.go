package backend

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/shaoyanji/deutsch-srs/internal/importexport"
	"github.com/shaoyanji/deutsch-srs/internal/store"
)

func (b *LocalBackend) ImportYAML(ctx context.Context, path string) (int, error) {
	deckFile, err := importexport.LoadYAML(path)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, c := range deckFile.Cards {
		_, err := b.store.AddCard(ctx, store.AddCardInput{
			Deck:     deckFile.Deck,
			Front:    c.Front,
			Back:     c.Back,
			NoteType: c.NoteType,
			Tags:     c.Tags,
		}, b.now().UTC())
		if err != nil {
			return count, fmt.Errorf("import yaml card: %w", err)
		}
		count++
	}
	return count, nil
}

func (b *LocalBackend) ImportCSV(ctx context.Context, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	rows, err := importexport.ReadCSV(f)
	if err != nil {
		return 0, err
	}

	count := 0
	for i, r := range rows {
		if strings.TrimSpace(r.Deck) == "" || strings.TrimSpace(r.Front) == "" || strings.TrimSpace(r.Back) == "" {
			return count, fmt.Errorf("invalid csv row %d: deck/front/back are required", i+1)
		}
		_, err := b.store.AddCard(ctx, store.AddCardInput{
			Deck:     r.Deck,
			Front:    r.Front,
			Back:     r.Back,
			NoteType: r.NoteType,
			Tags:     r.Tags,
		}, b.now().UTC())
		if err != nil {
			return count, fmt.Errorf("import csv row %d: %w", i+1, err)
		}
		count++
	}
	return count, nil
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
