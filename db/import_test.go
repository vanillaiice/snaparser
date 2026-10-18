package db

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vanillaiice/snaparser/v2/parser"
)

func importFixture() parser.Backup {
	text, title := "oldtoken message", "oldtoken conversation"

	chats := parser.ChatHistory{"oldtoken": {
		{From: "oldtoken", Content: &text, ConversationTitle: &title, MediaType: "TEXT", CreatedMicrosecond: 1_700_000_000_000_000, IsSender: true, IsSaved: true},
		{From: "other", MediaType: "PHOTO", MediaIDS: "photo", Created: "2023-11-14 22:13:20 UTC"},
	}}

	snaps := parser.SnapHistory{"oldtoken": {{From: "oldtoken", ConversationTitle: &title, MediaType: "PHOTO", CreatedMicroseconds: 1_700_000_001_000_000}}}

	friend := parser.Friend{Username: "oldtoken", DisplayName: "Old Friend", CreationTimestamp: "created", LastModifiedTimestamp: "modified", Source: "search"}

	return parser.Backup{
		Account: &parser.Account{
			BasicInformation:  parser.BasicInformation{Username: "oldtoken", Name: "Old Owner", CreationDate: "created", RegistrationIP: "127.0.0.1", Country: "QA", LastActive: "active"},
			DeviceInformation: parser.DeviceInformation{Make: "make", ModelID: "id", ModelName: "model", Language: "en", OSType: "os", OSVersion: "1", ConnectionType: "wifi"},
			DeviceHistory:     []parser.DeviceHistory{{Make: "make", Model: "model", StartTime: "start", DeviceType: "phone"}},
			LoginHistory:      []parser.LoginHistory{{IP: "127.0.0.1", Country: "QA", Created: "date", Status: "success", Device: "phone"}},
		},
		AccountHistory: &parser.AccountHistory{
			DisplayNameChange:       []parser.DisplayNameChange{{Date: "date", DisplayName: "oldtoken"}},
			EmailChange:             []parser.EmailChange{{Date: "date", EmailAddress: "old@example.test"}},
			MobileNumberChange:      []parser.MobileNumberChange{{Date: "date", MobileNumber: "123"}},
			PasswordChange:          []parser.PasswordChange{{Date: "date"}},
			SnapchatLinkedToBitmoji: []parser.SnapchatLinkedToBitmoji{{Date: "date"}},
			DownloadMyDataReports:   []parser.DownloadMyDataReports{{Date: "date", Status: "ready", EmailAddress: "old@example.test"}},
		},
		ChatHistory: &chats,
		Friends: &parser.Friends{
			Friends: []parser.Friend{friend}, FriendsRequestsSent: []parser.Friend{friend},
			BlockedUsers: []parser.Friend{friend}, DeletedFriends: []parser.Friend{friend},
			HiddenFriendSuggestions: []parser.Friend{friend}, IgnoredSnapchatters: []parser.Friend{friend}, PendingRequests: []parser.Friend{friend},
		},
		SnapHistory: &snaps,
		SnapProfile: &parser.SnapProfile{Profile: parser.Profile{
			Created: "date", ProfileTitle: "oldtoken", Location: "location", ProfileWebsite: "https://example.test", ProfilePhoto: "photo", HeroImage: "hero",
		}},
		TalkHistory: &parser.TalkHistory{
			IncomingCalls: []parser.Call{{DateTime: "2024-01-01 12:00:00 UTC", Type: "audio", PeopleInChat: 2, Result: "answered", City: "city", Country: "QA", Length: 42, Network: "wifi"}},
			OutgoingCalls: []parser.Call{{DateTime: "2024-01-02 12:00:00 UTC", Type: "video", PeopleInChat: 3, Result: "missed", City: "city", Country: "QA", Network: "mobile"}},
		},
		UserProfile: &parser.UserProfile{
			AppProfile:                parser.AppProfile{Country: "QA", CreationTime: "created", AccountCreationCountry: "QA", PlatformVersion: "1", InAppLanguage: "en"},
			Demographics:              parser.Demographics{CohortAge: "adult", DerivedAdDemographic: "demo"},
			Engagement:                []parser.Engagement{{Event: "opened", Occurrences: 9}},
			DiscoverChannelsViewed:    []parser.DiscoverChannelsViewed{{ChannelName: "oldtoken"}},
			BreakdownOfTimeSpentOnApp: []string{"oldtoken time"},
			AdsYouInteractedWith:      []parser.AdsYouInteractedWith{{AdvertiserName: "oldtoken", Date: "date"}},
			ContentCategories:         []string{"oldtoken content"},
			Interactions:              parser.Interactions{WebInteractions: []string{"oldtoken web"}, AppInteractions: []string{"oldtoken app"}},
			MobileAdID:                "ad-id",
		},
		Filenames:       []string{"/old/export/json/chat_history.json"},
		ExportDirectory: "/old/export",
	}
}

