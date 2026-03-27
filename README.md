# deutsch-srs

`deutsch-srs` is a thin local-first terminal app for German spaced repetition.

It is built with:

- Go
- Bubble Tea/Bubbles/Lip Gloss
- SQLite
- a small deterministic SM-2-like scheduler

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

Anki mode:

```bash
go run ./cmd/deutsch-srs \
  -backend anki \
  -anki-url http://127.0.0.1:8765
```

## Keybindings

- `j/k` or arrows: move
- `enter`: select in Browse; Add mode advances/submits; Search mode executes query
- `space`: reveal review answer
- `1/2/3/4`: Again/Hard/Good/Easy (after reveal)
- `tab`: switch mode (and form focus in Add mode)
- `esc`: back/cancel
- `?`: help
- `q`: quit

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
- Scheduler targets practical consistency, not exact Anki parity.
- Import/export helpers are intentionally narrow (simple YAML/CSV).
