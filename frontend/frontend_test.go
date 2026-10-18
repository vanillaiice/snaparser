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
	"time"

	"github.com/vanillaiice/snaparser/v2/db"
	"github.com/vanillaiice/snaparser/v2/parser"
)

// Exercise the real SQLite/FTS path, not a mock returning pre-rendered results.
func testArchive(t *testing.T) (http.Handler, *db.DB) {
	t.Helper()

	store, err := db.Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { store.Close() })

	history := parser.ChatHistory{}

	for i := range 57 {
		content := fmt.Sprintf("Hello memory %02d", i)

		timestamp := time.Date(2024, 6, 12, 12, i, 0, 0, time.UTC).UnixMicro()

		if i == 56 {
			// Snapchat sometimes supplies milliseconds in the microseconds field.
			timestamp /= 1000
		}

		history["alex.river"] = append(history["alex.river"], parser.Chat{
			From: "alex.river", Content: &content, MediaType: "TEXT", IsSender: i%2 == 0,
			CreatedMicrosecond: timestamp, IsSaved: i == 56,
		})
	}

	hostile := `<script>alert("hello")</script><img src=x onerror="bad()">`

	history["other.person"] = []parser.Chat{{From: "other.person", Content: &hostile, Created: "2024-06-11 18:20:00 UTC"}, {From: "other.person", MediaType: "PHOTO"}}

	for i := range 27 {
		history[fmt.Sprintf("quiet.%02d", i)] = nil
	}

	account := &parser.Account{BasicInformation: parser.BasicInformation{Name: "Archive Owner", Username: "owner"}}

	if err := store.Import(context.Background(), parser.Backup{Account: account, ChatHistory: &history}); err != nil {
		t.Fatal(err)
	}

	h, err := New(store)
	if err != nil {
		t.Fatal(err)
	}

	return h, store
}

func get(h http.Handler, target string, headers ...string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, target, nil)

	for i := 0; i < len(headers); i += 2 {
		r.Header.Set(headers[i], headers[i+1])
	}

	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	return w
}

func TestArchivePageAndFragments(t *testing.T) {
	h, _ := testArchive(t)

	full := get(h, "/")

	if full.Code != http.StatusOK {
		t.Fatalf("status %d: %s", full.Code, full.Body.String())
	}

	for _, want := range []string{"<!doctype html>", "All conversations", "Hello memory 56", "2024-06-12", "12:56:00", "@owner", "conversations_offset=25", "offset=40", "Load older messages", "Latest messages"} {
		if !strings.Contains(full.Body.String(), want) {
			t.Errorf("page missing %q", want)
		}
	}

	partial := get(h, "/?q=memory", "HX-Request", "true")

	if partial.Code != http.StatusOK || strings.Contains(partial.Body.String(), "<!doctype") || !strings.Contains(partial.Body.String(), `id="workspace"`) || !strings.Contains(partial.Body.String(), `id="archive"`) {
		t.Fatalf("invalid fragment response: %d %s", partial.Code, partial.Body.String())
	}

	restored := get(h, "/?q=memory", "HX-Request", "true", "HX-History-Restore-Request", "true")

	if !strings.Contains(restored.Body.String(), "<!doctype html>") {
		t.Fatal("history restoration needs the full document")
	}

	if full.Header().Get("Cache-Control") != "no-store" || !strings.Contains(full.Header().Get("Vary"), "HX-Request") {
		t.Fatal("archive responses must disable caching and vary by HTMX headers")
	}
}

