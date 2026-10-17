package db

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/vanillaiice/snaparser/parser"
)

func TestChatPositionMatchesConversationOrder(t *testing.T) {
	ctx := context.Background()

	store, err := Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	history := parser.ChatHistory{
		"conversation": {
			{CreatedMicrosecond: 1_700_000_000_000_000},
			{CreatedMicrosecond: 1_700_000_000_000_000},
			{CreatedMicrosecond: 1_700_000_001_000_000},
			{}, {},
		},
		"another": {{CreatedMicrosecond: 1_800_000_000_000_000}},
	}

	if err := store.Import(ctx, parser.Backup{ChatHistory: &history}); err != nil {
		t.Fatal(err)
	}

	conversations, err := store.ListConversations(ctx, Pagination{})
	if err != nil {
		t.Fatal(err)
	}

	for _, conversation := range conversations {
		chats, err := store.ListChats(ctx, MessageOptions{ConversationID: conversation.ID})
		if err != nil {
			t.Fatal(err)
		}

		for offset, chat := range chats {
			position, err := store.GetChatPosition(ctx, chat.ID)
			if err != nil || position.ConversationID != conversation.ID || position.Offset != offset {
				t.Fatalf("chat %d: position = %+v, error = %v; want conversation %d, offset %d", chat.ID, position, err, conversation.ID, offset)
			}
		}
	}

	if _, err := store.GetChatPosition(ctx, 999999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing chat returned %v", err)
	}

	for _, id := range []int64{0, -1} {
		if _, err := store.GetChatPosition(ctx, id); !errors.Is(err, ErrInvalidOptions) {
			t.Fatalf("invalid chat ID %d returned %v", id, err)
		}
	}
}
