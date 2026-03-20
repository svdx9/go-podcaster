package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/svdx9/go-podcaster/internal/db/queries"
	_ "modernc.org/sqlite"
)

// TODO: use the sql/schema file to create the schema
const createTables = `
	CREATE TABLE episodes (
	uuid          TEXT PRIMARY KEY,
	title         TEXT NOT NULL,
	description   TEXT NOT NULL,
	author        TEXT,
	pub_date      DATETIME NOT NULL,
	file_path     TEXT NOT NULL,
	file_name     TEXT NOT NULL,
	file_size     INTEGER NOT NULL,
	mime_type     TEXT NOT NULL,
	duration_secs INTEGER NOT NULL,
	created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)
`

func Open(ctx context.Context, dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pragmas := []string{
		"journal_mode=WAL",
		"foreign_keys=ON",
	}
	for _, pragma := range pragmas {
		_, err = db.ExecContext(ctx, "PRAGMA "+pragma)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to set PRAGMA %s: %w", pragma, err)
		}
	}

	err = createSchema(ctx, db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ensure schema: %w", err)
	}

	return db, nil
}

func NewQuerier(db *sql.DB) queries.Querier {
	return queries.New(db)
}

func createSchema(ctx context.Context, db *sql.DB) error {
	var name string
	row := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='episodes'")
	err := row.Scan(&name)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check schema: %w", err)
	}

	_, err = db.ExecContext(ctx, createTables)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}
