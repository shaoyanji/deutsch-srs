package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/shaoyanji/deutsch-srs/internal/backend"
	"github.com/shaoyanji/deutsch-srs/internal/srs"
)

const defaultDB = "~/.local/share/deutsch-srs/deutsch-srs.db"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	dbPath := os.Getenv("DEUTSCH_SRS_DB")
	if dbPath == "" {
		dbPath = defaultDB
	}
	// Expand ~
	if strings.HasPrefix(dbPath, "~/") {
		home, _ := os.UserHomeDir()
		dbPath = home + dbPath[1:]
	}

	starterYML := os.Getenv("DEUTSCH_SRS_STARTER")

	be, err := backend.NewLocalBackend(dbPath, starterYML)
	if err != nil {
		fatal("open db: %v", err)
	}
	defer be.Close()

	ctx := context.Background()
	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "due":
		cmdDue(ctx, be, args)
	case "show":
		cmdShow(ctx, be, args)
	case "answer":
		cmdAnswer(ctx, be, args)
	case "add":
		cmdAdd(ctx, be, args)
	case "stats":
		cmdStats(ctx, be, args)
	case "decks":
		cmdDecks(ctx, be)
	case "search":
		cmdSearch(ctx, be, args)
	case "import":
		cmdImport(ctx, be, args)
	case "limit":
		cmdLimit(ctx, be, args)
	default:
		fatal("unknown command: %s", cmd)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: deutsch-srs-cli <command> [args]

Commands:
  due [--deck NAME] [--limit N]   List cards due for review
  show CARD_ID                    Show a card (front only)
  answer CARD_ID GRADE            Record review (1=again 2=hard 3=good 4=easy)
  add --deck NAME --front F --back B [--tags t1,t2]
  stats [--deck NAME]             Review statistics
  decks                           List decks
  search QUERY                    Search cards
  import FILE... [--deck NAME] [--format auto|csv|tsv|yaml]
                                  [--dry-run] [--allow-duplicates]
                                  Import decks; cards already in the deck are skipped
  limit [--deck NAME] [--new-per-day N]
                                  Show or change a deck's daily new-card cap

