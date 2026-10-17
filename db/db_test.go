package db

import (
	"context"
	"testing"
)

func TestOpen(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("close", func(t *testing.T) {
		if err = db.Close(); err != nil {
			t.Error(err)
		}
	})
}

func TestInMemoryArchiveCanImportAndSearch(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	if err := store.Import(context.Background(), importFixture()); err != nil {
		t.Fatal(err)
	}

	matches, err := store.ListChats(context.Background(), MessageOptions{Query: "oldtoken"})
	if err != nil || len(matches) != 1 {
		t.Fatalf("in-memory archive lost its migrated schema: %+v, %v", matches, err)
	}
}
