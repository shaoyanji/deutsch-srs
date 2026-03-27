package ui

import "testing"

func TestParseIOCommand(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{in: "import yaml decks/starter.yaml", wantErr: false},
		{in: "import csv cards.csv", wantErr: false},
		{in: "export csv out.csv", wantErr: false},
		{in: "export yaml out.yaml", wantErr: true},
		{in: "import json in.json", wantErr: true},
		{in: "bad csv out.csv", wantErr: true},
		{in: "import csv", wantErr: true},
	}
	for _, tt := range tests {
		_, err := parseIOCommand(tt.in)
		if (err != nil) != tt.wantErr {
			t.Fatalf("input=%q err=%v wantErr=%t", tt.in, err, tt.wantErr)
		}
	}
}