func testStore(t *testing.T) *DB {
	t.Helper()

	store, err := Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { store.Close() })

	return store
}

func TestImportRoundTripsProfilesAndFriends(t *testing.T) {
	store := testStore(t)

	backup := importFixture()

	ctx := context.Background()

	if err := store.Import(ctx, backup); err != nil {
		t.Fatal(err)
	}

	account, err := store.GetAccount(ctx)
	if err != nil || !reflect.DeepEqual(account, backup.Account) {
		t.Fatalf("account round trip: %+v, %v", account, err)
	}

	snap, err := store.GetSnapProfile(ctx)
	if err != nil || !reflect.DeepEqual(snap, backup.SnapProfile) {
		t.Fatalf("snap profile round trip: %+v, %v", snap, err)
	}

	profile, err := store.GetUserProfile(ctx)
	if err != nil || !reflect.DeepEqual(profile, backup.UserProfile) {
		t.Fatalf("user profile round trip: %+v, %v", profile, err)
	}

	for _, category := range []string{"friends", "requests_sent", "blocked", "deleted", "hidden_suggestions", "ignored", "pending"} {
		friends, err := store.ListFriends(ctx, FriendOptions{Category: category, Query: "oldtoken"})
		if err != nil || len(friends) != 1 || friends[0].Category != category || friends[0].Friend != backup.Friends.Friends[0] {
			t.Fatalf("friend category %q round trip: %+v, %v", category, friends, err)
		}
	}

	chats, err := store.ListChats(ctx, MessageOptions{})
	if err != nil || len(chats) != 2 || !reflect.DeepEqual(chats[0].Chat, (*backup.ChatHistory)["oldtoken"][0]) || chats[1].Content != nil || chats[1].ConversationTitle != nil {
		t.Fatalf("chat fields and nullability lost: %+v, %v", chats, err)
	}

	snaps, err := store.ListSnaps(ctx, MessageOptions{Query: "oldtoken"})
	if err != nil || len(snaps) != 1 || !reflect.DeepEqual(snaps[0].Snap, (*backup.SnapHistory)["oldtoken"][0]) || snaps[0].ConversationID != chats[0].ConversationID {
		t.Fatalf("snap round trip/shared conversation: %+v, %v", snaps, err)
	}
}

func databaseSnapshot(t *testing.T, store *DB) map[string][][]any {
	t.Helper()

	tables, err := readRows(context.Background(), store.conn, "select name from sqlite_schema where type = 'table' and name not like '%_fts%' order by name", nil, func(row rowScanner) (string, error) {
		var name string

		err := row.Scan(&name)

		return name, err
	})
	if err != nil {
		t.Fatal(err)
	}

	snapshot := make(map[string][][]any)

	for _, table := range tables {
		rows, err := store.conn.Query(`select * from "` + table + `" order by rowid`)
		if err != nil {
			t.Fatal(err)
		}

		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}

		snapshot[table] = nil

		for rows.Next() {
			values := make([]any, len(columns))

			pointers := make([]any, len(columns))

			for i := range values {
				pointers[i] = &values[i]
			}

			if err := rows.Scan(pointers...); err != nil {
				t.Fatal(err)
			}

			snapshot[table] = append(snapshot[table], values)
		}

		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}

		rows.Close()
	}

	return snapshot
}

