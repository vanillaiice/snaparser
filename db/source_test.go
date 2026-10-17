package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vanillaiice/snaparser/parser"
)

func TestArchiveSourceSurvivesReopenAndRollsBack(t *testing.T) {
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "archive.db")

	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	backup := parser.Backup{Account: &parser.Account{}, ExportDirectory: "/original/export"}

	if err := store.Import(ctx, backup); err != nil {
		t.Fatal(err)
	}

	store.Close()

	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	if source, err := store.GetExportDirectory(ctx); err != nil || source != backup.ExportDirectory {
		t.Fatalf("persisted source = %q, %v", source, err)
	}

	if _, err := store.conn.Exec(`create trigger reject_source before insert on archive_source begin select raise(abort, 'test failure'); end`); err != nil {
		t.Fatal(err)
	}

	backup.ExportDirectory = "/replacement/export"

	if err := store.Import(ctx, backup); err == nil {
		t.Fatal("expected source insertion to fail")
	}

	if source, err := store.GetExportDirectory(ctx); err != nil || source != "/original/export" {
		t.Fatalf("failed replacement changed source: %q, %v", source, err)
	}

	if _, err := store.conn.Exec("drop trigger reject_source"); err != nil {
		t.Fatal(err)
	}

	backup.ExportDirectory = ""

	if err := store.Import(ctx, backup); err != nil {
		t.Fatal(err)
	}

	if source, err := store.GetExportDirectory(ctx); err != nil || source != "" {
		t.Fatalf("replacement retained stale source: %q, %v", source, err)
	}
}
