# AGENTS.md

## Repo identity

This repo is a thin Go TUI for German spaced repetition.

Keep it:
- local-first
- keyboard-first
- small
- explicit
- testable

Do not turn it into:
- a framework
- a sync server
- an AnkiWeb replacement
- a direct `.anki2` editor (reading a downloaded copy is fine)
- a plugin platform

## Tech shape

- Go
- Bubble Tea
- SQLite
- small deterministic SM-2-like scheduler
- optional AnkiConnect compatibility
- optional read-only pull from a self-hosted anki-sync-server

## Priorities

Build in this order:
1. local TUI
2. local SQLite backend
3. review loop
4. add/search/stats
5. thin AnkiConnect support
6. read-only pull from a sync server (`internal/anksync`)

Do not chase exact Anki parity.
Do not implement the write half of the Anki sync protocol. `internal/anksync`
logs in, reads meta and downloads the collection; the merge path
(start/chunk/applyChanges/finish) and upload stay out, because a half
implemented merge loses reviews silently.
Do not write directly to `.anki2`.

## Coding rules

Prefer:
- small packages
- narrow interfaces
- explicit state transitions
- additive diffs
- factual comments
- tests for logic-heavy code

Avoid:
- giant abstractions
- speculative plugin systems
- hidden magic
- broad refactors

## Output format

When finishing work, report:
1. exact files changed
2. concise summary per file
3. how to run
4. what remains deferred
5. AnkiConnect caveats
