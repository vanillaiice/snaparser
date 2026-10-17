package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vanillaiice/snaparser/parser"
)

func timestampBackup() parser.Backup {
	// Real export shape: the numeric value is milliseconds despite its field name.
	timestamp := time.Date(2026, 10, 12, 1, 29, 14, 708_000_000, time.UTC)

	chats := parser.ChatHistory{
		"milliseconds": {{From: "friend", Created: "2026-10-12 01:29:14 UTC", CreatedMicrosecond: timestamp.UnixMilli()}},
		"microseconds": {{From: "friend", Created: "2026-10-11 01:29:14 UTC", CreatedMicrosecond: timestamp.Add(-24 * time.Hour).UnixMicro()}},
		"unknown":      {{From: "friend"}},
	}

	snaps := parser.SnapHistory{
		"milliseconds": {{CreatedMicroseconds: timestamp.Add(-time.Hour).UnixMilli()}},
	}

	return parser.Backup{ChatHistory: &chats, SnapHistory: &snaps}
}

func checkTimestamps(t *testing.T, store *DB) {
	t.Helper()

	ctx := context.Background()

	expected := time.Date(2026, 10, 12, 1, 29, 14, 708_000_000, time.UTC).UnixMicro()

	chats, err := store.ListChats(ctx, MessageOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(chats) != 3 || chats[0].CreatedMicrosecond != expected || chats[1].CreatedMicrosecond != expected-int64(24*time.Hour/time.Microsecond) || chats[2].CreatedMicrosecond != 0 {
		t.Fatalf("mixed timestamp units must sort by actual date and retain unknown timestamps: %+v", chats)
	}

	snaps, err := store.ListSnaps(ctx, MessageOptions{})
	if err != nil || len(snaps) != 1 || snaps[0].CreatedMicroseconds != expected-int64(time.Hour/time.Microsecond) {
		t.Fatalf("snap timestamps must also be normalized: %+v, %v", snaps, err)
	}

	conversations, err := store.ListConversations(ctx, Pagination{})
	if err != nil || len(conversations) != 3 || conversations[0].Key != "milliseconds" || conversations[0].LastActivityMicroseconds != expected {
		t.Fatalf("conversation dates and ordering must use canonical microseconds: %+v, %v", conversations, err)
	}

	results, err := store.ListChats(ctx, MessageOptions{Query: "friend"})
	if err != nil || len(results) != 3 {
		t.Fatalf("timestamp correction must preserve full-text search: %d results, %v", len(results), err)
	}
}

func TestImportNormalizesTimestampUnits(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	if err := store.Import(context.Background(), timestampBackup()); err != nil {
		t.Fatal(err)
	}

	checkTimestamps(t, store)
}

func TestMigrationCorrectsExistingTimestampUnits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.db")

	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Import(context.Background(), timestampBackup()); err != nil {
		store.Close()

		t.Fatal(err)
	}

	// Recreate version 2's raw millisecond rows, leaving genuine microseconds
	// untouched, so reopening exercises migration of an existing archive.
	for _, query := range []string{
		"update chats set created_microseconds = created_microseconds / 1000 where created = '2026-10-12 01:29:14 UTC'",
		"update snaps set created_microseconds = created_microseconds / 1000",
		"drop table archive_source",
		"update schema_migrations set version = 2, dirty = 0",
	} {
		if _, err := store.conn.Exec(query); err != nil {
			store.Close()

			t.Fatal(err)
		}
	}

	store.Close()

	for range 2 {
		store, err = Open(path)
		if err != nil {
			t.Fatal(err)
		}

		checkTimestamps(t, store)

		store.Close()
	}
}
