package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shaoyanji/deutsch-srs/internal/anki"
	"github.com/shaoyanji/deutsch-srs/internal/backend"
	"github.com/shaoyanji/deutsch-srs/internal/ui"
)

type Config struct {
	Backend    string
	DBPath     string
	StarterYML string
	AnkiURL    string
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
		return lb, lb.Close, nil
	case "anki":
		if cfg.AnkiURL == "" {
			return nil, nil, fmt.Errorf("anki url is required in anki mode")
		}
		return anki.NewBackend(cfg.AnkiURL), nil, nil
	default:
		return nil, nil, fmt.Errorf("unknown backend: %s", cfg.Backend)
	}
}
