package frontend

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/vanillaiice/snaparser/db"
	"github.com/vanillaiice/snaparser/parser"
)

func managedExport(t *testing.T, name, text string) string {
	t.Helper()

	root := t.TempDir()

	for _, directory := range []string{"json", "chat_media"} {
		if err := os.Mkdir(filepath.Join(root, directory), 0700); err != nil {
			t.Fatal(err)
		}
	}

	for _, filename := range []string{"account.json", "account_history.json", "chat_history.json", "friends.json", "snap_history.json", "snap_pro.json", "talk_history.json", "user_profile.json"} {
		if err := os.WriteFile(filepath.Join(root, "json", filename), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
	}

	writeExportJSON(t, root, "account.json", &parser.Account{BasicInformation: parser.BasicInformation{Username: name}})

	writeExportJSON(t, root, "chat_history.json", parser.ChatHistory{name: {{From: name, Content: &text, Created: "2025-09-15 12:00:00 UTC", MediaIDS: "b~picture", MediaType: "PHOTO"}}})

	if err := os.WriteFile(filepath.Join(root, "chat_media", "2025-09-15_b~picture.jpg"), []byte(name+" picture"), 0600); err != nil {
		t.Fatal(err)
	}

	return root
}

func writeExportJSON(t *testing.T, root, filename string, value any) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "json", filename), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func managedArchive(t *testing.T) *Archive {
	t.Helper()

	a, err := OpenArchive(t.Context(), filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { a.Close() })

	return a
}

func postArchive(h http.Handler, target string, values url.Values, headers ...string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))

	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	for i := 0; i < len(headers); i += 2 {
		r.Header.Set(headers[i], headers[i+1])
	}

	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	return w
}

func waitImport(t *testing.T, a *Archive) {
	t.Helper()

	a.mu.RLock()

	job := a.job

	a.mu.RUnlock()

	if job == nil {
		t.Fatal("no background import was started")
	}

	select {
	case <-job.done:
	case <-time.After(10 * time.Second):
		t.Fatal("background import did not finish")
	}
}

func selectExport(t *testing.T, a *Archive, root string) {
	t.Helper()

	response := postArchive(a, "/archive/import", url.Values{"directory": {root}})

	if response.Code != http.StatusSeeOther || !strings.HasPrefix(response.Header().Get("Location"), "/setup?job=") {
		t.Fatalf("selection did not start an import: %d %s", response.Code, response.Body.String())
	}

	waitImport(t, a)

	if p := a.snapshot(); p.Error != "" {
		t.Fatal(p.Error)
	}
}

func TestSetupStartsWithoutCreatingDatabase(t *testing.T) {
	a := managedArchive(t)

	response := get(a, "/")

	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/setup" {
		t.Fatal("an unconfigured archive must open setup")
	}

	setup := get(a, "/setup")

	for _, want := range []string{"Open your Snapchat export", `name="directory"`, "Browse folders", `action="/archive/import"`, `class="folder-browser"`, `name="browse"`, "Use this folder"} {
		if !strings.Contains(setup.Body.String(), want) {
			t.Errorf("setup missing %q", want)
		}
	}

	if setup.Code != 200 || setup.Header().Get("Cache-Control") != "no-store" || strings.Contains(setup.Body.String(), `action="/archive/rebuild"`) {
		t.Fatal("initial setup must render without rebuild controls")
	}

	if _, err := os.Stat(a.path); !os.IsNotExist(err) {
		t.Fatal("opening setup created a database before any export was selected")
	}

	if asset := get(a, "/assets/app.css"); asset.Code != 200 {
		t.Fatal("setup assets are unavailable")
	}

	if rebuild := postArchive(a, "/archive/rebuild", nil); rebuild.Code != 400 {
		t.Fatal("rebuilding without a saved export should fail")
	}
}

func TestExistingEmptyDatabaseAndInvalidFirstSelectionStayInSetup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.db")

	store, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	store.Close()

	a, err := OpenArchive(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}

	defer a.Close()

	if response := get(a, "/"); response.Code != http.StatusSeeOther {
		t.Fatal("a database without an imported archive should open setup")
	}

	postArchive(a, "/archive/import", url.Values{"directory": {t.TempDir()}})

	waitImport(t, a)

	if state := a.snapshot(); state.HasArchive || state.Error == "" {
		t.Fatal("an invalid first selection must stay in setup with an error")
	}

	selectExport(t, a, managedExport(t, "owner", "First valid archive"))

	if response := get(a, "/"); response.Code != 200 || !strings.Contains(response.Body.String(), "First valid archive") {
		t.Fatal("a valid retry must initialize the empty database and open the app")
	}
}

