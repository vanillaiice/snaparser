package frontend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanillaiice/snaparser/db"
	"github.com/vanillaiice/snaparser/parser"
)

func testDetailsArchive(t *testing.T) (http.Handler, *db.DB) {
	t.Helper()

	store, err := db.Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { store.Close() })

	friends := &parser.Friends{BlockedUsers: []parser.Friend{{Username: "blocked.person", DisplayName: `<script>bad()</script>`}}}

	snaps := parser.SnapHistory{}

	calls := &parser.TalkHistory{OutgoingCalls: []parser.Call{{DateTime: "2025-01-01 12:00:00 UTC", Type: "video_100%", Result: "missed"}}}

	history := &parser.AccountHistory{}

	for i := range 45 {
		friends.Friends = append(friends.Friends, parser.Friend{Username: fmt.Sprintf("person.%02d", i), DisplayName: "Friendly Person", Source: "search"})

		snaps["alex"] = append(snaps["alex"], parser.Snap{From: "alex", MediaType: "PHOTO", CreatedMicroseconds: 1_700_000_000_000_000 + int64(i)*1_000_000, IsSender: i%2 == 0})

		calls.IncomingCalls = append(calls.IncomingCalls, parser.Call{DateTime: fmt.Sprintf("2024-01-01 12:%02d:00 UTC", i), Type: "audio", Result: "answered", PeopleInChat: 2, Length: int32(i), Network: "wifi"})

		history.DisplayNameChange = append(history.DisplayNameChange, parser.DisplayNameChange{Date: "date", DisplayName: fmt.Sprintf("Changed-%02d", i)})
	}

	snaps["other"] = []parser.Snap{{From: "other", MediaType: "VIDEO", CreatedMicroseconds: 1_800_000_000_000_000}}

	backup := parser.Backup{
		Account: &parser.Account{
			BasicInformation:  parser.BasicInformation{Username: "owner", Name: `<img src=x onerror="bad()">`, Country: "QA"},
			DeviceInformation: parser.DeviceInformation{Make: "Test Make", ModelName: "Test Model"},
			DeviceHistory:     []parser.DeviceHistory{{Model: "Historic Model"}},
			LoginHistory:      []parser.LoginHistory{{IP: "127.0.0.1", Status: "success"}},
		},
		AccountHistory: history, Friends: friends, SnapHistory: &snaps, TalkHistory: calls,
		SnapProfile: &parser.SnapProfile{Profile: parser.Profile{ProfileTitle: "Public Profile", ProfileWebsite: "https://example.test"}},
		UserProfile: &parser.UserProfile{
			AppProfile:             parser.AppProfile{Country: "QA", PlatformVersion: "Test Platform"},
			Engagement:             []parser.Engagement{{Event: "App Opens", Occurrences: 3}},
			DiscoverChannelsViewed: []parser.DiscoverChannelsViewed{{ChannelName: "Test Channel"}},
			AdsYouInteractedWith:   []parser.AdsYouInteractedWith{{AdvertiserName: "Test Advertiser"}},
			ContentCategories:      []string{"Test Category"}, BreakdownOfTimeSpentOnApp: []string{"Test Time"},
			Interactions: parser.Interactions{WebInteractions: []string{"Test Web"}, AppInteractions: []string{"Test App"}},
		},
	}

	if err := store.Import(context.Background(), backup); err != nil {
		t.Fatal(err)
	}

	h, err := New(store)
	if err != nil {
		t.Fatal(err)
	}

	return h, store
}

