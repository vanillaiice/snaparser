package frontend

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/vanillaiice/snaparser/v2/db"
	"github.com/vanillaiice/snaparser/v2/parser"
)

func TestFriendNamesOpenMatchingChat(t *testing.T) {
	h, store := testDetailsArchive(t)

	text, decoy := "The correct conversation", "A group with the same sender"

	history := parser.ChatHistory{
		"zz.friend": {{From: "owner", Content: &text}},
		"aaa.group": {{From: "zz.friend", Content: &decoy}},
	}

	for i := range 30 {
		history[fmt.Sprintf("quiet.%02d", i)] = nil
	}

	friends := &parser.Friends{
		Friends:      []parser.Friend{{Username: "zz.friend", DisplayName: "<Friend & Co>"}, {Username: "missing.friend", DisplayName: "zz.friend"}},
		BlockedUsers: []parser.Friend{{Username: "zz.friend", DisplayName: "Blocked Friend"}},
	}

	if err := store.Import(context.Background(), parser.Backup{ChatHistory: &history, Friends: friends}); err != nil {
		t.Fatal(err)
	}

	conversations, err := store.ListConversations(context.Background(), db.Pagination{Limit: 500})
	if err != nil {
		t.Fatal(err)
	}

	var conversationID int64

	for _, conversation := range conversations {
		if conversation.Key == "zz.friend" {
			conversationID = conversation.ID
		}
	}

	target := fmt.Sprintf("/?conversation=%d", conversationID)

	body := get(h, "/friends?q=friend&filter=friends").Body.String()

	for _, name := range []string{"zz.friend", "&lt;Friend &amp; Co&gt;"} {
		if !strings.Contains(body, `href="`+target+`">`+name+`</a>`) {
			t.Errorf("friend name %q must link to its exact conversation", name)
		}
	}

	if !strings.Contains(body, "No chat history") || strings.Contains(body, `>missing.friend</a>`) {
		t.Fatal("friends without a conversation must not link to someone else's chat")
	}

	blocked := get(h, "/friends?filter=blocked").Body.String()

	if !strings.Contains(blocked, `href="`+target+`">Blocked Friend</a>`) {
		t.Fatal("friends in other categories should also link to their chats")
	}

	for _, headers := range [][]string{nil, {"HX-Request", "true"}} {
		response := get(h, target, headers...)

		body := response.Body.String()

		if response.Code != 200 || !strings.Contains(body, text) || strings.Contains(body, decoy) || !strings.Contains(body, `id="query" name="q" type="search" value=""`) {
			t.Fatal("friend links must open the full conversation without carrying friend filters")
		}
	}
}

func TestSnapHistoryLoadsChronologicalBubblesAndOlderFragments(t *testing.T) {
	h, store := testDetailsArchive(t)

	items, err := store.ListSnaps(context.Background(), db.MessageOptions{Query: "alex", Pagination: db.Pagination{Limit: 500}})
	if err != nil || len(items) != 45 {
		t.Fatalf("snap fixture: %d, %v", len(items), err)
	}

	target := fmt.Sprintf("/snaps?conversation=%d", items[0].ConversationID)

	body := get(h, target).Body.String()

	oldest := strings.Index(body, fmt.Sprintf(`<article id="message-%d"`, items[39].ID))

	newest := strings.Index(body, fmt.Sprintf(`<article id="message-%d"`, items[0].ID))

	if oldest < 0 || newest <= oldest || strings.Count(body, `<article id="message-`) != 40 || !strings.Contains(body, "Load older snaps") {
		t.Fatal("snap conversations must open at the latest window in chronological reading order")
	}

	older := get(h, target+"&offset=40", "HX-Request", "true", "HX-Target", "load-older")

	body = older.Body.String()

	if older.Code != 200 || strings.Contains(body, `id="workspace"`) || strings.Contains(body, `class="message-feed`) || strings.Count(body, `<article id="message-`) != 5 || !strings.Contains(body, "No older snaps") {
		t.Fatal("older snaps must return a prependable fragment")
	}

	native := get(h, target+"&offset=40").Body.String()

	if !strings.Contains(native, "Newer snaps") || !strings.Contains(native, "Latest snaps") || !strings.Contains(native, `href="`+target+`"`) {
		t.Fatal("native snap history navigation must stay in the snap conversation")
	}
}

func TestSnapSearchResultOpensSnapContext(t *testing.T) {
	h, store := testDetailsArchive(t)

	items, err := store.ListSnaps(context.Background(), db.MessageOptions{Query: "alex", Pagination: db.Pagination{Limit: 1, Offset: 40}})
	if err != nil || len(items) != 1 {
		t.Fatalf("target snap: %+v, %v", items, err)
	}

	target := items[0]

	body := get(h, "/snaps?q=alex&offset=40&conversations_offset=25").Body.String()

	links := regexp.MustCompile(`class="message-context-link" href="([^"]+)"`)

	var contextURL *url.URL

	for _, match := range links.FindAllStringSubmatch(body, -1) {
		u, err := url.Parse(html.UnescapeString(match[1]))
		if err != nil {
			t.Fatal(err)
		}

		if u.Query().Get("message") == fmt.Sprint(target.ID) {
			contextURL = u
		}
	}

	if contextURL == nil || contextURL.Path != "/snaps" || contextURL.Query().Get("q") != "" || contextURL.Query().Get("offset") != "" || contextURL.Query().Get("conversations_offset") != "25" || contextURL.Fragment != fmt.Sprintf("message-%d", target.ID) {
		t.Fatalf("snap context URL must preserve the snap view and clear search: %v", contextURL)
	}

	for _, headers := range [][]string{nil, {"HX-Request", "true"}} {
		response := get(h, contextURL.RequestURI(), headers...)

		body := response.Body.String()

		if response.Code != 200 || !strings.Contains(body, fmt.Sprintf(`<article id="message-%d" class="message outgoing highlighted"`, target.ID)) || strings.Contains(body, `class="search-summary"`) || !strings.Contains(body, "Snap metadata") {
			t.Fatal("snap context must load surrounding snaps and highlight the selected snap")
		}
	}

	for _, path := range []string{"/snaps?message=-1", "/snaps?message=abc"} {
		if response := get(h, path); response.Code != 400 {
			t.Errorf("invalid snap context returned %d", response.Code)
		}
	}

	if response := get(h, "/snaps?message=999999"); response.Code != 404 || !strings.Contains(response.Body.String(), "Snap not found") {
		t.Fatal("missing snap context must return a recoverable error")
	}

	store.Close()

	failed := get(h, "/snaps?offset=40", "HX-Request", "true", "HX-Target", "load-older")

	if failed.Code != 500 || !strings.Contains(failed.Body.String(), "Retry") || strings.Contains(failed.Body.String(), `id="workspace"`) {
		t.Fatal("failed snap history requests must return a retryable fragment")
	}
}
