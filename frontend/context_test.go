package frontend

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/vanillaiice/snaparser/db"
)

func TestSearchResultOpensFullConversationContext(t *testing.T) {
	h, store := testArchive(t)

	chats, err := store.ListChats(context.Background(), db.MessageOptions{Query: `"Hello memory 10"`})
	if err != nil || len(chats) != 1 {
		t.Fatalf("expected one target message: %v", err)
	}

	target := chats[0]

	search := get(h, "/?q=memory&offset=40&conversations_offset=25").Body.String()

	link := regexp.MustCompile(`class="message-context-link" href="([^"]+)"`)

	var contextURL string

	for _, match := range link.FindAllStringSubmatch(search, -1) {
		u, err := url.Parse(html.UnescapeString(match[1]))
		if err != nil {
			t.Fatal(err)
		}

		if u.Query().Get("message") == fmt.Sprint(target.ID) {
			contextURL = u.String()

			if u.Query().Get("q") != "" || u.Query().Get("offset") != "" || u.Query().Get("conversation") != fmt.Sprint(target.ConversationID) || u.Query().Get("conversations_offset") != "25" || u.Fragment != fmt.Sprintf("message-%d", target.ID) {
				t.Fatalf("incorrect context URL: %s", u)
			}
		}
	}

	if contextURL == "" || !strings.Contains(search, `class="message-context-text"`) {
		t.Fatal("search results need clickable message text and a context link")
	}

	requestURL, err := url.Parse(contextURL)
	if err != nil {
		t.Fatal(err)
	}

	for _, headers := range [][]string{nil, {"HX-Request", "true", "HX-Target", "workspace"}} {
		response := get(h, requestURL.RequestURI(), headers...)

		body := response.Body.String()

		if response.Code != 200 || strings.Contains(body, `class="search-summary"`) || strings.Contains(body, `class="message-context-link"`) || !strings.Contains(body, `id="query" name="q" type="search" value=""`) {
			t.Fatalf("context must clear the active search: status %d", response.Code)
		}

		for _, text := range []string{"Hello memory 00", "Hello memory 10", "Hello memory 30", "Newer messages", "Latest messages"} {
			if !strings.Contains(body, text) {
				t.Errorf("context missing %q", text)
			}
		}

		if first, targetIndex, last := strings.Index(body, "Hello memory 00"), strings.Index(body, "Hello memory 10"), strings.Index(body, "Hello memory 30"); first >= targetIndex || targetIndex >= last {
			t.Fatal("surrounding messages must render in chronological order")
		}

		if strings.Count(body, "highlighted") != 1 || !strings.Contains(body, fmt.Sprintf(`<article id="message-%d" class="message outgoing highlighted" tabindex="-1"`, target.ID)) {
			t.Fatal("exactly the target message must be highlighted")
		}
	}
}

func TestMessageContextValidation(t *testing.T) {
	h, _ := testArchive(t)

	for _, target := range []string{"/?message=-1", "/?message=abc", "/?message=999999999999999999999999"} {
		if response := get(h, target); response.Code != 400 {
			t.Errorf("invalid message request %s returned %d", target, response.Code)
		}
	}

	response := get(h, "/?message=999999", "HX-Request", "true")

	if response.Code != 404 || !strings.Contains(response.Body.String(), "Message not found") {
		t.Fatal("missing context messages need a recoverable error")
	}
}
