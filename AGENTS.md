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
- a direct `.anki2` editor
- a plugin platform

## Tech shape

- Go
- Bubble Tea
- SQLite
- small deterministic SM-2-like scheduler
- optional AnkiConnect compatibility

## Priorities

Build in this order:
1. local TUI
2. local SQLite backend
3. review loop
4. add/search/stats
5. thin AnkiConnect support

Do not chase exact Anki parity.
Do not implement Anki sync protocol.
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