func TestSelectionImportsOpensAndPersistsArchive(t *testing.T) {
	a := managedArchive(t)

	root := managedExport(t, "first.owner", "First selected archive")

	selectExport(t, a, root)

	body := get(a, "/").Body.String()

	if !strings.Contains(body, "First selected archive") || !strings.Contains(body, "@first.owner") || !strings.Contains(body, `href="/setup"`) {
		t.Fatal("selection must open the full app and retain archive controls")
	}

	media := get(a, "/media/2025-09-15_b~picture.jpg")

	if media.Code != 200 || media.Body.String() != "first.owner picture" {
		t.Fatal("selected export attachments were not activated")
	}

	status := get(a, "/setup/status", "HX-Request", "true")

	if status.Header().Get("HX-Redirect") != "/" {
		t.Fatal("completed imports must navigate to the archive")
	}

	setup := get(a, "/setup").Body.String()

	if !strings.Contains(setup, "Rebuild archive") || !strings.Contains(setup, root) || !strings.Contains(setup, "Import selected folder") || !strings.Contains(setup, folderURL(filepath.Join(root, "json"))) {
		t.Fatal("archive management must expose the saved source and both actions")
	}

	if err := a.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenArchive(t.Context(), a.path)
	if err != nil {
		t.Fatal(err)
	}

	defer reopened.Close()

	if response := get(reopened, "/"); response.Code != 200 || !strings.Contains(response.Body.String(), "First selected archive") {
		t.Fatal("a browser import must survive restarting without data flags")
	}
}

func TestReturningFromSetupRestoresAppHeader(t *testing.T) {
	a := managedArchive(t)

	selectExport(t, a, managedExport(t, "archive.owner", "Archive navigation"))

	setup := get(a, "/setup").Body.String()

	links := regexp.MustCompile(`<a\b[^>]*href="/"[^>]*>`).FindAllString(setup, -1)

	if len(links) != 2 {
		t.Fatal("setup should offer the logo and back link to return to the app")
	}

	for _, link := range links {
		if !strings.Contains(link, `hx-boost="false"`) {
			t.Fatalf("returning from setup must reload the header outside #workspace: %s", link)
		}
	}

	for range 2 {
		body := get(a, "/").Body.String()

		if !strings.Contains(body, `href="/setup" hx-boost="false">Archive</a>`) || !strings.Contains(body, "@archive.owner") {
			t.Fatal("returning to the app must restore archive controls and the account name")
		}

		if response := get(a, "/setup"); response.Code != http.StatusOK {
			t.Fatal("archive management must remain available after returning to the app")
		}
	}
}

func TestSwitchAndRebuildRefreshDataAndAttachments(t *testing.T) {
	a := managedArchive(t)

	first := managedExport(t, "first.owner", "Original archive content")

	second := managedExport(t, "second.owner", "Replacement archive content")

	selectExport(t, a, first)

	oldAttachment := regexp.MustCompile(`<img src="([^"]+)"`).FindStringSubmatch(get(a, "/").Body.String())
	if len(oldAttachment) != 2 {
		t.Fatal("the first archive should contain an attachment")
	}

	selectExport(t, a, second)

	body := get(a, "/").Body.String()

	if !strings.Contains(body, "Replacement archive content") || strings.Contains(body, "Original archive content") || strings.Contains(body, "@first.owner") {
		t.Fatal("switching folders must replace the database and the active viewer")
	}

	if media := get(a, "/media/2025-09-15_b~picture.jpg"); media.Body.String() != "second.owner picture" {
		t.Fatal("switching folders retained the previous media directory")
	}

	if stale := get(a, oldAttachment[1]); stale.Code != http.StatusNotFound {
		t.Fatal("an old page must not load another export's same-named attachment")
	}

	text := "Rebuilt archive content"

	writeExportJSON(t, second, "chat_history.json", parser.ChatHistory{"second.owner": {{Content: &text, Created: "2025-09-15 12:00:00 UTC", MediaIDS: "b~new"}}})

	if err := os.WriteFile(filepath.Join(second, "chat_media", "2025-09-15_b~new.png"), []byte("new media"), 0600); err != nil {
		t.Fatal(err)
	}

	if response := postArchive(a, "/archive/rebuild", nil); response.Code != http.StatusSeeOther {
		t.Fatal("rebuild did not start")
	}

	waitImport(t, a)

	body = get(a, "/").Body.String()

	if !strings.Contains(body, text) || strings.Contains(body, "Replacement archive content") || !strings.Contains(body, "/media/2025-09-15_b~new.png") {
		t.Fatal("rebuilding must refresh both parsed data and the media index")
	}

	if media := get(a, "/media/2025-09-15_b~new.png"); media.Code != 200 || media.Body.String() != "new media" {
		t.Fatal("newly indexed attachments are unavailable after rebuilding")
	}

	if source, err := a.store.GetExportDirectory(context.Background()); err != nil || source != second {
		t.Fatalf("rebuild changed the selected source: %q, %v", source, err)
	}
}