func TestChronologicalHistoryAndOlderFragments(t *testing.T) {
	h, _ := testArchive(t)

	target := "/?conversation=1"

	body := get(h, target).Body.String()

	if first, last := strings.Index(body, "Hello memory 17"), strings.Index(body, "Hello memory 56"); first < 0 || last < first {
		t.Fatal("the latest window must render oldest-to-newest")
	}

	older := get(h, target+"&offset=40", "HX-Request", "true", "HX-Target", "load-older")

	body = older.Body.String()

	if older.Code != 200 || strings.Contains(body, `id="workspace"`) || strings.Contains(body, `class="message-feed`) || strings.Contains(body, "Hello memory 17") {
		t.Fatal("older history must be a prependable fragment without replacing the current view")
	}

	if strings.Count(body, `<article id="message-`) != 17 || !strings.Contains(body, "No older messages") {
		t.Fatal("the final history window should include exactly the remaining messages")
	}

	if first, last := strings.Index(body, "Hello memory 00"), strings.Index(body, "Hello memory 16"); first < 0 || last < first {
		t.Fatal("older fragments must also render oldest-to-newest")
	}

	native := get(h, target+"&offset=40")

	if native.Code != 200 || !strings.Contains(native.Body.String(), "Newer messages") || !strings.Contains(native.Body.String(), "Latest messages") {
		t.Fatal("native older-page navigation must offer a route back to newer history")
	}

	search := get(h, "/?q=memory").Body.String()

	if newest, older := strings.Index(search, "Hello memory 56"), strings.Index(search, "Hello memory 17"); newest < 0 || older < newest {
		t.Fatal("search should retain database relevance order rather than reverse it")
	}
}

func TestSearchScopePaginationAndEscaping(t *testing.T) {
	h, store := testArchive(t)

	conversations, err := store.ListConversations(context.Background(), db.Pagination{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}

	var otherID int64

	for _, c := range conversations {
		if c.Key == "other.person" {
			otherID = c.ID
		}
	}

	search := get(h, "/?q=mem")

	if search.Code != 200 || !strings.Contains(search.Body.String(), "Hello memory 56") || strings.Contains(search.Body.String(), "&lt;script&gt;") {
		t.Fatal("prefix search should match memory messages only")
	}

	pageTwo := get(h, "/?q=mem&offset=40")

	if pageTwo.Code != 200 || !strings.Contains(pageTwo.Body.String(), "41–57") || strings.Contains(pageTwo.Body.String(), "Hello memory 56") || !strings.Contains(pageTwo.Body.String(), "q=mem") {
		t.Fatal("second page should preserve the search and show the remaining 17 matches")
	}

	scoped := get(h, fmt.Sprintf("/?q=memory&conversation=%d", otherID))

	if scoped.Code != 200 || !strings.Contains(scoped.Body.String(), "No matching messages") || strings.Contains(scoped.Body.String(), "Hello memory") {
		t.Fatal("search must stay within the selected conversation")
	}

	unsafe := get(h, fmt.Sprintf("/?conversation=%d", otherID))

	if unsafe.Code != 200 || strings.Contains(unsafe.Body.String(), `<script>alert`) || !strings.Contains(unsafe.Body.String(), "&lt;script&gt;") || !strings.Contains(unsafe.Body.String(), "Attachment unavailable") {
		t.Fatal("message content should be escaped and missing media explained")
	}

	for _, query := range []string{`"`, `*`, `hello OR (`, `"hello"`, `-`, `<script>`, `sender:hello`, `hello NEAR world`} {
		response := get(h, "/?q="+url.QueryEscape(query))

		if response.Code != 200 {
			t.Errorf("literal search %q returned %d", query, response.Code)
		}
	}
}

func TestInvalidRequestsAndAssets(t *testing.T) {
	h, _ := testArchive(t)

	if response := get(h, "/?q="+url.QueryEscape(strings.Repeat("猫", 250))); response.Code != 200 {
		t.Fatal("the search limit should count characters, not UTF-8 bytes")
	}

	for _, target := range []string{"/?offset=-1", "/?conversation=abc", "/?conversations_offset=9999999999999999999", "/?q=" + strings.Repeat("x", 501)} {
		response := get(h, target, "HX-Request", "true")

		if response.Code != 400 || !strings.Contains(response.Body.String(), `id="archive"`) {
			t.Errorf("%s: expected renderable 400 response, got %d", target, response.Code)
		}
	}

	if response := get(h, "/?conversation=999999"); response.Code != 404 || !strings.Contains(response.Body.String(), "The archive may have been replaced.") {
		t.Fatal("missing conversations should have a recoverable 404 page")
	}

	for _, asset := range []string{"app.css", "htmx-2.0.11.min.js", "favicon.svg"} {
		response := get(h, "/assets/"+asset)

		if response.Code != 200 || response.Body.Len() == 0 || response.Header().Get("Content-Security-Policy") == "" {
			t.Errorf("embedded asset %s unavailable", asset)
		}
	}

	if response := get(h, "/missing"); response.Code != 404 {
		t.Fatal("unknown paths should return 404")
	}
}

func TestEmptyAndUnavailableDatabase(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { store.Close() })

	h, err := New(store)
	if err != nil {
		t.Fatal(err)
	}

	response := get(h, "/")

	if response.Code != 200 || !strings.Contains(response.Body.String(), "No messages") {
		t.Fatal("an empty database should show the welcome state")
	}

	store.Close()

	response = get(h, "/", "HX-Request", "true")

	if response.Code != 500 || !strings.Contains(response.Body.String(), "Please try again in a moment.") || strings.Contains(response.Body.String(), "database is closed") {
		t.Fatal("database errors should show a safe, renderable error state")
	}

	response = get(h, "/?offset=40", "HX-Request", "true", "HX-Target", "load-older")

	if response.Code != 500 || !strings.Contains(response.Body.String(), `id="load-older"`) || !strings.Contains(response.Body.String(), "Retry") || strings.Contains(response.Body.String(), `id="workspace"`) {
		t.Fatal("failed older-history loads must be retryable without replacing existing messages")
	}
}

