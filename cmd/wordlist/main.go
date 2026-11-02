// Command wordlist builds a vocabulary deck from a kaikki.org Wiktionary extract
// and an OpenSubtitles frequency list, which is how the bundled A1-B2 word decks
// are generated. See README.md for the download commands.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shaoyanji/deutsch-srs/internal/importexport"
	"github.com/shaoyanji/deutsch-srs/internal/wordlist"
)

func main() {
	var (
		kaikki     = flag.String("kaikki", "", "path to a kaikki.org JSONL extract (e.g. kaikki.org-dictionary-German.jsonl)")
		freq       = flag.String("freq", "", "path to an OpenSubtitles frequency list (word count per line, most frequent first)")
		out        = flag.String("out", "", "output deck file (.yaml or .csv)")
		deck       = flag.String("deck", "", "deck name stored in the deck file")
		level      = flag.String("level", "", "level band to build: a1, a2, b1 or b2 (default: every ranked lemma)")
		cumulative = flag.Bool("cumulative", false, "with -level, keep every word up to the level instead of only that level's band")
		pos        = flag.String("pos", strings.Join(wordlist.DefaultPos, ","), "comma-separated kaikki parts of speech to keep")
		limit      = flag.Int("limit", 0, "maximum number of cards (0 = all)")
		minRank    = flag.Int("min-rank", 0, "drop lemmas more frequent than this rank (0 = none)")
		maxRank    = flag.Int("max-rank", 0, "drop lemmas rarer than this rank (0 = no cap)")
		maxGlos    = flag.Int("max-gloss", 120, "trim translations to this many characters")
		unrank     = flag.Bool("allow-unranked", false, "keep lemmas the frequency list has never seen")
		stat       = flag.Bool("stats", true, "print drop counts and a sample")
	)
	flag.Parse()

	if *kaikki == "" || *freq == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: wordlist -kaikki <extract.jsonl> -freq <de_full.txt> -out <deck.yaml> [-deck NAME] [-level b1]")
		flag.PrintDefaults()
		os.Exit(2)
	}

	deckName := *deck
	if deckName == "" {
		deckName = "Wortschatz"
		if *level != "" {
			deckName += " " + strings.ToUpper(*level)
		}
	}

	floor, ceiling := *minRank, *maxRank
	if *level != "" {
		levelCeiling := wordlist.LevelLimit(*level)
		if levelCeiling == 0 {
			fmt.Fprintf(os.Stderr, "unknown level %q (want a1, a2, b1 or b2)\n", *level)
			os.Exit(2)
		}
		if ceiling == 0 || levelCeiling < ceiling {
			ceiling = levelCeiling
		}
		// Per-level decks are bands: A2 holds the words A1 did not already teach,
		// so importing all four does not drill the same word four times.
		if !*cumulative {
			if levelFloor := wordlist.LevelFloor(*level); levelFloor+1 > floor {
				floor = levelFloor + 1
			}
		}
	}

	freqFile, err := os.Open(*freq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open frequency list: %v\n", err)
		os.Exit(1)
	}
	ranks, err := wordlist.LoadFrequency(freqFile)
	freqFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "read frequency list: %v\n", err)
		os.Exit(1)
	}

	kaikkiFile, err := os.Open(*kaikki)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open kaikki extract: %v\n", err)
		os.Exit(1)
	}
	cards, stats, err := wordlist.Build(kaikkiFile, ranks, wordlist.Options{
		Pos:           splitList(*pos),
		MinRank:       floor,
		MaxRank:       ceiling,
		AllowUnranked: *unrank,
		Limit:         *limit,
		MaxGloss:      *maxGlos,
		Level:         *level,
	})
	kaikkiFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build deck: %v\n", err)
		os.Exit(1)
	}
	if len(cards) == 0 {
		fmt.Fprintln(os.Stderr, "no cards built — check -pos and the input files")
		os.Exit(1)
	}

	if err := writeDeck(*out, deckName, cards); err != nil {
		fmt.Fprintf(os.Stderr, "write deck: %v\n", err)
		os.Exit(1)
	}

	if *stat {
		report(deckName, *out, floor, ceiling, stats, cards)
	}
}

func report(deckName, path string, floor, ceiling int, stats wordlist.Stats, cards []wordlist.Card) {
	fmt.Printf("deck: %s\n", deckName)
	fmt.Printf("file: %s\n", path)
	if floor > 1 || ceiling > 0 {
		fmt.Printf("band: rank %d", floor)
		if ceiling > 0 {
			fmt.Printf("-%d", ceiling)
		} else {
			fmt.Printf(" and rarer")
		}
		fmt.Printf("\n")
	}
	fmt.Printf("kept %d cards from %d entries\n", stats.Kept, stats.Entries)
	fmt.Printf("  dropped: %d wrong part of speech, %d inflected forms, %d without a translation,\n"+
		"           %d unranked, %d too rare, %d too common, %d no senses, %d duplicate headwords, %d unparsable lines\n",
		stats.WrongPos, stats.FormOf, stats.NoGloss, stats.Unranked, stats.TooRare, stats.TooCommon,
		stats.NoSense, stats.Duplicate, stats.Unparsable)
	if stats.Demoted > 0 {
		sample := make([]string, 0, 5)
		for i := len(cards) - 1; i >= 0 && len(sample) < 5; i-- {
			if cards[i].Demoted {
				sample = append(sample, cards[i].Front)
			}
		}
		fmt.Printf("  %d cards kept but ranked last: their frequency rank belongs to another word\n", stats.Demoted)
		fmt.Printf("  with the same spelling (e.g. %s)\n", strings.Join(sample, ", "))
	}
	fmt.Println("first cards:")
	for i, c := range cards {
		if i >= 8 {
			break
		}
		fmt.Printf("  %-6d %-24s %s\n", c.Rank, c.Front, c.Back)
	}
}

func writeDeck(path, deckName string, cards []wordlist.Card) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if strings.EqualFold(filepath.Ext(path), ".csv") {
		rows := make([]importexport.CSVCard, 0, len(cards))
		for _, c := range cards {
			rows = append(rows, importexport.CSVCard{
				Deck: deckName, Front: c.Front, Back: c.Back, NoteType: "vocab", Tags: c.Tags,
			})
		}
		return importexport.WriteCSV(f, rows)
	}

	deck := importexport.DeckFile{Deck: deckName, Cards: make([]importexport.CardFile, 0, len(cards))}
	for _, c := range cards {
		deck.Cards = append(deck.Cards, importexport.CardFile{
			Front: c.Front, Back: c.Back, NoteType: "vocab", Tags: c.Tags,
		})
	}
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(deck); err != nil {
		return err
	}
	return enc.Close()
}

func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
