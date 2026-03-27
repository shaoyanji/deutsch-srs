# deutsch-srs

`deutsch-srs` is a thin local-first terminal app for German spaced repetition.

It is built with:

- Go
- Bubble Tea/Bubbles/Lip Gloss
- SQLite
- a deterministic forgetting-curve-based local scheduler

## What It Is

- keyboard-first split-pane TUI
- explicit modes: Browse, Review, Add, Search, Stats, Help overlay
- local SQLite-backed study state
- optional AnkiConnect backend for edge compatibility
- simple import/export helpers

## What It Is Not

- not a full Anki clone
- not an AnkiWeb replacement
- not an Anki sync implementation
- not a direct `.anki2` editor
- not a plugin platform

## Backend Modes

1. Local mode (primary): `internal/backend/local.go`
2. AnkiConnect mode (secondary): `internal/anki/backend.go`

The UI depends on one backend interface (`internal/backend/backend.go`).

## Run

```bash
make tidy
make run
```

Default local mode flags:

```bash
go run ./cmd/deutsch-srs \
  -backend local \
  -db ./deutsch-srs.db \
  -starter ./decks/starter.yaml
```

When `-starter` is provided in local mode, startup seeds that deck only if the database is empty.

Anki mode:

```bash
go run ./cmd/deutsch-srs \
  -backend anki \
  -anki-url http://127.0.0.1:8765
```

## Keybindings

- `j/k` or arrows: move
- `enter`: select in Browse; Add mode submits on Tags and runs IO command on IO field; Search mode executes query
- `space`: reveal review answer
- `1/2/3/4`: Again/Hard/Good/Easy (after reveal)
- `tab`: switch mode (and form focus in Add mode)
- `esc`: back/cancel
- `?`: help
- `q`: quit (outside focused text-entry fields)
- `ctrl+c`: hard quit

While typing in Add/Search text inputs, printable keys are inserted into the input instead of triggering global quit/navigation bindings.

## Architecture

```text
cmd/deutsch-srs/main.go
internal/app/
internal/ui/
internal/backend/
internal/store/
internal/srs/
internal/anki/
internal/importexport/
decks/starter.yaml
```

## Tests

```bash
make test
```

Covers:

- scheduler transitions
- SQLite storage operations
- due-card selection path
- local review loop end-to-end
- starter seeding idempotency
- add-card persistence across reopen
- starter/import parsing
- AnkiConnect response envelope parsing

## Limitations

- Anki mode intentionally does not write review scheduling state back to Anki.
- Local scheduler uses per-card stability/difficulty with a forgetting-curve model, and is not exact Anki parity.
- Import/export helpers are intentionally narrow (simple YAML/CSV).

## Local Import/Export (Minimal Flow)

In Add mode, use the `IO command` field for local-only import/export:

- `import yaml <path>`
- `import csv <path>`
- `export csv <path>`

The UI shows explicit success or error text after each command.