func TestThemePreference(t *testing.T) {
	h, _ := testArchive(t)

	for _, theme := range []string{"light", "dark", "system"} {
		t.Run(theme, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/theme", strings.NewReader(url.Values{"theme": {theme}}.Encode()))

			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("HX-Request", "true")

			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
				t.Fatalf("theme selection must preserve the current DOM: %d %s", w.Code, w.Body.String())
			}

			cookies := w.Result().Cookies()

			if len(cookies) != 1 || cookies[0].Name != themeCookieName || cookies[0].Value != theme || cookies[0].Path != "/" || cookies[0].MaxAge != 365*24*60*60 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
				t.Fatalf("unexpected theme preference cookie: %+v", cookies)
			}

			r = httptest.NewRequest(http.MethodGet, "/?q=memory", nil)

			r.AddCookie(cookies[0])

			w = httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if w.Code != 200 || !strings.Contains(w.Body.String(), `value="`+theme+`" selected`) {
				t.Fatal("the saved theme must be selected on the next page load")
			}
		})
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)

	r.AddCookie(&http.Cookie{Name: themeCookieName, Value: "unknown"})

	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if !strings.Contains(w.Body.String(), `value="system" selected`) {
		t.Fatal("unrecognized theme cookies should fall back to System")
	}
}

func TestThemeNativeFormAndValidation(t *testing.T) {
	h, _ := testArchive(t)

	for _, target := range []string{"/?q=hello&conversation=1&offset=40", "https://example.com/", "//example.com/", "/other"} {
		values := url.Values{"theme": {"dark"}, "return_to": {target}}

		r := httptest.NewRequest(http.MethodPost, "/theme", strings.NewReader(values.Encode()))

		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()

		h.ServeHTTP(w, r)

		expected := "/"

		if strings.HasPrefix(target, "/?") {
			expected = target
		}

		if w.Code != http.StatusSeeOther || w.Header().Get("Location") != expected {
			t.Fatalf("native theme forms must return to a local archive URL: %d %q", w.Code, w.Header().Get("Location"))
		}
	}

	for _, theme := range []string{"", "invalid"} {
		r := httptest.NewRequest(http.MethodPost, "/theme", strings.NewReader(url.Values{"theme": {theme}}.Encode()))

		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()

		h.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest || len(w.Result().Cookies()) != 0 {
			t.Fatal("invalid theme selections must not change the saved preference")
		}
	}
}