Environment:
  DEUTSCH_SRS_DB       Database path (default: ~/.local/share/deutsch-srs/deutsch-srs.db)
  DEUTSCH_SRS_STARTER  Starter deck YAML path`)
}

// --- Commands ---

func cmdDue(ctx context.Context, be backend.Backend, args []string) {
	deck := ""
	limit := 20
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--deck":
			i++
			if i < len(args) {
				deck = args[i]
			}
		case "--limit":
			i++
			if i < len(args) {
				limit, _ = strconv.Atoi(args[i])
			}
		}
	}

	// If no deck specified, get due cards from all decks
	if deck == "" {
		decks, err := be.ListDecks(ctx)
		if err != nil {
			fatal("due: %v", err)
		}
		var all []srs.Card
		for _, d := range decks {
			cards, err := be.DueCards(ctx, d.Name, limit)
			if err != nil {
				continue
			}
			all = append(all, cards...)
		}
		if len(all) > limit {
			all = all[:limit]
		}
		output(all)
		return
	}

	cards, err := be.DueCards(ctx, deck, limit)
	if err != nil {
		fatal("due: %v", err)
	}
	output(cards)
}

func cmdShow(ctx context.Context, be backend.Backend, args []string) {
	if len(args) < 1 {
		fatal("show: CARD_ID required")
	}
	// Search across all decks
	decks, err := be.ListDecks(ctx)
	if err != nil {
		fatal("show: %v", err)
	}
	for _, d := range decks {
		cards, err := be.DueCards(ctx, d.Name, 1000)
		if err != nil {
			continue
		}
		for _, c := range cards {
			if c.ID == args[0] {
				output(map[string]any{
					"id":       c.ID,
					"front":    c.Front,
					"deck":     c.Deck,
					"tags":     c.Tags,
					"due_at":   c.DueAt,
					"reps":     c.Reps,
					"lapses":   c.Lapses,
					"interval": c.IntervalDays,
				})
				return
			}
		}
	}
	fatal("show: card %s not found", args[0])
}

func cmdAnswer(ctx context.Context, be backend.Backend, args []string) {
	if len(args) < 2 {
		fatal("answer: CARD_ID GRADE required")
	}
	cardID := args[0]
	gradeInt, err := strconv.Atoi(args[1])
	if err != nil || gradeInt < 1 || gradeInt > 4 {
		fatal("answer: grade must be 1-4 (again/hard/good/easy)")
	}
	grade := srs.ReviewGrade(gradeInt)
	if err := be.AnswerCard(ctx, cardID, grade); err != nil {
		fatal("answer: %v", err)
	}
	output(map[string]string{"status": "ok", "card_id": cardID, "grade": args[1]})
}

func cmdAdd(ctx context.Context, be backend.Backend, args []string) {
	var deck, front, back, tagsStr string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--deck":
			i++
			if i < len(args) {
				deck = args[i]
			}
		case "--front":
			i++
			if i < len(args) {
				front = args[i]
			}
		case "--back":
			i++
			if i < len(args) {
				back = args[i]
			}
		case "--tags":
			i++
			if i < len(args) {
				tagsStr = args[i]
			}
		}
	}
	if deck == "" || front == "" || back == "" {
		fatal("add: --deck, --front, --back required")
	}
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}
	card, err := be.AddCard(ctx, backend.AddCardInput{
		Deck:  deck,
		Front: front,
		Back:  back,
		Tags:  tags,
	})
	if err != nil {
		fatal("add: %v", err)
	}
	output(map[string]any{"status": "ok", "id": card.ID, "deck": card.Deck, "front": card.Front})
}

func cmdStats(ctx context.Context, be backend.Backend, args []string) {
	stats, err := be.Stats(ctx)
	if err != nil {
		fatal("stats: %v", err)
	}
	output(stats)
}

func cmdDecks(ctx context.Context, be backend.Backend) {
	decks, err := be.ListDecks(ctx)
	if err != nil {
		fatal("decks: %v", err)
	}
	output(decks)
}

func cmdSearch(ctx context.Context, be backend.Backend, args []string) {
	if len(args) < 1 {
		fatal("search: query required")
	}
	query := strings.Join(args, " ")
	cards, err := be.SearchCards(ctx, query, 20)
	if err != nil {
		fatal("search: %v", err)
	}
	output(cards)
}

func cmdImport(ctx context.Context, be *backend.LocalBackend, args []string) {
	var (
		deckOverride string
		format       string
		skipExisting = true
		dryRun       bool
		paths        []string
	)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--deck":
			i++
			if i < len(args) {
				deckOverride = args[i]
			}
		case "--format":
			i++
			if i < len(args) {
				format = args[i]
			}
		case "--allow-duplicates":
			skipExisting = false
		case "--dry-run":
			dryRun = true
		default:
			paths = append(paths, args[i])
		}
	}
	if len(paths) == 0 {
		fatal("import: FILE required (yaml, csv or tsv)")
	}

	type fileResult struct {
		File    string `json:"file"`
		Deck    string `json:"deck"`
		Added   int    `json:"added"`
		Skipped int    `json:"skipped"`
		Invalid int    `json:"invalid"`
	}
	results := make([]fileResult, 0, len(paths))
	total := fileResult{}
	for _, path := range paths {
		res, err := be.ImportFile(ctx, path, backend.ImportOptions{
			DeckOverride: deckOverride,
			Format:       format,
			SkipExisting: skipExisting,
			DryRun:       dryRun,
		})
		if err != nil {
			fatal("import %s: %v", path, err)
		}
		results = append(results, fileResult{
			File:    res.File,
			Deck:    res.Deck,
			Added:   res.Added,
			Skipped: res.Skipped,
			Invalid: res.Invalid,
		})
		total.Added += res.Added
		total.Skipped += res.Skipped
		total.Invalid += res.Invalid
	}

	output(map[string]any{
		"files":         results,
		"added":         total.Added,
		"skipped":       total.Skipped,
		"invalid":       total.Invalid,
		"dry_run":       dryRun,
		"skip_existing": skipExisting,
	})
}

func cmdLimit(ctx context.Context, be *backend.LocalBackend, args []string) {
	var deck string
	newPerDay := -1
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--deck":
			i++
			if i < len(args) {
				deck = args[i]
			}
		case "--new-per-day":
			i++
			if i < len(args) {
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					fatal("limit: --new-per-day must be zero or more")
				}
				newPerDay = n
			}
		}
	}
	if newPerDay >= 0 {
		if deck == "" {
			fatal("limit: --deck is required when setting a cap")
		}
		if err := be.SetDeckNewPerDay(ctx, deck, newPerDay); err != nil {
			fatal("limit: %v", err)
		}
	}

	limits, err := be.DeckLimits(ctx)
	if err != nil {
		fatal("limit: %v", err)
	}
	if deck != "" {
		for _, l := range limits {
			if l.Deck == deck {
				output(l)
				return
			}
		}
		fatal("limit: deck %q not found", deck)
	}
	output(limits)
}

// --- Helpers ---

func output(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatal("encode: %v", err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
