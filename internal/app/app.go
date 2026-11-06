package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shaoyanji/deutsch-srs/internal/anki"
	"github.com/shaoyanji/deutsch-srs/internal/anksync"
	"github.com/shaoyanji/deutsch-srs/internal/backend"
	"github.com/shaoyanji/deutsch-srs/internal/ui"
	"github.com/shaoyanji/deutsch-srs/internal/version"
)

type Config struct {
	Backend    string
	DBPath     string
	StarterYML string
	AnkiURL    string
	NewPerDay  int

	// anki-sync backend: a self-hosted sync server, read-only.
	SyncURL   string
	SyncUser  string
	SyncPass  string
	SyncCache string
}

func Run(cfg Config) error {
	be, closeFn, err := newBackend(cfg)
	if err != nil {
		return err
	}
	if closeFn != nil {
		defer closeFn()
	}

	model := ui.NewModel(be)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func newBackend(cfg Config) (backend.Backend, func() error, error) {
	switch cfg.Backend {
	case "local", "":
		lb, err := backend.NewLocalBackend(cfg.DBPath, cfg.StarterYML)
		if err != nil {
			return nil, nil, err
		}
		if cfg.NewPerDay >= 0 {
			lb.SetDefaultNewPerDay(cfg.NewPerDay)
		}
		return lb, lb.Close, nil
	case "anki":
		if cfg.AnkiURL == "" {
			return nil, nil, fmt.Errorf("anki url is required in anki mode")
		}
		return anki.NewBackend(cfg.AnkiURL), nil, nil
	case "anki-sync", "ankisync":
		if cfg.SyncURL == "" {
			return nil, nil, fmt.Errorf("sync url is required in anki-sync mode")
		}
		if cfg.SyncUser == "" {
			return nil, nil, fmt.Errorf("sync user is required in anki-sync mode")
		}
		if cfg.SyncPass == "" {
			return nil, nil, fmt.Errorf("sync password is required in anki-sync mode (-sync-pass-file or DEUTSCH_SRS_SYNC_PASS)")
		}
		sb := anksync.NewBackend(anksync.Config{
			URL:           cfg.SyncURL,
			User:          cfg.SyncUser,
			Password:      cfg.SyncPass,
			CachePath:     cfg.SyncCache,
			ClientVersion: "deutsch-srs/" + version.Version,
		})
		return sb, sb.Close, nil
	default:
		return nil, nil, fmt.Errorf("unknown backend: %s", cfg.Backend)
	}
}
