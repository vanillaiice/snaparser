package db

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestCallsRoundTripFilteringAndPagination(t *testing.T) {
	store := testStore(t)

	backup := importFixture()

	backup.TalkHistory.OutgoingCalls[0].Type = "video_100%"

	ctx := context.Background()

	if err := store.Import(ctx, backup); err != nil {
		t.Fatal(err)
	}

	calls, err := store.ListCalls(ctx, CallOptions{})
	if err != nil || len(calls) != 2 || calls[0].Direction != "outgoing" || calls[1].Direction != "incoming" {
		t.Fatalf("call order/directions: %+v, %v", calls, err)
	}

	if !reflect.DeepEqual(calls[0].Call, backup.TalkHistory.OutgoingCalls[0]) || !reflect.DeepEqual(calls[1].Call, backup.TalkHistory.IncomingCalls[0]) {
		t.Fatal("call fields changed during import")
	}

	for _, tc := range []struct {
		options CallOptions
		count   int
	}{
		{CallOptions{Direction: "incoming"}, 1},
		{CallOptions{Direction: "outgoing", Query: "VIDEO_100%"}, 1},
		{CallOptions{Direction: "incoming", Query: "video"}, 0},
		{CallOptions{Query: "WIFI"}, 1},
		{CallOptions{Query: "%"}, 1},
		{CallOptions{Query: "' OR 1=1 --"}, 0},
		{CallOptions{Pagination: Pagination{Limit: 1, Offset: 1}}, 1},
		{CallOptions{Pagination: Pagination{Offset: 2}}, 0},
	} {
		got, err := store.ListCalls(ctx, tc.options)
		if err != nil || len(got) != tc.count {
			t.Errorf("options %+v: %d calls, %v; want %d", tc.options, len(got), err, tc.count)
		}
	}

	for _, options := range []CallOptions{{Direction: "invalid"}, {Pagination: Pagination{Limit: 501}}, {Pagination: Pagination{Offset: -1}}} {
		if _, err := store.ListCalls(ctx, options); !errors.Is(err, ErrInvalidOptions) {
			t.Errorf("invalid call options %+v returned %v", options, err)
		}
	}
}

func TestAccountChangesPreserveExportedValuesAndPageOrder(t *testing.T) {
	store := testStore(t)

	ctx := context.Background()

	if err := store.Import(ctx, importFixture()); err != nil {
		t.Fatal(err)
	}

	changes, err := store.ListAccountChanges(ctx, Pagination{})
	if err != nil || len(changes) != 6 {
		t.Fatalf("account changes: %+v, %v", changes, err)
	}

	want := []AccountChange{
		{1, "display_name", "date", "", "oldtoken"},
		{2, "email", "date", "", "old@example.test"},
		{3, "mobile_number", "date", "", "123"},
		{4, "password", "date", "", ""},
		{5, "bitmoji_linked", "date", "", ""},
		{6, "data_report", "date", "ready", "old@example.test"},
	}

	if !reflect.DeepEqual(changes, want) {
		t.Fatalf("account changes = %+v, want %+v", changes, want)
	}

	page, err := store.ListAccountChanges(ctx, Pagination{Limit: 2, Offset: 2})
	if err != nil || !reflect.DeepEqual(page, want[2:4]) {
		t.Fatalf("history page = %+v, %v", page, err)
	}

	if _, err := store.ListAccountChanges(ctx, Pagination{Offset: -1}); !errors.Is(err, ErrInvalidOptions) {
		t.Fatalf("invalid history pagination returned %v", err)
	}
}
