package frontend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanillaiice/snaparser/db"
	"github.com/vanillaiice/snaparser/parser"
)

func TestMediaResolutionRenderingAndServing(t *testing.T) {
	root := t.TempDir()

	media := filepath.Join(root, "chat_media")

	if err := os.Mkdir(media, 0700); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"2025-09-15_b~image.jpg", "2025-09-15_b~video.mp4", "2025-09-15_b~audio.m4a", "2025-09-15_b~note.mp4", "2025-09-15_b~unknown.unknown", "2025-09-14_b~wrong-date.jpg"} {
		if err := os.WriteFile(filepath.Join(media, name), []byte("attachment bytes"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	store, err := db.Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	text := "Caption alongside media"

	history := parser.ChatHistory{"friend": {
		{From: "friend", Created: "2025-09-15 19:32:37 UTC", Content: &text, MediaIDS: " b~image | b~video | b~audio | b~unknown | absent "},
		{From: "friend", Created: "2025-09-15 19:32:37 UTC", MediaIDS: "b~wrong-date"},
		{From: "friend", Created: "2025-09-15 19:32:37 UTC", MediaType: "NOTE", MediaIDS: "b~note"},
	}}

	if err := store.Import(context.Background(), parser.Backup{ChatHistory: &history, ExportDirectory: root}); err != nil {
		t.Fatal(err)
	}

	h, err := New(store)
	if err != nil {
		t.Fatal(err)
	}

	body := get(h, "/").Body.String()

	for _, want := range []string{text, `<img src="/media/2025-09-15_b~image.jpg"`, `<video controls`, `<audio controls`, "Download attachment", "Attachment unavailable"} {
		if !strings.Contains(body, want) {
			t.Errorf("media page missing %q", want)
		}
	}

	if strings.Contains(body, root) || strings.Contains(body, "/media/2025-09-14_b~wrong-date.jpg") {
		t.Fatal("local paths and attachments from other dates must not be rendered")
	}

	if !strings.Contains(body, `<audio controls preload="none" src="/media/2025-09-15_b~note.mp4"`) || strings.Count(body, `<video controls`) != 1 {
		t.Fatal("MP4 voice notes must use audio controls while actual videos retain video controls")
	}

	for _, tc := range []struct{ name, contentType string }{
		{"2025-09-15_b~image.jpg", "image/jpeg"},
		{"2025-09-15_b~video.mp4", "video/mp4"},
		{"2025-09-15_b~audio.m4a", "audio/"},
		{"2025-09-15_b~unknown.unknown", "application/octet-stream"},
	} {
		response := get(h, "/media/"+url.PathEscape(tc.name))

		if response.Code != 200 || response.Body.String() != "attachment bytes" || !strings.HasPrefix(response.Header().Get("Content-Type"), tc.contentType) {
			t.Fatalf("attachment %s: status=%d, type=%q, body=%q", tc.name, response.Code, response.Header().Get("Content-Type"), response.Body.String())
		}

		if response.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("attachments must not be cached")
		}

		if strings.HasSuffix(tc.name, ".unknown") && !strings.HasPrefix(response.Header().Get("Content-Disposition"), "attachment;") {
			t.Fatal("unknown types must download")
		}
	}

	ranged := get(h, "/media/2025-09-15_b~video.mp4", "Range", "bytes=0-3")

	if ranged.Code != http.StatusPartialContent || ranged.Body.String() != "atta" {
		t.Fatal("media range requests must support seeking")
	}

	head := httptest.NewRecorder()

	h.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/media/2025-09-15_b~image.jpg", nil))

	if head.Code != 200 || head.Body.Len() != 0 {
		t.Fatal("media HEAD must return headers without a body")
	}

	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("outside secret"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(filepath.Join(root, "secret.txt"), filepath.Join(media, "escape.jpg")); err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"/media/", "/media/missing.jpg", "/media/..%2Fsecret.txt", "/media/%2Fsecret.txt", "/media/escape.jpg"} {
		response := get(h, target)

		if response.Code < 300 || strings.Contains(response.Body.String(), "outside secret") {
			t.Fatalf("unexpected media access: %s, status %d", target, response.Code)
		}
	}

	if err := os.Remove(filepath.Join(media, "2025-09-15_b~image.jpg")); err != nil {
		t.Fatal(err)
	}

	if response := get(h, "/media/2025-09-15_b~image.jpg"); response.Code != 404 {
		t.Fatal("removed attachment must return 404")
	}

	if err := os.RemoveAll(media); err != nil {
		t.Fatal(err)
	}

	if _, err := New(store); err == nil {
		t.Fatal("viewer startup must abort when the saved media directory is missing")
	}
}