func TestArchiveSectionsRenderFullDocumentsAndFragments(t *testing.T) {
	h, _ := testDetailsArchive(t)

	for _, section := range []string{"friends", "snaps", "calls", "account"} {
		t.Run(section, func(t *testing.T) {
			full := get(h, "/"+section)

			body := full.Body.String()

			if full.Code != 200 || !strings.Contains(body, "<!doctype html>") || !strings.Contains(body, `id="records"`) || !strings.Contains(body, `href="/`+section+`" aria-current="page"`) {
				t.Fatalf("section document failed: %d", full.Code)
			}

			fragment := get(h, "/"+section, "HX-Request", "true")
			if fragment.Code != 200 || strings.Contains(fragment.Body.String(), "<!doctype") || !strings.Contains(fragment.Body.String(), `id="workspace"`) {
				t.Fatalf("invalid section fragment: %d", fragment.Code)
			}

			restored := get(h, "/"+section, "HX-Request", "true", "HX-History-Restore-Request", "true")
			if !strings.Contains(restored.Body.String(), "<!doctype html>") {
				t.Fatal("history restore must return a document")
			}

			if full.Header().Get("Cache-Control") != "no-store" || !strings.Contains(full.Header().Get("Vary"), "HX-Request") {
				t.Fatal("detail pages must retain archive cache headers")
			}
		})
	}
}

func TestFriendsSearchCategoriesAndPagination(t *testing.T) {
	h, _ := testDetailsArchive(t)

	first := get(h, "/friends?q=person&filter=friends").Body.String()

	if strings.Count(first, "Friendly Person") != 40 || !strings.Contains(first, "/friends?filter=friends&amp;offset=40&amp;q=person") || strings.Contains(first, "blocked.person") {
		t.Fatal("friend search must preserve category and page at 40 rows")
	}

	second := get(h, "/friends?q=person&filter=friends&offset=40").Body.String()

	if strings.Count(second, "Friendly Person") != 5 || !strings.Contains(second, "41–45") || !strings.Contains(second, "/friends?filter=friends&amp;q=person") {
		t.Fatal("friend pagination lost results or filters")
	}

	blocked := get(h, "/friends?filter=blocked").Body.String()

	if !strings.Contains(blocked, "blocked.person") || !strings.Contains(blocked, "&lt;script&gt;") || strings.Contains(blocked, "<script>bad()") || strings.Contains(blocked, "Friendly Person") {
		t.Fatal("blocked friends must be filtered and escaped")
	}

	for _, query := range []string{`"`, `*`, `person OR (`, `' OR 1=1 --`} {
		if response := get(h, "/friends?q="+url.QueryEscape(query)); response.Code != 200 {
			t.Errorf("literal friend query %q returned %d", query, response.Code)
		}
	}
}

func TestSnapsScopeOrderingAndConversationLinks(t *testing.T) {
	h, store := testDetailsArchive(t)

	conversations, err := store.ListConversations(context.Background(), db.Pagination{})
	if err != nil {
		t.Fatal(err)
	}

	var alexID int64

	for _, conversation := range conversations {
		if conversation.Key == "alex" {
			alexID = conversation.ID
		}
	}

	target := fmt.Sprintf("/snaps?conversation=%d&q=alex", alexID)

	body := get(h, target).Body.String()

	if strings.Count(body, "PHOTO</td>") != 40 || strings.Contains(body, "VIDEO</td>") || !strings.Contains(body, fmt.Sprintf(`href="/?conversation=%d"`, alexID)) || !strings.Contains(body, "Incoming</td>") || !strings.Contains(body, "Outgoing</td>") {
		t.Fatal("snaps must be scoped and link to unfiltered chat conversations")
	}

	second := get(h, target+"&offset=40").Body.String()

	if strings.Count(second, "PHOTO</td>") != 5 || !strings.Contains(second, "41–45") {
		t.Fatal("snap pagination failed")
	}

	if response := get(h, "/snaps?conversation=999999"); response.Code != 404 {
		t.Fatal("missing snap conversations should return 404")
	}
}

