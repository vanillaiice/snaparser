package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanillaiice/snaparser/db"
)

func exportFile(t *testing.T, directory, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(directory, "json", name), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func exportRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	for _, name := range []string{"json", "chat_media"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}

	for _, name := range []string{"account.json", "account_history.json", "chat_history.json", "friends.json", "snap_history.json", "snap_pro.json", "talk_history.json", "user_profile.json"} {
		exportFile(t, root, name, `{}`)
	}

	return root
}

func TestImportCommandCreatesSearchableArchive(t *testing.T) {
	root := exportRoot(t)

	directory := root

	exportFile(t, directory, "account.json", `{"Basic Information":{"Username":"owner","Name":"Archive Owner"}}`)

	exportFile(t, directory, "chat_history.json", `{"alex":[{"From":"alex","Media Type":"TEXT","Content":"hello from the export","Created(microseconds)":1718190000000000}]}`)

	exportFile(t, directory, "bitmoji.json", `{"unsupported":"ignored"}`)

	path := filepath.Join(root, "archive.db")

	app := newApp()

	var output bytes.Buffer

	app.Writer = &output

	if err := app.Run(context.Background(), []string{"snaparser", "import", "--data", root, "--db", path}); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output.String(), "Imported 8 JSON files") {
		t.Fatalf("unexpected import output: %s", output.String())
	}

	store, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	if source, err := store.GetExportDirectory(context.Background()); err != nil || source != root {
		t.Fatalf("export root was not saved: %q, %v", source, err)
	}

	account, err := store.GetAccount(context.Background())
	if err != nil || account.BasicInformation.Username != "owner" {
		t.Fatalf("account was not imported: %v", err)
	}

	chats, err := store.ListChats(context.Background(), db.MessageOptions{Query: "hello"})
	if err != nil || len(chats) != 1 || chats[0].Content == nil || *chats[0].Content != "hello from the export" {
		t.Fatalf("imported chats must be searchable: %v", err)
	}
}

func TestFailedImportPreservesArchive(t *testing.T) {
	directory := exportRoot(t)

	exportFile(t, directory, "account.json", `{"Basic Information":{"Username":"original"}}`)

	path := filepath.Join(t.TempDir(), "archive.db")

	if _, err := importArchive(context.Background(), directory, path); err != nil {
		t.Fatal(err)
	}

	exportFile(t, directory, "account.json", `{"Basic Information":{"Username":"replacement"}}`)

	exportFile(t, directory, "chat_history.json", `{"broken":`)

	if _, err := importArchive(context.Background(), directory, path); err == nil {
		t.Fatal("a malformed supplied file must fail the import")
	}

	store, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	account, err := store.GetAccount(context.Background())
	if err != nil || account.BasicInformation.Username != "original" {
		t.Fatalf("a parsing failure must preserve the previous backup: %v", err)
	}

	newPath := filepath.Join(t.TempDir(), "not-created.db")

	if _, err := importArchive(context.Background(), directory, newPath); err == nil {
		t.Fatal("expected malformed input to fail")
	}

	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatal("invalid input must not create a new database")
	}
}

func TestInvalidExportLayoutPreservesArchive(t *testing.T) {
	ctx := context.Background()

	original := exportRoot(t)

	exportFile(t, original, "account.json", `{"Basic Information":{"Username":"original"}}`)

	path := filepath.Join(t.TempDir(), "archive.db")

	if _, err := importArchive(ctx, original, path); err != nil {
		t.Fatal(err)
	}

	for _, kind := range []string{"json directory", "missing JSON", "missing media", "invalid additional JSON"} {
		t.Run(kind, func(t *testing.T) {
			root := exportRoot(t)

			exportFile(t, root, "account.json", `{"Basic Information":{"Username":"replacement"}}`)

			var err error

			switch kind {
			case "json directory":
				root = filepath.Join(root, "json")

			case "missing JSON":
				err = os.Remove(filepath.Join(root, "json", "friends.json"))

			case "missing media":
				err = os.Remove(filepath.Join(root, "chat_media"))

			case "invalid additional JSON":
				exportFile(t, root, "bitmoji.json", `{"broken":`)
			}

			if err != nil {
				t.Fatal(err)
			}

			if _, err := importArchive(ctx, root, path); err == nil {
				t.Fatal("invalid export must abort")
			}

			store, err := db.Open(path)
			if err != nil {
				t.Fatal(err)
			}

			defer store.Close()

			account, err := store.GetAccount(ctx)
			if err != nil || account.BasicInformation.Username != "original" {
				t.Fatalf("archive changed: %v", err)
			}

			if source, err := store.GetExportDirectory(ctx); err != nil || source != original {
				t.Fatalf("source changed: %q, %v", source, err)
			}

			newPath := filepath.Join(t.TempDir(), "uncreated.db")

			if _, err := importArchive(ctx, root, newPath); err == nil {
				t.Fatal("invalid export must abort before opening database")
			}

			if _, err := os.Stat(newPath); !os.IsNotExist(err) {
				t.Fatal("invalid export created a database")
			}
		})
	}
}

func TestServeDataImportsBeforeStartingServer(t *testing.T) {
	directory := exportRoot(t)

	exportFile(t, directory, "account.json", `{"Basic Information":{"Username":"ready-to-browse"}}`)

	path := filepath.Join(t.TempDir(), "archive.db")

	app := newApp()

	var output bytes.Buffer

	app.Writer = &output

	// An invalid address lets the action finish without binding a test port.
	err := app.Run(context.Background(), []string{"snaparser", "serve", "--data", directory, "--db", path, "--listen", "invalid-address"})
	if err == nil || !strings.Contains(err.Error(), "missing port") {
		t.Fatalf("expected the action to reach server startup: %v", err)
	}

	store, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	account, err := store.GetAccount(context.Background())
	if err != nil || account.BasicInformation.Username != "ready-to-browse" {
		t.Fatalf("serve --data must create and populate the archive first: %v", err)
	}
}
