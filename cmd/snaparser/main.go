/*
Snaparser imports Snapchat JSON exports and serves an embedded archive browser.
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/vanillaiice/snaparser/db"
	"github.com/vanillaiice/snaparser/frontend"
	"github.com/vanillaiice/snaparser/parser"
)

func main() {
	if err := newApp().Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func newApp() *cli.App {
	return &cli.App{
		Name:    "snaparser",
		Suggest: true,
		Version: "v0.1.4",
		Authors: []*cli.Author{{Name: "vanillaiice", Email: "vanillaiice1@proton.me"}},
		Usage:   "parse snapchat chat history to human friendly format",
		Commands: []*cli.Command{{
			Name:  "import",
			Usage: "create or replace an archive database from an unzipped Snapchat export",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "data", Required: true, Usage: "unchanged Snapchat export root containing json/ and chat_media/"},
				&cli.StringFlag{Name: "db", Value: "snaparser.db", Usage: "archive database to create or replace"},
			},
			Action: importCommand,
		}, {
			Name:  "serve",
			Usage: "browse an imported archive in the embedded web frontend",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "db", Value: "snaparser.db", Usage: "path to an existing archive database"},
				&cli.StringFlag{Name: "data", Usage: "import this unchanged Snapchat export root before serving (replaces the selected archive)"},
				&cli.StringFlag{Name: "listen", Value: "127.0.0.1:8080", Usage: "HTTP listen address"},
			},
			Action: serve,
		}},
	}
}

func importArchive(ctx context.Context, directory, path string) (int, error) {
	// Parse every supplied file before opening the database or running migrations.
	backup, err := parser.LoadBackup(directory)
	if err != nil {
		return 0, err
	}

	store, err := db.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open archive: %w", err)
	}

	defer store.Close()

	if err := store.Import(ctx, backup); err != nil {
		return 0, err
	}

	return len(backup.Filenames), nil
}

func importCommand(c *cli.Context) error {
	count, err := importArchive(c.Context, c.String("data"), c.String("db"))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(c.App.Writer, "Imported %d JSON files into %s\n", count, c.String("db"))

	return err
}

func serve(c *cli.Context) error {
	if c.String("data") != "" {
		if err := importCommand(c); err != nil {
			return err
		}
	}

	path := c.String("db")

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("open archive %q: %w; create it with 'snaparser serve --data /path/to/export', or select an existing database with --db", path, err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("archive %q must be a database file", path)
	}

	store, err := db.Open(path)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}

	defer store.Close()

	viewer, err := frontend.New(store)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(c.Context, os.Interrupt, syscall.SIGTERM)

	defer stop()

	server := &http.Server{
		Addr: c.String("listen"), Handler: viewer,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	done := make(chan struct{})

	defer close(done)

	shutdown := make(chan error, 1)

	go func() {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			defer cancel()

			shutdown <- server.Shutdown(shutdownCtx)

		case <-done:
		}
	}()

	log.Printf("Snaparser archive: http://%s", server.Addr)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return <-shutdown
}
