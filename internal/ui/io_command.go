package ui

import (
	"fmt"
	"strings"
)

type ioCommand struct {
	Action string
	Format string
	Path   string
}

func parseIOCommand(raw string) (ioCommand, error) {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) != 3 {
		return ioCommand{}, fmt.Errorf("use: import yaml <path> | import csv <path> | export csv <path>")
	}
	cmd := ioCommand{
		Action: strings.ToLower(fields[0]),
		Format: strings.ToLower(fields[1]),
		Path:   fields[2],
	}
	switch cmd.Action {
	case "import":
		if cmd.Format != "yaml" && cmd.Format != "csv" {
			return ioCommand{}, fmt.Errorf("import format must be yaml or csv")
		}
	case "export":
		if cmd.Format != "csv" {
			return ioCommand{}, fmt.Errorf("export format must be csv")
		}
	default:
		return ioCommand{}, fmt.Errorf("action must be import or export")
	}
	return cmd, nil
}
