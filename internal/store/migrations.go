package store

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS cards (
		id TEXT PRIMARY KEY,
		deck TEXT NOT NULL,
		front TEXT NOT NULL,
		back TEXT NOT NULL,
		note_type TEXT NOT NULL DEFAULT '',
		tags TEXT NOT NULL DEFAULT '',
		due_at TEXT NOT NULL,
		interval_days INTEGER NOT NULL DEFAULT 0,
		ease REAL NOT NULL DEFAULT 2.5,
		reps INTEGER NOT NULL DEFAULT 0,
		lapses INTEGER NOT NULL DEFAULT 0,
		suspended INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_cards_deck ON cards(deck);`,
	`CREATE INDEX IF NOT EXISTS idx_cards_due ON cards(due_at, suspended);`,
	`CREATE TABLE IF NOT EXISTS review_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		card_id TEXT NOT NULL,
		grade INTEGER NOT NULL,
		answered_at TEXT NOT NULL,
		interval_days INTEGER NOT NULL,
		ease REAL NOT NULL,
		FOREIGN KEY(card_id) REFERENCES cards(id)
	);`,
}
