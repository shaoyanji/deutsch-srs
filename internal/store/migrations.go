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
		stability_days REAL NOT NULL DEFAULT 1.0,
		difficulty REAL NOT NULL DEFAULT 0.35,
		last_review_at TEXT NOT NULL DEFAULT '',
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
	`CREATE INDEX IF NOT EXISTS idx_review_log_card ON review_log(card_id, answered_at);`,
	`CREATE TABLE IF NOT EXISTS deck_settings (
		deck TEXT PRIMARY KEY,
		new_per_day INTEGER NOT NULL,
		updated_at TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS schema_meta (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,
	// cards is an ordinary rowid table, so an external-content FTS5 table keyed
	// on cards.rowid keeps no second copy of the text. 'rebuild' (see
	// ensureSearchIndex) fills it once for databases that predate it.
	`CREATE VIRTUAL TABLE IF NOT EXISTS cards_fts USING fts5(
		front, back, tags, deck,
		content='cards',
		tokenize='unicode61'
	);`,
	// Triggers, not application code: no write path can forget to index a card.
	// UPDATE OF fires only when searchable text changes, so answering a card
	// does not reindex it.
	`CREATE TRIGGER IF NOT EXISTS cards_fts_ai AFTER INSERT ON cards BEGIN
		INSERT INTO cards_fts(rowid, front, back, tags, deck)
		VALUES (new.rowid, new.front, new.back, new.tags, new.deck);
	END;`,
	`CREATE TRIGGER IF NOT EXISTS cards_fts_ad AFTER DELETE ON cards BEGIN
		INSERT INTO cards_fts(cards_fts, rowid, front, back, tags, deck)
		VALUES ('delete', old.rowid, old.front, old.back, old.tags, old.deck);
	END;`,
	`CREATE TRIGGER IF NOT EXISTS cards_fts_au AFTER UPDATE OF front, back, tags, deck ON cards BEGIN
		INSERT INTO cards_fts(cards_fts, rowid, front, back, tags, deck)
		VALUES ('delete', old.rowid, old.front, old.back, old.tags, old.deck);
		INSERT INTO cards_fts(rowid, front, back, tags, deck)
		VALUES (new.rowid, new.front, new.back, new.tags, new.deck);
	END;`,
}
