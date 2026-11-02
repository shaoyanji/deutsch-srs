# deutsch-srs

A keyboard-first terminal app for learning German with spaced repetition.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/go-1.21+-blue.svg)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey.svg)

## Features

- **Terminal UI**: Fast, keyboard-driven interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Spaced Repetition**: SM-2-like scheduler based on forgetting curve research
- **Local-First**: SQLite database stored locally; no account or internet required
- **Import/Export**: YAML and CSV support for deck management
- **AnkiConnect Compatible**: Optional backend for Anki integration (read-only scheduling)

## Installation

### From GitHub Releases

Download the latest binary for your platform from the [releases page](https://github.com/shaoyanji/deutsch-srs/releases):

| Platform | Binary |
|----------|--------|
| Linux (amd64) | `deutsch-srs-linux-amd64` |
| macOS (Intel) | `deutsch-srs-darwin-amd64` |
| macOS (Apple Silicon) | `deutsch-srs-darwin-arm64` |
| Windows (amd64) | `deutsch-srs-windows-amd64.exe` |

**Linux/macOS:**
```bash
chmod +x deutsch-srs-<platform>
./deutsch-srs-<platform>
```

**Windows:**
```powershell
deutsch-srs-windows-amd64.exe
```

### From Source

Requires Go 1.21 or later:

```bash
git clone https://github.com/shaoyanji/deutsch-srs.git
cd deutsch-srs
go build -o deutsch-srs ./cmd/deutsch-srs
./deutsch-srs
```

### Using Go Install

```bash
go install github.com/shaoyanji/deutsch-srs/cmd/deutsch-srs@latest
deutsch-srs
```

## Quick Start

```bash
# Start with default settings (creates deutsch-srs.db in current directory)
deutsch-srs

# Or specify a custom database path
deutsch-srs -db ~/.deutsch-srs/my-decks.db

# Use with AnkiConnect (optional)
deutsch-srs -backend anki -anki-url http://127.0.0.1:8765
```

On first run with the `-starter` flag (default: `./decks/starter.yaml`), the app seeds a starter German deck if the database is empty.

## Usage

### Modes

| Mode | Description |
|------|-------------|
| **Browse** | Select a deck to review |
| **Review** | Study cards with spaced repetition |
| **Add** | Create new cards manually |
| **Search** | Find cards by word prefix, translation, tag or deck name |
| **Stats** | View learning statistics |

### Keybindings

#### Navigation
| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Select / Confirm |
| `Tab` | Switch mode |
| `Esc` | Go back |

#### Review
| Key | Action |
|-----|--------|
| `Space` | Reveal answer |
| `1` | Again (reset, show soon) |
| `2` | Hard |
| `3` | Good |
| `4` | Easy |

#### Global
| Key | Action |
|-----|--------|
| `?` | Toggle help |
| `q` | Quit |
| `Ctrl+C` | Force quit |

### Adding Cards

In **Add** mode, fill in the fields:
1. Deck name
2. Front (German)
3. Back (English)
4. Tags (comma-separated)
5. IO command (optional, for import/export)

Press `Enter` on the Tags field to submit. Use `Tab` to navigate between fields.

### Import/Export

Import accepts YAML, CSV and TSV, and works out which one it is looking at:

- **Format** comes from the file extension, or is sniffed from the content
  (`.yaml`/`.yml`, `.csv`, `.tsv`, `.txt`, anything else).
- **Delimiter** is sniffed per file, so comma, semicolon and tab separated
  lists all load (`Hund;dog` is a valid two-column deck).
- **Header rows** are recognised by name (`front`/`wort`/`vorn`, `back`/
  `übersetzung`/`bedeutung`, `deck`, `tags`, `note_type`), so the columns may
  appear in any order. With no header, the layout is
  `deck,front,back,note_type,tags`, and a two-column file becomes `front,back`.
- **Re-importing is safe**: a card already in the deck (same deck, same front,
  case-insensitive) is skipped rather than duplicated.

From the TUI, use the IO command field in **Add** mode:

```yaml
import csv ./meine-liste.csv
import yaml ./my-deck.yaml
export csv ./export.csv
```

From the CLI, for bulk loads and scripting:

```bash
# Import one or more files; already-present cards are skipped
deutsch-srs-cli import ./meine-liste.csv ./redemittel.tsv

# Override the deck name, or preview without writing
deutsch-srs-cli import ./liste.txt --deck "B2 Wortschatz" --dry-run

# Allow genuine duplicates instead of skipping them
deutsch-srs-cli import ./liste.csv --allow-duplicates
```

The CLI prints a summary (`added`, `skipped`, `invalid`) instead of every card
id, and writes the whole file in one transaction.

### Search

Search runs against an SQLite FTS5 index over the card front, back, tags and
deck name, so it costs the same on a 200-card deck and a 77,000-card one (about
2 ms against a scanned 100 ms, measured on the 77k word deck). Every query word
matches as a prefix: `hund` finds `Hund` and `Hundehütte`, but not `Dachshund`.

Whole words rank above prefixes, so `haus` puts `Haus` before `Hausschlüssel`
rather than leaving the order to bm25, which scores both the same. Umlauts are
folded, so `uber` finds `über` and `schon` also finds `schön`.

The index is maintained by triggers on `cards`, and a database created before it
existed is backfilled once on open (about 2 s for 77k cards). Re-importing a deck
does not rebuild it.

## Bundled decks

| File | Contents |
|------|----------|
| `decks/starter.yaml` | A handful of first sentences, seeded on first run |
| `decks/b2-schreiben.yaml` | B2 telc writing preparation: 44 cards |

`decks/b2-schreiben.yaml` covers the writing subtest (30 minutes, at least 150
words, 45 points): the exam facts and the three rating criteria, the four
task types with a worked prompt each, a letter/email skeleton, Redemittel
grouped by task and function, a grammar card on connector word order, a
four-item error checklist, and one full model answer per task type.

```bash
# Load it in one transaction; safe to re-run, existing cards are skipped
deutsch-srs-cli import ./decks/b2-schreiben.yaml
```

## Generating a word deck

`cmd/wordlist` builds a frequency-ordered vocabulary deck from two public
sources, which is how the A1-B2 word lists are regenerated:

| Source | What it contributes |
|--------|---------------------|
| [kaikki.org](https://kaikki.org/dictionary/German/) German Wiktionary extract | headwords, part of speech, English translations |
| [FrequencyWords](https://github.com/hermitdave/FrequencyWords) `de_full.txt` | OpenSubtitles token ranks, so the deck leads with words people use |

```bash
curl -L -o kaikki-de.jsonl https://kaikki.org/dictionary/German/kaikki.org-dictionary-German.jsonl
curl -L -o de_full.txt \
  https://raw.githubusercontent.com/hermitdave/FrequencyWords/master/content/2018/de/de_full.txt

# A1 to B2: each level is a frequency band (a1=600, a2=1500, b1=3000, b2=6000)
go run ./cmd/wordlist -kaikki kaikki-de.jsonl -freq de_full.txt \
  -out decks/wortschatz-b1.yaml -deck "Wortschatz B1" -level b1

# Load it; re-running is safe
deutsch-srs-cli import ./decks/wortschatz-b1.yaml
```

| Flag | Meaning |
|------|---------|
| `-level` | frequency ceiling: `a1`, `a2`, `b1`, `b2` (a proxy, not an exam syllabus) |
| `-pos` | parts of speech to keep; default `noun,verb,adj,adv` |
| `-limit` | hard cap on cards, applied after ordering |
| `-max-rank` | drop lemmas rarer than this rank (overrides `-level` when smaller) |
| `-max-gloss` | trim translations to this many characters (default 120) |
| `-allow-unranked` | keep lemmas the frequency list never saw, ordered last |

The build reports what it dropped, so a deck can be audited instead of trusted.
Entries are rejected when they are inflected forms (`ist`, `Häuser`), when a
sense only describes another lemma (`plural of Haus`, `obsolete form of freien`),
when a sense is tagged as existing only inside compounds (`ein-` as "motion into
something"), or when a sense is the auxiliary use of a verb (`haben` as "forms
the perfect aspect" - the card is "to have; to own"). A noun or adjective whose
lowercase spelling belongs to a closed-class word keeps its card but ranks last,
because its frequency came from that other word: `Ich` (the ego) inherits the
rank of the pronoun `ich`.

Adverbs are worth knowing about: they are in the default set because `gestern`
and `immer` are vocabulary, but `mit`, `aber` and `so` are filed as adverbs in
Wiktionary too, and their cards are grammar rather than vocabulary. Build with
`-pos noun,verb,adj` if you want content words only.

## Configuration

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-backend` | `local` | Backend mode: `local` or `anki` |
| `-db` | `./deutsch-srs.db` | Path to SQLite database |
| `-starter` | `./decks/starter.yaml` | Path to starter deck YAML |
| `-anki-url` | `http://127.0.0.1:8765` | AnkiConnect URL (anki mode only) |
| `-new-per-day` | `20` | Unseen cards a deck introduces per day (`0` = none) |
| `-version` | - | Print version info and exit |

### Daily new cards

A review session is due reviews first, then unseen cards - but only as many
new ones as the deck allows per day. That is what keeps a 70,000-card word
list usable: you get the backlog you have actually learned plus a fixed dose
of new material, not 70,000 cards at once.

The app default is 20 new cards per deck per day (`-new-per-day`). Per-deck
settings live in the database, so they survive a restart:

```bash
# What is each deck set to, and how much of today's allowance is used?
deutsch-srs-cli limit

# Give one deck its own pace; 0 pauses new cards entirely
deutsch-srs-cli limit --deck "Wortschatz B2" --new-per-day 8
```

When a batch of 50 cards is finished, `enter` loads the next one instead of
sending you back to the deck list. Once reviews are done and the day's new
cards are used, the review screen says so.

### Deck YAML Format

```yaml
deck: basics
cards:
  - front: Guten Morgen
    back: Good morning
    tags: greeting,basics
  - front: Danke
    back: Thank you
    tags: basics
```

## Architecture

```
deutsch-srs
├── cmd/deutsch-srs/      # Main entry point
├── cmd/cli/              # Headless CLI (import, search, decks, limit)
├── cmd/wordlist/         # Builds A1-B2 decks from kaikki + a frequency list
├── internal/
│   ├── app/              # App lifecycle
│   ├── ui/               # Bubble Tea TUI
│   ├── backend/          # Local & AnkiConnect backends
│   ├── store/            # SQLite persistence and the FTS5 search index
│   ├── srs/              # Spaced repetition scheduler
│   ├── anki/             # AnkiConnect client
│   ├── wordlist/         # kaikki/frequency join, sense cleaning
│   └── importexport/     # YAML/CSV helpers
└── decks/                # Bundled decks (starter vocabulary, B2 telc writing)
```

## Development

```bash
# Format code
make fmt

# Tidy dependencies
make tidy

# Run tests
make test

# Build
make build
./bin/deutsch-srs --version

# Run directly
make run
```

### Building for Release

```bash
# Build for current platform
make build VERSION=v0.1.0

# Cross-compile (see Makefile for targets)
make build-linux
make build-macos
make build-windows
```

## Limitations

- **Anki Mode**: Read-only scheduling; does not write review state back to Anki
- **Scheduler**: Uses per-card stability/difficulty model; not exact Anki algorithm parity
- **Import/Export**: Intentionally narrow scope (simple YAML/CSV only)

## Project Status

- ✅ Local mode: Fully functional
- ✅ AnkiConnect mode: Secondary compatibility layer
- 🚧 Active development

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [SQLite](https://sqlite.org/) - Embedded database
- SM-2 algorithm inspiration from [SuperMemo](https://www.super-memory.com/)
