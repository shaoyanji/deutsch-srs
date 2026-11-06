package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/shaoyanji/deutsch-srs/internal/app"
	"github.com/shaoyanji/deutsch-srs/internal/backend"
	"github.com/shaoyanji/deutsch-srs/internal/version"
)

func main() {
	var cfg app.Config
	var showVersion bool
	var syncPassFile string
	flag.StringVar(&cfg.Backend, "backend", "local", "backend to use: local|anki|anki-sync")
	flag.StringVar(&cfg.DBPath, "db", "./deutsch-srs.db", "path to sqlite database")
	flag.StringVar(&cfg.StarterYML, "starter", "./decks/starter.yaml", "path to starter deck yaml")
	flag.StringVar(&cfg.AnkiURL, "anki-url", "http://127.0.0.1:8765", "ankiconnect base url")
	flag.StringVar(&cfg.SyncURL, "sync-url", "", "anki sync server base url, e.g. http://anki.frieren.lan (backend anki-sync)")
	flag.StringVar(&cfg.SyncUser, "sync-user", "", "anki sync server username (backend anki-sync)")
	flag.StringVar(&syncPassFile, "sync-pass-file", "", "file holding the sync password; DEUTSCH_SRS_SYNC_PASS is read instead when unset")
	flag.StringVar(&cfg.SyncCache, "sync-cache", "", "downloaded collection path (default: user cache dir)")
	flag.IntVar(&cfg.NewPerDay, "new-per-day", backend.DefaultNewPerDay, "unseen cards a deck introduces per day (0 = none)")
	flag.BoolVar(&showVersion, "version", false, "print version/build info")
	flag.Parse()
	if showVersion {
		fmt.Println(version.String())
		return
	}

	cfg.SyncPass = os.Getenv("DEUTSCH_SRS_SYNC_PASS")
	if syncPassFile != "" {
		data, err := os.ReadFile(syncPassFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		cfg.SyncPass = strings.TrimSpace(string(data))
	}

	if err := app.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