func TestImportReplacesWholeBackupAndSearchIndexes(t *testing.T) {
	store := testStore(t)

	ctx := context.Background()

	if err := store.Import(ctx, importFixture()); err != nil {
		t.Fatal(err)
	}

	before := databaseSnapshot(t, store)

	schema := func() []string {
		objects, err := readRows(ctx, store.conn, "select type || ':' || name || ':' || coalesce(sql, '') from sqlite_schema order by name", nil, func(row rowScanner) (string, error) {
			var object string

			err := row.Scan(&object)

			return object, err
		})
		if err != nil {
			t.Fatal(err)
		}

		return objects
	}

	schemaBefore := schema()

	text := "newtoken"

	chats := parser.ChatHistory{"new conversation": {{Content: &text}}}

	for range 2 {
		if err := store.Import(ctx, parser.Backup{ChatHistory: &chats}); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(schemaBefore, schema()) {
			t.Fatal("replacement changed schema, indexes, or triggers")
		}

		for table, rows := range databaseSnapshot(t, store) {
			want := 0

			switch table {
			case "schema_migrations":
				if !reflect.DeepEqual(rows, before[table]) {
					t.Fatal("replacement changed migration history")
				}

				continue

			case "chats", "conversations", "import_history":
				want = 1
			}

			if len(rows) != want {
				t.Errorf("%s has %d rows after replacement, want %d", table, len(rows), want)
			}
		}

		old, err := store.Search(ctx, SearchOptions{Query: "oldtoken"})
		if err != nil || len(old) != 0 {
			t.Fatalf("stale full-text results: %+v, %v", old, err)
		}

		current, err := store.Search(ctx, SearchOptions{Query: "newtoken"})
		if err != nil || len(current) != 1 || current[0].Kind != SearchChats {
			t.Fatalf("replacement is not searchable: %+v, %v", current, err)
		}

		if _, err := store.GetAccount(ctx); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("omitted account must be removed: %v", err)
		}
	}
}

func TestFailedImportRestoresEveryTableAndSearchIndex(t *testing.T) {
	store := testStore(t)

	ctx := context.Background()

	if err := store.Import(ctx, importFixture()); err != nil {
		t.Fatal(err)
	}

	before := databaseSnapshot(t, store)

	searchBefore, err := store.Search(ctx, SearchOptions{Query: "oldtoken", Pagination: Pagination{Limit: 500}})
	if err != nil {
		t.Fatal(err)
	}

	kinds := make(map[SearchKind]bool)

	for _, result := range searchBefore {
		kinds[result.Kind] = true
	}

	for _, kind := range []SearchKind{SearchChats, SearchSnaps, SearchFriends, SearchAccounts, SearchSnapProfiles, SearchChannels, SearchAdvertisers, SearchProfileEntries} {
		if !kinds[kind] {
			t.Fatalf("fixture must populate search index %s", kind)
		}
	}

	// Fail after all source tables and FTS triggers have been changed.
	if _, err := store.conn.Exec(`create trigger reject_import before insert on archive_source begin select raise(abort, 'test failure'); end`); err != nil {
		t.Fatal(err)
	}

	replacement := importFixture()

	text := "newtoken"

	(*replacement.ChatHistory)["oldtoken"][0].Content = &text

	if err := store.Import(ctx, replacement); err == nil {
		t.Fatal("expected replacement to fail")
	}

	if after := databaseSnapshot(t, store); !reflect.DeepEqual(before, after) {
		t.Fatal("failed replacement changed stored backup or import metadata")
	}

	searchAfter, err := store.Search(ctx, SearchOptions{Query: "oldtoken", Pagination: Pagination{Limit: 500}})
	if err != nil || !reflect.DeepEqual(searchBefore, searchAfter) {
		t.Fatalf("rollback changed old search results: %+v, %v", searchAfter, err)
	}

	current, err := store.Search(ctx, SearchOptions{Query: "newtoken"})
	if err != nil || len(current) != 0 {
		t.Fatalf("rolled-back message leaked into search: %+v, %v", current, err)
	}

	if err := store.Import(ctx, parser.Backup{}); err == nil {
		t.Fatal("empty backup accepted")
	}

	if after := databaseSnapshot(t, store); !reflect.DeepEqual(before, after) {
		t.Fatal("empty backup changed the archive")
	}
}
