package db

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	ctx := context.Background()

	if err := RunMigrations(path); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err = conn.Ping(); err != nil {
		conn.Close()

		return nil, err
	}

	conn.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	} {
		if _, err = conn.ExecContext(ctx, pragma); err != nil {
			conn.Close()

			return nil, err
		}
	}

	db := &DB{conn: conn}

	return db, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}
