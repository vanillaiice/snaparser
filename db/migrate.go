package db

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*sql
var migrations embed.FS

func RunMigrations(dsn string) error {
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}

	defer conn.Close()

	conn.SetMaxOpenConns(1)

	return runMigrations(conn)
}

// Migrate the caller's connection so in-memory archives retain their schema.
func runMigrations(conn *sql.DB) error {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}

	defer source.Close()

	driver, err := sqlite.WithInstance(conn, &sqlite.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return err
	}

	// m.Close would also close the connection owned by the caller.
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
