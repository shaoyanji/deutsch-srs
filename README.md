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

# Pull from a self-hosted anki-sync-server (read-only)
DEUTSCH_SRS_SYNC_PASS=… deutsch-srs -backend anki-sync \
  -sync-url http://anki.frieren.lan -sync-user devji
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
| `b` | Bury — out of today's queue, back tomorrow, schedule untouched |
| `x` | Suspend — hidden until unsuspended, schedule untouched |
| `u` | Undo the last grade, bury or suspend |

Bury and suspend are deliberately not grades: a card you cannot face today
should not be scored as a lapse, so neither action moves the interval, ease or
repetition count. `u` is the way back out of all three — it restores the card's
stored state, puts it back in its slot in the session, and takes a grade back
out of today's review count and the review log. The last 50 actions are
undoable, as long as the session lasts.

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

| File | Cards | Contents | In git |
|------|-------|----------|--------|
| `decks/starter.yaml` | 6 | A handful of first sentences, seeded on first run | yes |
| `decks/wortschatz-a1.yaml` | 214 | Vocabulary, frequency ranks 1-600 | fetched |
| `decks/wortschatz-a2.yaml` | 467 | Vocabulary, frequency ranks 601-1500 | fetched |
| `decks/wortschatz-b1.yaml` | 762 | Vocabulary, frequency ranks 1501-3000 | fetched |
| `decks/wortschatz-b2.yaml` | 1418 | Vocabulary, frequency ranks 3001-6000 | fetched |
| `decks/b2-schreiben.yaml` | 44 | B2 telc writing preparation | yes |

