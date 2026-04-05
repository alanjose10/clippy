package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

// ErrNotFound indicates that the resource was not found in the db
var ErrNotFound = errors.New("not found in db")

type DB struct {
	db *sql.DB
}

// Open opens the db and sets the necessary config
func Open(ctx context.Context, dbPath string) (*DB, error) {
	d := &DB{}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	d.db = db

	// verify connectivity
	if err := d.db.Ping(); err != nil {
		return nil, err
	}

	// sqlite only supports one writer
	d.db.SetMaxOpenConns(1)

	if err := d.migrate(ctx); err != nil {
		return nil, err
	}

	return d, nil
}

// Close closes the db connection
func (d *DB) Close() error {
	if d == nil {
		return nil
	}
	return d.db.Close()
}

func (d *DB) migrate(ctx context.Context) error {
	migrations := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA foreign_keys=ON;`,
		`CREATE TABLE IF NOT EXISTS rooms (
			id         TEXT PRIMARY KEY,
			created_at INTEGER NOT NULL DEFAULT (unixepoch('now')),
			pin_hash   TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS snippets (
			id          TEXT PRIMARY KEY,
			room_id     TEXT NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
			name        TEXT,
			content_enc BLOB NOT NULL,
			language    TEXT NOT NULL DEFAULT 'plaintext',
			tags        TEXT NOT NULL DEFAULT '',

			expires_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL DEFAULT (unixepoch('now')),
			updated_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_snippets_room_expires
			ON snippets(room_id, expires_at)`,
	}
	for _, query := range migrations {
		if _, err := d.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("running migration %q: %w", query, err)
		}
	}
	return nil
}