func TestFailedSelectionPreservesArchiveAndMedia(t *testing.T) {
	a := managedArchive(t)

	first := managedExport(t, "first.owner", "Preserved archive content")

	selectExport(t, a, first)

	for _, directory := range []string{t.TempDir(), filepath.Join(t.TempDir(), "missing")} {
		postArchive(a, "/archive/import", url.Values{"directory": {directory}})

		waitImport(t, a)

		if p := a.snapshot(); p.Error == "" || p.CurrentDirectory != first {
			t.Fatal("failed import must report an error and preserve the saved source")
		}

		if body := get(a, "/").Body.String(); !strings.Contains(body, "Preserved archive content") {
			t.Fatal("failed selection replaced the working archive")
		}

		if media := get(a, "/media/2025-09-15_b~picture.jpg"); media.Body.String() != "first.owner picture" {
			t.Fatal("failed selection replaced the working media snapshot")
		}

		if response := get(a, "/setup/status", "HX-Request", "true"); response.Header().Get("HX-Redirect") != "/setup" {
			t.Fatal("failed background imports must return to selection with an error")
		}
	}

	second := managedExport(t, "second.owner", "Must be rolled back")

	connection, err := sql.Open("sqlite", a.path)
	if err != nil {
		t.Fatal(err)
	}

	defer connection.Close()

	if _, err := connection.Exec(`create trigger reject_import before insert on archive_source begin select raise(abort, 'test failure'); end`); err != nil {
		t.Fatal(err)
	}

	postArchive(a, "/archive/import", url.Values{"directory": {second}})

	waitImport(t, a)

	if a.snapshot().Error == "" || !strings.Contains(get(a, "/").Body.String(), "Preserved archive content") || get(a, "/media/2025-09-15_b~picture.jpg").Body.String() != "first.owner picture" {
		t.Fatal("an insertion failure must preserve both the database and media snapshot")
	}

	if err := os.WriteFile(filepath.Join(first, "json", "chat_history.json"), []byte(`{"broken":`), 0600); err != nil {
		t.Fatal(err)
	}

	postArchive(a, "/archive/rebuild", nil)

	waitImport(t, a)

	if a.snapshot().Error == "" || !strings.Contains(get(a, "/").Body.String(), "Preserved archive content") {
		t.Fatal("a failed rebuild must preserve the last working archive")
	}
}

