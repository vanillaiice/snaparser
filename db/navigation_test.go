package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/vanillaiice/snaparser/parser"
)

func TestFriendConversationMatchesUsername(t *testing.T) {
	store := testStore(t)

	backup := importFixture()

	backup.Friends.Friends = append(backup.Friends.Friends,
		parser.Friend{Username: "missing", DisplayName: "oldtoken"},
		parser.Friend{Username: "oldtoken.suffix", DisplayName: "oldtoken"},
	)

	ctx := context.Background()

	if err := store.Import(ctx, backup); err != nil {
		t.Fatal(err)
	}

	conversations, err := store.ListConversations(ctx, Pagination{})
	if err != nil || len(conversations) != 1 {
		t.Fatalf("fixture conversation: %+v, %v", conversations, err)
	}

	friends, err := store.ListFriends(ctx, FriendOptions{Query: "oldtoken", Pagination: Pagination{Limit: 500}})
	if err != nil || len(friends) != 9 {
		t.Fatalf("friend search: %+v, %v", friends, err)
	}

	for _, friend := range friends {
		want := int64(0)

		if friend.Username == "oldtoken" {
			want = conversations[0].ID
		}

		if friend.ConversationID != want {
			t.Errorf("friend %q (%s): conversation %d, want %d", friend.Username, friend.Category, friend.ConversationID, want)
		}
	}
}

func TestSnapConversationsAndPositions(t *testing.T) {
	store := testStore(t)

	ctx := context.Background()

	chats := parser.ChatHistory{
		"chat.only":  {{CreatedMicrosecond: 1_900_000_000_000_000}},
		"older.snap": {{CreatedMicrosecond: 1_900_000_000_000_000}},
	}

	snaps := parser.SnapHistory{
		"older.snap": {{CreatedMicroseconds: 1_700_000_000_000_000}, {CreatedMicroseconds: 1_700_000_000_000_000}, {}, {}},
		"newer.snap": {{CreatedMicroseconds: 1_800_000_000_000_000}},
	}

	if err := store.Import(ctx, parser.Backup{ChatHistory: &chats, SnapHistory: &snaps}); err != nil {
		t.Fatal(err)
	}

	conversations, err := store.ListSnapConversations(ctx, Pagination{})
	if err != nil || len(conversations) != 2 || conversations[0].Key != "newer.snap" || conversations[1].Key != "older.snap" || conversations[1].SnapCount != 4 || conversations[1].LastActivityMicroseconds != 1_700_000_000_000_000 {
		t.Fatalf("snap sidebar must use snap activity and omit chat-only conversations: %+v, %v", conversations, err)
	}

	page, err := store.ListSnapConversations(ctx, Pagination{Limit: 1, Offset: 1})
	if err != nil || len(page) != 1 || page[0] != conversations[1] {
		t.Fatalf("snap conversation pagination: %+v, %v", page, err)
	}

	for _, conversation := range conversations {
		items, err := store.ListSnaps(ctx, MessageOptions{ConversationID: conversation.ID})
		if err != nil {
			t.Fatal(err)
		}

		for offset, item := range items {
			position, err := store.GetSnapPosition(ctx, item.ID)
			if err != nil || position.ConversationID != conversation.ID || position.Offset != offset {
				t.Fatalf("snap %d: position %+v, %v; want offset %d in conversation %d", item.ID, position, err, offset, conversation.ID)
			}
		}
	}

	if _, err := store.GetSnapPosition(ctx, 999999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing snap returned %v", err)
	}

	if _, err := store.GetSnapPosition(ctx, -1); !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("invalid snap returned %v", err)
	}

	if _, err := store.ListSnapConversations(ctx, Pagination{Offset: -1}); !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("invalid sidebar pagination returned %v", err)
	}
}
