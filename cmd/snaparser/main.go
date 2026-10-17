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
	"strings"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"
	"github.com/vanillaiice/snaparser/db"
	"github.com/vanillaiice/snaparser/frontend"
	"github.com/vanillaiice/snaparser/parser"
)

const version = "1.0.0"

func main() {
	if err := newApp().Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func newApp() *cli.Command {
	var configPath string

	source := configSources(&configPath)

	return &cli.Command{
		Name:    "snaparser",
		Suggest: true,
		Version: version,
		Authors: []any{"vanillaiice <vanillaiice1@proton.me>"},
		Usage:   "browse and search a local Snapchat export",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "config", Aliases: []string{"c"}, Value: defaultConfigFile,
				Destination: &configPath, Usage: "TOML configuration file (optional at the default path)",
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			return ctx, validateConfig(configPath, c.IsSet("config"))
		},
		Commands: []*cli.Command{
			{
				Name:  "import",
				Usage: "create or replace an archive database from an unzipped Snapchat export",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "data", Sources: source("data"), Usage: "unchanged Snapchat export root containing json/ and chat_media/"},
					databaseFlag(source("db")),
				},
				Action: importCommand,
			},
			{
				Name:  "serve",
				Usage: "browse an imported archive in the embedded web frontend",
				Flags: []cli.Flag{
					databaseFlag(source("db")),
					&cli.StringFlag{Name: "data", Sources: source("data"), Usage: "import this export before serving (replaces the selected archive)"},
					&cli.StringFlag{
						Name: "listen", Value: "127.0.0.1:8080", Sources: source("server.listen"),
						Usage: "HTTP listen address",
						Validator: func(address string) error {
							if strings.TrimSpace(address) == "" {
								return errors.New("listen address must not be empty")
							}

							return nil
						},
					},
					timeoutFlag("read-header-timeout", 5*time.Second, source),
					timeoutFlag("read-timeout", 15*time.Second, source),
					timeoutFlag("write-timeout", 30*time.Second, source),
					timeoutFlag("idle-timeout", 60*time.Second, source),
					timeoutFlag("shutdown-timeout", 5*time.Second, source),
				},
				Action: serve,
			},
		},
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

func importCommand(ctx context.Context, c *cli.Command) error {
	if strings.TrimSpace(c.String("data")) == "" {
		return errors.New("import requires --data or a nonempty data value in the TOML configuration")
	}

	count, err := importArchive(ctx, c.String("data"), c.String("db"))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(c.Root().Writer, "Imported %d JSON files into %s\n", count, c.String("db"))

	return err
}

func serve(ctx context.Context, c *cli.Command) error {
	if c.String("data") != "" {
		if err := importCommand(ctx, c); err != nil {
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

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)

	defer stop()

	server := &http.Server{
		Addr: c.String("listen"), Handler: viewer,
		ReadHeaderTimeout: c.Duration("read-header-timeout"),
		ReadTimeout:       c.Duration("read-timeout"),
		WriteTimeout:      c.Duration("write-timeout"),
		IdleTimeout:       c.Duration("idle-timeout"),
	}

	done := make(chan struct{})

	defer close(done)

	shutdown := make(chan error, 1)

	go func() {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), c.Duration("shutdown-timeout"))

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
