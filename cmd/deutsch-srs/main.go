package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/shaoyanji/deutsch-srs/internal/app"
	"github.com/shaoyanji/deutsch-srs/internal/version"
)

func main() {
	var cfg app.Config
	var showVersion bool
	flag.StringVar(&cfg.Backend, "backend", "local", "backend to use: local|anki")
	flag.StringVar(&cfg.DBPath, "db", "./deutsch-srs.db", "path to sqlite database")
	flag.StringVar(&cfg.StarterYML, "starter", "./decks/starter.yaml", "path to starter deck yaml")
	flag.StringVar(&cfg.AnkiURL, "anki-url", "http://127.0.0.1:8765", "ankiconnect base url")
	flag.BoolVar(&showVersion, "version", false, "print version/build info")
	flag.Parse()
	if showVersion {
		fmt.Println(version.String())
		return
	}

	if err := app.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