func TestCallsFilteringAndPagination(t *testing.T) {
	h, _ := testDetailsArchive(t)

	body := get(h, "/calls?filter=incoming&q=audio").Body.String()

	if strings.Count(body, "answered</td>") != 40 || strings.Contains(body, "missed</td>") || !strings.Contains(body, "/calls?filter=incoming&amp;offset=40&amp;q=audio") {
		t.Fatal("calls must filter, search, and paginate")
	}

	second := get(h, "/calls?filter=incoming&q=audio&offset=40").Body.String()

	if strings.Count(second, "answered</td>") != 5 || !strings.Contains(second, "41–45") {
		t.Fatal("call pagination failed")
	}

	outgoing := get(h, "/calls?filter=outgoing&q="+url.QueryEscape("_100%")).Body.String()

	if !strings.Contains(outgoing, "video_100%") || strings.Contains(outgoing, "answered</td>") {
		t.Fatal("call search must use literal text and direction")
	}
}

func TestAccountDetailsAndHistoryPagination(t *testing.T) {
	h, _ := testDetailsArchive(t)

	body := get(h, "/account").Body.String()

	for _, want := range []string{"Account details", "owner", "Test Model", "Historic Model", "127.0.0.1", "Public Profile", "Test Platform", "App Opens", "Test Channel", "Test Advertiser", "Test Category", "Test Time", "Test Web", "Test App", "Account history", "Changed-00", "Changed-39", "&lt;img"} {
		if !strings.Contains(body, want) {
			t.Errorf("account view missing %q", want)
		}
	}

	if strings.Contains(body, `<img src=x onerror=`) || strings.Contains(body, "Changed-40") {
		t.Fatal("account data must be escaped and history paginated")
	}

	second := get(h, "/account?offset=40").Body.String()

	if !strings.Contains(second, "Changed-44") || strings.Contains(second, "Changed-39") || !strings.Contains(second, "41–45") {
		t.Fatal("account history second page failed")
	}
}

func TestDetailsEmptyInvalidAndUnavailableStates(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	h, err := New(store)
	if err != nil {
		t.Fatal(err)
	}

	for _, section := range []string{"friends", "snaps", "calls", "account"} {
		if response := get(h, "/"+section); response.Code != 200 || !strings.Contains(response.Body.String(), "No ") {
			t.Errorf("empty %s view failed: %d", section, response.Code)
		}
	}

	for _, target := range []string{"/friends?filter=unknown", "/calls?filter=unknown", "/snaps?filter=unknown", "/account?offset=-1", "/calls?offset=abc", "/snaps?conversation=-1", "/friends?q=" + strings.Repeat("x", 501)} {
		if response := get(h, target, "HX-Request", "true"); response.Code != 400 || !strings.Contains(response.Body.String(), `id="records"`) {
			t.Errorf("invalid request %s returned %d", target, response.Code)
		}
	}

	store.Close()

	for _, section := range []string{"friends", "snaps", "calls", "account"} {
		response := get(h, "/"+section, "HX-Request", "true")
		if response.Code != 500 || !strings.Contains(response.Body.String(), "Please try again") || strings.Contains(response.Body.String(), "database is closed") {
			t.Errorf("unavailable %s view leaked an error or failed: %d", section, response.Code)
		}
	}
}

func TestThemeNativeRedirectSupportsArchiveSections(t *testing.T) {
	h, _ := testDetailsArchive(t)

	for _, target := range []string{"/friends?filter=blocked", "/snaps?conversation=1", "/calls?offset=40", "/account?offset=40", "https://example.test/account", "//example.test/calls", "/missing"} {
		r := httptest.NewRequest(http.MethodPost, "/theme", strings.NewReader(url.Values{"theme": {"dark"}, "return_to": {target}}.Encode()))

		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()

		h.ServeHTTP(w, r)

		want := target

		if strings.Contains(target, "example.test") || target == "/missing" {
			want = "/"
		}

		if w.Code != http.StatusSeeOther || w.Header().Get("Location") != want {
			t.Errorf("theme return %q: status %d, location %q", target, w.Code, w.Header().Get("Location"))
		}
	}
}
