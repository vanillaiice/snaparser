package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type rowScanner interface {
	Scan(...any) error
}

// GetExportDirectory returns the absolute source root saved during import.
// An empty result means the archive predates source tracking or was imported
// programmatically without a source directory.
func (d *DB) GetExportDirectory(ctx context.Context) (string, error) {
	var directory string

	err := d.conn.QueryRowContext(ctx, "select export_directory from archive_source where id = 1").Scan(&directory)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	return directory, err
}

type rowQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func readRows[T any](ctx context.Context, q rowQuerier, query string, args []any, scan func(rowScanner) (T, error)) ([]T, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	items := make([]T, 0)

	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

const conversationSelect = `select v.id, v.conversation_key,
    (select count(*) from chats where conversation_id = v.id),
    (select count(*) from snaps where conversation_id = v.id),
    max(coalesce((select max(created_microseconds) from chats where conversation_id = v.id), 0),
        coalesce((select max(created_microseconds) from snaps where conversation_id = v.id), 0)) as last_activity
from conversations v`

func scanConversation(row rowScanner) (Conversation, error) {
	var c Conversation

	err := row.Scan(&c.ID, &c.Key, &c.ChatCount, &c.SnapCount, &c.LastActivityMicroseconds)

	return c, err
}

// ListConversations returns the most recently active conversations first,
// breaking timestamp ties by ID. Empty conversations are included.
func (d *DB) ListConversations(ctx context.Context, page Pagination) ([]Conversation, error) {
	limit, offset, err := page.bounds()
	if err != nil {
		return nil, err
	}

	return readRows(ctx, d.conn, conversationSelect+" order by last_activity desc, v.id asc limit ? offset ?", []any{limit, offset}, scanConversation)
}

// GetConversation returns sql.ErrNoRows when the ID does not exist.
func (d *DB) GetConversation(ctx context.Context, id int64) (Conversation, error) {
	if id <= 0 {
		return Conversation{}, fmt.Errorf("%w: conversation ID must be positive", ErrInvalidOptions)
	}

	return scanConversation(d.conn.QueryRowContext(ctx, conversationSelect+" where v.id = ?", id))
}

// ListChats lists newest messages first, with descending ID as a tie breaker.
// With Query set, results are ordered by FTS rank, then timestamp and ID.
func (d *DB) ListChats(ctx context.Context, options MessageOptions) ([]Chat, error) {
	query, args, err := messageQuery("chats", `m.id, m.conversation_id, m.sender, m.media_type, m.created,
        m.content, m.conversation_title, m.is_sender, m.created_microseconds, m.is_saved, m.media_ids`, options)
	if err != nil {
		return nil, err
	}

	return readRows(ctx, d.conn, query, args, func(row rowScanner) (Chat, error) {
		var c Chat

		err := row.Scan(&c.ID, &c.ConversationID, &c.From, &c.MediaType, &c.Created,
			&c.Content, &c.ConversationTitle, &c.IsSender, &c.CreatedMicrosecond, &c.IsSaved, &c.MediaIDS)

		return c, err
	})
}

// GetChatPosition returns sql.ErrNoRows for a missing chat. The lookup uses the
// same timestamp and ID ordering as ListChats without a search query.
func (d *DB) GetChatPosition(ctx context.Context, id int64) (ChatPosition, error) {
	if id <= 0 {
		return ChatPosition{}, fmt.Errorf("%w: chat ID must be positive", ErrInvalidOptions)
	}

	var position ChatPosition

	err := d.conn.QueryRowContext(ctx, `select target.conversation_id,
        (select count(*) from chats newer
         where newer.conversation_id = target.conversation_id
         and (newer.created_microseconds > target.created_microseconds
              or (newer.created_microseconds = target.created_microseconds and newer.id > target.id)))
        from chats target where target.id = ?`, id).Scan(&position.ConversationID, &position.Offset)

	return position, err
}

// ListSnaps lists newest snaps first. Search results use FTS rank followed by
// descending timestamp and ID, as with ListChats.
func (d *DB) ListSnaps(ctx context.Context, options MessageOptions) ([]Snap, error) {
	query, args, err := messageQuery("snaps", `m.id, m.conversation_id, m.sender, m.media_type, m.created,
        m.conversation_title, m.is_sender, m.created_microseconds`, options)
	if err != nil {
		return nil, err
	}

	return readRows(ctx, d.conn, query, args, func(row rowScanner) (Snap, error) {
		var s Snap

		err := row.Scan(&s.ID, &s.ConversationID, &s.From, &s.MediaType, &s.Created,
			&s.ConversationTitle, &s.IsSender, &s.CreatedMicroseconds)

		return s, err
	})
}

// table and columns are supplied only by ListChats and ListSnaps.
func messageQuery(table, columns string, options MessageOptions) (string, []any, error) {
	limit, offset, err := options.Pagination.bounds()
	if err != nil {
		return "", nil, err
	}

	if options.ConversationID < 0 {
		return "", nil, fmt.Errorf("%w: conversation ID must be nonnegative", ErrInvalidOptions)
	}

	query := "select " + columns + " from " + table + " m"

	args := make([]any, 0, 4)

	conditions := make([]string, 0, 2)

	order := "m.created_microseconds desc, m.id desc"

	if text := strings.TrimSpace(options.Query); text != "" {
		fts := table + "_fts"

		query += " join " + fts + " on " + fts + ".rowid = m.id"

		conditions = append(conditions, fts+" match ?")

		args = append(args, text)

		order = fts + ".rank, " + order
	}

	if options.ConversationID != 0 {
		conditions = append(conditions, "m.conversation_id = ?")

		args = append(args, options.ConversationID)
	}

	if len(conditions) != 0 {
		query += " where " + strings.Join(conditions, " and ")
	}

	query += " order by " + order + " limit ? offset ?"

	return query, append(args, limit, offset), nil
}

// ListFriends orders by username (case insensitive), category, then ID. Search
// results are ordered by FTS rank before these tie breakers. Category values
// are the same as those stored by Import, such as friends, blocked, or pending.
func (d *DB) ListFriends(ctx context.Context, options FriendOptions) ([]Friend, error) {
	limit, offset, err := options.Pagination.bounds()
	if err != nil {
		return nil, err
	}

	query := `select f.id, f.category, f.username, f.display_name,
        f.creation_timestamp, f.last_modified_timestamp, f.source from friends f`

	args := make([]any, 0, 4)

	conditions := make([]string, 0, 2)

	order := "f.username collate nocase, f.category, f.id"

	if text := strings.TrimSpace(options.Query); text != "" {
		query += " join friends_fts on friends_fts.rowid = f.id"

		conditions = append(conditions, "friends_fts match ?")

		args = append(args, text)

		order = "friends_fts.rank, " + order
	}

	if options.Category != "" {
		conditions = append(conditions, "f.category = ?")

		args = append(args, options.Category)
	}

	if len(conditions) != 0 {
		query += " where " + strings.Join(conditions, " and ")
	}

	query += " order by " + order + " limit ? offset ?"

	return readRows(ctx, d.conn, query, append(args, limit, offset), func(row rowScanner) (Friend, error) {
		var f Friend

		err := row.Scan(&f.ID, &f.Category, &f.Username, &f.DisplayName, &f.CreationTimestamp, &f.LastModifiedTimestamp, &f.Source)

		return f, err
	})
}

// Search searches all FTS indexes, or one Kind. Results are grouped by kind,
// then ordered by source ID, since FTS ranks from separate indexes are not
// comparable. Use ListChats/ListSnaps/ListFriends for relevance-ranked results.
// Invalid FTS5 syntax returns a database error. Excerpts contain no HTML;
// callers must escape them when rendering HTML.
func (d *DB) Search(ctx context.Context, options SearchOptions) ([]SearchResult, error) {
	limit, offset, err := options.Pagination.bounds()
	if err != nil {
		return nil, err
	}

	text := strings.TrimSpace(options.Query)

	if text == "" {
		return nil, fmt.Errorf("%w: search query must not be blank", ErrInvalidOptions)
	}

	sources := []SearchKind{SearchChats, SearchSnaps, SearchFriends, SearchAccounts,
		SearchSnapProfiles, SearchChannels, SearchAdvertisers, SearchProfileEntries}

	parts := make([]string, 0, len(sources))

	args := make([]any, 0, len(sources)*2+2)

	for _, kind := range sources {
		if options.Kind != "" && options.Kind != kind {
			continue
		}

		table := string(kind)

		fts := table + "_fts"

		conversation := "0"

		join := ""

		if kind == SearchChats || kind == SearchSnaps {
			conversation = "m.conversation_id"

			join = " join " + table + " m on m.id = " + fts + ".rowid"
		}

		parts = append(parts, "select ? as kind, "+fts+".rowid as id, "+conversation+" as conversation_id, "+
			"coalesce(snippet("+fts+", -1, '', '', ' … ', 24), '') as text from "+fts+join+" where "+fts+" match ?")

		args = append(args, string(kind), text)
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: unknown search kind %q", ErrInvalidOptions, options.Kind)
	}

	query := strings.Join(parts, " union all ") + " order by kind, id limit ? offset ?"

	return readRows(ctx, d.conn, query, append(args, limit, offset), func(row rowScanner) (SearchResult, error) {
		var result SearchResult

		err := row.Scan(&result.Kind, &result.ID, &result.ConversationID, &result.Text)

		return result, err
	})
}
