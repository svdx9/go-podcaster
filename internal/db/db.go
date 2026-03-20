package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

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

	return db, nil
}
