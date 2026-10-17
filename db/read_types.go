package db

import (
	"errors"
	"fmt"

	"github.com/vanillaiice/snaparser/parser"
)

// ErrInvalidOptions identifies invalid pagination, identifiers, or search options.
var ErrInvalidOptions = errors.New("invalid query options")

// Pagination uses an offset into a stable ordering within the current backup.
// A zero Limit defaults to 50; the maximum is 500. Values must be nonnegative.
// Offsets and record IDs should be discarded after importing another backup.
type Pagination struct {
	Limit  int
	Offset int
}

func (p Pagination) bounds() (int, int, error) {
	if p.Limit < 0 || p.Limit > 500 || p.Offset < 0 {
		return 0, 0, fmt.Errorf("%w: limit must be 0..500 and offset must be nonnegative", ErrInvalidOptions)
	}

	limit := p.Limit

	if limit == 0 {
		limit = 50
	}

	return limit, p.Offset, nil
}

// Conversation summarizes both chat and snap history for one backup key.
// LastActivityMicroseconds is zero when neither history has a timestamp.
type Conversation struct {
	ID                       int64
	Key                      string
	ChatCount                int64
	SnapCount                int64
	LastActivityMicroseconds int64
}

// Chat retains nullable message fields and adds database identifiers.
type Chat struct {
	ID             int64
	ConversationID int64
	parser.Chat
}

// ChatPosition locates a chat in its conversation's newest-first ordering.
// Offset counts newer chats, using descending ID to break timestamp ties.
type ChatPosition struct {
	ConversationID int64
	Offset         int
}

// Snap retains nullable metadata and adds database identifiers.
type Snap struct {
	ID              int64
	ConversationID  int64
	ConversationKey string
	parser.Snap
}

// Call retains the exported call fields and its incoming/outgoing direction.
type Call struct {
	ID        int64
	Direction string
	parser.Call
}

// CallOptions filters calls by direction and literal text across exported fields.
type CallOptions struct {
	Pagination
	Direction string
	Query     string
}

// AccountChange is one event from the exported account history.
type AccountChange struct {
	ID     int64
	Kind   string
	Date   string
	Status string
	Value  string
}

// Friend includes the imported friend category and database identifier.
type Friend struct {
	ID       int64
	Category string
	parser.Friend
}

// MessageOptions filters chats or snaps. ConversationID zero selects all
// conversations. Query is an FTS5 expression (words, quoted phrases, prefixes,
// and boolean operators); an empty query lists records without searching.
type MessageOptions struct {
	Pagination
	ConversationID int64
	Query          string
}

// FriendOptions filters friends by imported category and/or FTS5 expression.
// Empty filters select all categories and all names respectively.
type FriendOptions struct {
	Pagination
	Category string
	Query    string
}

// SearchKind identifies a source table in a search result.
type SearchKind string

const (
	SearchChats          SearchKind = "chats"
	SearchSnaps          SearchKind = "snaps"
	SearchFriends        SearchKind = "friends"
	SearchAccounts       SearchKind = "accounts"
	SearchSnapProfiles   SearchKind = "snap_profiles"
	SearchChannels       SearchKind = "discover_channels"
	SearchAdvertisers    SearchKind = "ad_interactions"
	SearchProfileEntries SearchKind = "user_profile_entries"
)

// SearchOptions requires a nonblank FTS5 Query. Kind zero searches all indexed
// tables; otherwise it restricts results to one of the SearchKind constants.
type SearchOptions struct {
	Pagination
	Query string
	Kind  SearchKind
}

// SearchResult identifies a matching source row. Text is a plain-text excerpt
// from the matched field, with no HTML markup. ConversationID is zero for
// sources other than chats and snaps.
type SearchResult struct {
	Kind           SearchKind
	ID             int64
	ConversationID int64
	Text           string
}