The four word decks are bands, not nesting levels: each one starts where the
previous ended, so importing all four studies every word exactly once — 2,861
cards, no word in two levels. They hold nouns, verbs and adjectives only:
grammar belongs in a grammar reference, not in a vocabulary deck, so articles,
pronouns, prepositions, conjunctions and adverbs are filtered out by part of
speech, and the handful of function words Wiktionary files as nouns (`Ja` = "a
yes", `Habe` = "belongings", `Würde` = "dignity") are excluded by spelling.

### Fetching the word decks

Those 2,861 cards are the output of `cmd/wordlist`, not source: they are
published as a [gist](https://gist.github.com/shaoyanji/4e0e42be411144351d8fb1f2b574795f)
pinned to one revision, and `internal/deckfetch/manifest.json` holds that
revision plus a SHA-256 per file. Keeping them out of the tree stops every
regeneration from rewriting ~290 KiB in history, and the checksums mean a
mutated or truncated download is refused instead of imported.

```bash
deutsch-srs-cli fetch                  # all four, into ~/.local/share/deutsch-srs/decks
deutsch-srs-cli fetch --only a1,a2     # just the beginner levels
deutsch-srs-cli fetch --out ./decks    # anywhere else (decks/wortschatz-*.yaml is gitignored)
deutsch-srs-cli fetch --check          # verify the local copies, no network

deutsch-srs-cli import ~/.local/share/deutsch-srs/decks/wortschatz-a1.yaml
```

A copy that already matches its checksum is skipped, so re-running is free;
`--force` re-downloads. The starter deck is compiled in, so the app works on a
fresh clone with no fetch at all.

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

# A1 to B2: each level is its own frequency band (a1=1-600, a2=601-1500,
# b1=1501-3000, b2=3001-6000), so the decks do not repeat each other.
# -pos keeps nouns, verbs and adjectives only; -exclude drops the function words
# Wiktionary files as nouns; -content-only drops cards whose rank belongs to
# another word with the same spelling.
for lvl in a1 a2 b1 b2; do
  go run ./cmd/wordlist -kaikki kaikki-de.jsonl -freq de_full.txt \
    -out decks/wortschatz-$lvl.yaml -deck "Wortschatz ${lvl^^}" -level $lvl \
    -pos noun,verb,adj -content-only -exclude decks/grammar-words.txt
done

# Load it; re-running is safe
deutsch-srs-cli import ./wortschatz-b1.yaml
```

Regenerating publishes a new deck, so the gist needs the new file and the
manifest needs a new `revision`, `sha256` and `bytes` for it — `fetch` refuses
a file whose checksum disagrees with the manifest, which is the point.

| Flag | Meaning |
|------|---------|
| `-level` | that level's frequency band: `a1`, `a2`, `b1`, `b2` (a proxy, not an exam syllabus) |
| `-cumulative` | with `-level`, keep every word up to the level instead of only its own band |
| `-min-rank` | drop lemmas more frequent than this rank (0 = none) |
| `-pos` | parts of speech to keep; default `noun,verb,adj,adv` |
| `-limit` | hard cap on cards, applied after ordering |
| `-max-rank` | drop lemmas rarer than this rank (overrides `-level` when smaller) |
| `-max-gloss` | trim translations to this many characters (default 120) |
| `-allow-unranked` | keep lemmas the frequency list never saw, ordered last |
| `-exclude` | file of spellings to drop whatever their part of speech (case-insensitive), one per line |
| `-content-only` | drop cards whose rank belongs to another word with the same spelling, instead of keeping them at the end |

The build reports what it dropped, so a deck can be audited instead of trusted.
Entries are rejected when they are inflected forms (`ist`, `Häuser`), when a
sense only describes another lemma (`plural of Haus`, `obsolete form of freien`),
when a sense is tagged as existing only inside compounds (`ein-` as "motion into
something"), or when a sense is the auxiliary use of a verb (`haben` as "forms
the perfect aspect" - the card is "to have; to own"). A card whose frequency
came from a different word with the same spelling keeps its place in the deck but
ranks last: `Ich` (the ego) inherits the rank of the pronoun `ich`, `einen` (to
unite) inherits the rank of the article form `einen` <- `ein`, and `auf` (open)
inherits the rank of the preposition. Two limits keep that from over-reaching: a
reading that Wiktionary marks rare, slang or archaic does not shadow anything
(`Frau` outranks the rare pronoun `frau`), and an ordinary adjective does not lose
its rank to a preposition it happens to be spelled like (`lang`).

Adverbs are worth knowing about: they are in the default part-of-speech set
because `gestern` and `immer` are vocabulary, but `mit`, `aber` and `so` are filed
as adverbs in Wiktionary too, and their cards are grammar. The bundled decks
leave adverbs out entirely and add `-content-only -exclude decks/grammar-words.txt`,
which is the combination that keeps `Ja`, `Habe` and `Würde` out: Wiktionary
files them as nouns, and the rank they would inherit belongs to the particle.

## Configuration

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-backend` | `local` | Backend mode: `local`, `anki` or `anki-sync` |
| `-db` | `./deutsch-srs.db` | Path to SQLite database |
| `-starter` | `./decks/starter.yaml` | Path to starter deck YAML |
| `-anki-url` | `http://127.0.0.1:8765` | AnkiConnect URL (anki mode only) |
| `-sync-url` | - | anki-sync-server base URL, e.g. `http://anki.frieren.lan` |
| `-sync-user` | - | sync server username |
| `-sync-pass-file` | - | file holding the sync password (`DEUTSCH_SRS_SYNC_PASS` otherwise) |
| `-sync-cache` | user cache dir | where the downloaded collection is kept |
| `-new-per-day` | `20` | Unseen cards a deck introduces per day (`0` = none) |
| `-version` | - | Print version info and exit |

### Anki sync server (read-only)

`-backend anki-sync` logs into a self-hosted `anki-sync-server` with the
protocol Anki itself speaks (sync version 11: an `anki-sync` JSON header and a
zstd body), downloads the collection once, and reviews that snapshot.

The write half is deliberately absent: no merge, no upload. Grading, burying
and suspending return `ErrReadOnly` instead of writing to the local copy,
because a review that the next download would erase is worse than a refusal.

```bash
DEUTSCH_SRS_SYNC_PASS="$(cat ~/.config/deutsch-srs/sync-pass)" \
  deutsch-srs -backend anki-sync \
    -sync-url http://anki.frieren.lan \
    -sync-user devji
```

`DEUTSCH_SRS_SYNC_URL`, `DEUTSCH_SRS_SYNC_USER` and `DEUTSCH_SRS_SYNC_PASS`
also drive the live test: `go test ./internal/anksync -run Live -v` is skipped
when the URL is unset.

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
│   ├── anksync/          # anki-sync-server client (pull only) + collection reader
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
- **anki-sync Mode**: Pull only; downloads the server's collection and never pushes reviews back
- **Scheduler**: Uses per-card stability/difficulty model; not exact Anki algorithm parity
- **Import/Export**: Intentionally narrow scope (simple YAML/CSV only)

## Project Status

- ✅ Local mode: Fully functional
- ✅ AnkiConnect mode: Secondary compatibility layer
- ✅ anki-sync mode: Read-only pull from a self-hosted sync server
- 🚧 Active development

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [SQLite](https://sqlite.org/) - Embedded database
- SM-2 algorithm inspiration from [SuperMemo](https://www.super-memory.com/)
