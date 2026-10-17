package db

import (
	"context"
	"fmt"
	"strings"
)

// ListCalls orders exported dates newest first, breaking ties by descending ID.
// Query is a literal substring, not a SQL wildcard or FTS expression.
func (d *DB) ListCalls(ctx context.Context, options CallOptions) ([]Call, error) {
	limit, offset, err := options.Pagination.bounds()
	if err != nil {
		return nil, err
	}

	if options.Direction != "" && options.Direction != "incoming" && options.Direction != "outgoing" {
		return nil, fmt.Errorf("%w: unknown call direction", ErrInvalidOptions)
	}

	query := `select id, call_type, date_time, type, people_in_chat, result, city, country, length_seconds, network
        from calls where (? = '' or call_type = ?)`

	args := []any{options.Direction, options.Direction}

	if text := strings.TrimSpace(options.Query); text != "" {
		query += ` and instr(lower(date_time || ' ' || type || ' ' || result || ' ' || city || ' ' || country || ' ' || network), lower(?)) > 0`

		args = append(args, text)
	}

	query += " order by date_time desc, id desc limit ? offset ?"

	return readRows(ctx, d.conn, query, append(args, limit, offset), func(row rowScanner) (Call, error) {
		var call Call

		err := row.Scan(&call.ID, &call.Direction, &call.DateTime, &call.Type, &call.PeopleInChat,
			&call.Result, &call.City, &call.Country, &call.Length, &call.Network)

		return call, err
	})
}

// ListAccountChanges returns account events in import order, by ID.
func (d *DB) ListAccountChanges(ctx context.Context, page Pagination) ([]AccountChange, error) {
	limit, offset, err := page.bounds()
	if err != nil {
		return nil, err
	}

	return readRows(ctx, d.conn, `select id, change_type, date, status, value from account_changes order by id limit ? offset ?`, []any{limit, offset}, func(row rowScanner) (AccountChange, error) {
		var change AccountChange

		err := row.Scan(&change.ID, &change.Kind, &change.Date, &change.Status, &change.Value)

		return change, err
	})
}