func TestFolderBrowserAndCrossOriginProtection(t *testing.T) {
	a := managedArchive(t)

	root := t.TempDir()

	for _, name := range []string{"folder & one", "json", "chat_media"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "file.txt"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	response := get(a, folderURL(root))

	body := response.Body.String()

	if response.Code != 200 || !strings.Contains(body, "folder &amp; one") || !strings.Contains(body, "Parent folder") || !strings.Contains(body, "Use this folder") || strings.Contains(body, "file.txt") {
		t.Fatal("folder browser must list escaped directories and selection controls")
	}

	if response := get(a, folderURL(filepath.Join(root, "missing"))); response.Code != 200 || !strings.Contains(response.Body.String(), "Unable to read this folder") {
		t.Fatal("unreadable browse paths must have a recoverable error")
	}

	for _, endpoint := range []string{"/archive/import", "/archive/rebuild"} {
		for _, headers := range [][]string{{"Origin", "https://other.example"}, {"Sec-Fetch-Site", "cross-site"}} {
			if response := postArchive(a, endpoint, url.Values{"directory": {root}}, headers...); response.Code != http.StatusForbidden {
				t.Errorf("cross-origin %s returned %d", endpoint, response.Code)
			}
		}
	}

	if response := postArchive(a, "/archive/import", url.Values{"directory": {""}}); response.Code != 400 {
		t.Fatal("empty selections must fail")
	}

	if response := postArchive(a, "/archive/import", url.Values{"directory": {strings.Repeat("x", 20<<10)}}); response.Code != 400 {
		t.Fatal("oversized forms must fail")
	}

	if a.snapshot().JobID != 0 {
		t.Fatal("rejected requests must not start a background import")
	}
}

func TestImportProgressStaysResponsiveAndRequestsSeeCompleteSnapshot(t *testing.T) {
	a := managedArchive(t)

	selectExport(t, a, managedExport(t, "first.owner", "Old snapshot"))

	second := managedExport(t, "second.owner", "New snapshot")

	lock, err := sql.Open("sqlite", a.path)
	if err != nil {
		t.Fatal(err)
	}

	defer lock.Close()

	lock.SetMaxOpenConns(1)

	if _, err := lock.Exec("begin immediate"); err != nil {
		t.Fatal(err)
	}

	defer lock.Exec("rollback")

	requestContext, cancelRequest := context.WithCancel(context.Background())

	request := httptest.NewRequest(http.MethodPost, "/archive/import", strings.NewReader(url.Values{"directory": {second}}.Encode())).WithContext(requestContext)

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	started := httptest.NewRecorder()

	a.ServeHTTP(started, request)

	// Returning a response/disconnecting must not cancel the background import.
	cancelRequest()

	deadline := time.After(5 * time.Second)

	for a.snapshot().Stage != "Saving the archive" {
		select {
		case <-deadline:
			t.Fatal("import did not reach the database write stage")
		case <-time.After(time.Millisecond):
		}
	}

	status := get(a, "/setup/status", "HX-Request", "true")

	if status.Code != 200 || !strings.Contains(status.Body.String(), "Saving the archive") || !strings.Contains(status.Body.String(), `id="setup-status"`) {
		t.Fatal("progress polling must remain responsive during an import")
	}

	if setup := get(a, "/setup"); setup.Code != http.StatusSeeOther || setup.Header().Get("Location") != started.Header().Get("Location") {
		t.Fatal("opening setup during an import must join its progress page")
	}

	if progress := get(a, started.Header().Get("Location")); progress.Code != 200 || !strings.Contains(progress.Body.String(), `<noscript><meta http-equiv="refresh"`) {
		t.Fatal("native progress pages need a refresh fallback")
	}

	if media := get(a, "/media/2025-09-15_b~picture.jpg"); media.Code != 200 || media.Body.String() != "first.owner picture" {
		t.Fatal("media playback must remain available during the database write")
	}

	again := postArchive(a, "/archive/rebuild", nil)

	if again.Code != http.StatusSeeOther || again.Header().Get("Location") != started.Header().Get("Location") {
		t.Fatal("overlapping submissions must reuse the active job")
	}

	response := make(chan *httptest.ResponseRecorder, 1)

	go func() { response <- get(a, "/") }()

	select {
	case <-response:
		t.Fatal("archive requests must wait while its database and media snapshot change")
	case <-time.After(20 * time.Millisecond):
	}

	if _, err := lock.Exec("commit"); err != nil {
		t.Fatal(err)
	}

	waitImport(t, a)

	select {
	case result := <-response:
		if result.Code != 200 || !strings.Contains(result.Body.String(), "New snapshot") || strings.Contains(result.Body.String(), "Old snapshot") {
			t.Fatal("an in-flight request observed an incomplete replacement")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("archive requests remained blocked after replacement")
	}
}

func TestMissingSavedMediaCanBeReselected(t *testing.T) {
	a := managedArchive(t)

	root := managedExport(t, "owner", "Recovered archive")

	selectExport(t, a, root)

	a.Close()

	moved := filepath.Join(t.TempDir(), "moved-export")

	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenArchive(t.Context(), a.path)
	if err != nil {
		t.Fatal(err)
	}

	defer reopened.Close()

	if response := get(reopened, "/setup"); response.Code != 200 || !strings.Contains(response.Body.String(), "saved export folder is unavailable") {
		t.Fatal("a moved source must open recovery setup instead of preventing startup")
	}

	selectExport(t, reopened, moved)

	if response := get(reopened, "/"); response.Code != 200 || !strings.Contains(response.Body.String(), "Recovered archive") {
		t.Fatal("reselecting a moved export did not recover the application")
	}
}
