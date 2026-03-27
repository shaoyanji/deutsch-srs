package importexport

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type DeckFile struct {
	Deck  string     `yaml:"deck"`
	Cards []CardFile `yaml:"cards"`
}

type CardFile struct {
	Front    string   `yaml:"front"`
	Back     string   `yaml:"back"`
	NoteType string   `yaml:"note_type"`
	Tags     []string `yaml:"tags"`
}

func LoadYAML(path string) (DeckFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return DeckFile{}, err
	}

	var deck DeckFile
	if err := yaml.Unmarshal(b, &deck); err != nil {
		return DeckFile{}, fmt.Errorf("parse yaml: %w", err)
	}
	if deck.Deck == "" {
		return DeckFile{}, fmt.Errorf("deck name is required")
	}
	return deck, nil
}
