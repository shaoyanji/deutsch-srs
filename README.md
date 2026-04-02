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
| **Search** | Find cards by text or tags |
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

In **Add** mode, use the IO command field:

```yaml
# Import from YAML
import yaml ./my-deck.yaml

# Import from CSV
import csv ./my-deck.csv

# Export to CSV
export csv ./export.csv
```

## Configuration

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-backend` | `local` | Backend mode: `local` or `anki` |
| `-db` | `./deutsch-srs.db` | Path to SQLite database |
| `-starter` | `./decks/starter.yaml` | Path to starter deck YAML |
| `-anki-url` | `http://127.0.0.1:8765` | AnkiConnect URL (anki mode only) |
| `-version` | - | Print version info and exit |

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
├── internal/
│   ├── app/              # App lifecycle
│   ├── ui/               # Bubble Tea TUI
│   ├── backend/          # Local & AnkiConnect backends
│   ├── store/            # SQLite persistence
│   ├── srs/              # Spaced repetition scheduler
│   ├── anki/             # AnkiConnect client
│   └── importexport/     # YAML/CSV helpers
└── decks/                # Sample decks
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
