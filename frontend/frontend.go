// Package frontend serves Snaparser's embedded, server-rendered archive browser.
package frontend

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vanillaiice/snaparser/v2/db"
	"github.com/vanillaiice/snaparser/v2/parser"
)

//go:embed templates/*.html assets/*
var files embed.FS

const (
	conversationPageSize = 25
	messagePageSize      = 40
	themeCookieName      = "snaparser_theme"
)

// Store is satisfied by *db.DB. The caller owns the database's lifecycle.
type Store interface {
	ListConversations(context.Context, db.Pagination) ([]db.Conversation, error)
	ListSnapConversations(context.Context, db.Pagination) ([]db.Conversation, error)
	GetConversation(context.Context, int64) (db.Conversation, error)
	GetChatPosition(context.Context, int64) (db.ChatPosition, error)
	GetSnapPosition(context.Context, int64) (db.MessagePosition, error)
	ListChats(context.Context, db.MessageOptions) ([]db.Chat, error)
	ListSnaps(context.Context, db.MessageOptions) ([]db.Snap, error)
	ListFriends(context.Context, db.FriendOptions) ([]db.Friend, error)
	ListCalls(context.Context, db.CallOptions) ([]db.Call, error)
	ListAccountChanges(context.Context, db.Pagination) ([]db.AccountChange, error)
	GetAccount(context.Context) (*parser.Account, error)
	GetSnapProfile(context.Context) (*parser.SnapProfile, error)
	GetUserProfile(context.Context) (*parser.UserProfile, error)
	GetExportDirectory(context.Context) (string, error)
}

// New returns a handler mounted at /. Assets, templates, and HTMX are embedded;
// attachments are read from the export root saved with the archive.
func New(store Store) (http.Handler, error) {
	if store == nil {
		return nil, errors.New("frontend: a database store is required")
	}

	exportDirectory, err := store.GetExportDirectory(context.Background())
	if err != nil {
		return nil, fmt.Errorf("read archive source: %w", err)
	}

	var mediaDirectory string

	var mediaIndex parser.MediaIndex

	if exportDirectory != "" {
		mediaDirectory = filepath.Join(exportDirectory, "chat_media")

		mediaIndex, err = parser.NewMediaIndex(mediaDirectory)
		if err != nil {
			return nil, fmt.Errorf("load archive media (keep the export in its original location): %w", err)
		}
	}

	return newWithMedia(store, mediaDirectory, mediaIndex, false)
}

func parseTemplates() (*template.Template, error) {
	return template.New("archive").Funcs(template.FuncMap{
		"number": number,
		"date":   date,
		"label":  label,
	}).ParseFS(files, "templates/*.html")
}

// Prepare a viewer before importing, so committing data and replacing its media
// snapshot can happen together under the managed archive's request lock.
func newWithMedia(store Store, mediaDirectory string, mediaIndex parser.MediaIndex, managed bool) (http.Handler, error) {
	templates, err := parseTemplates()
	if err != nil {
		return nil, fmt.Errorf("parse frontend templates: %w", err)
	}

	assets, err := fs.Sub(files, "assets")
	if err != nil {
		return nil, err
	}

	h := &handler{
		store: store, templates: templates, managed: managed,
		mediaDirectory: mediaDirectory, mediaIndex: mediaIndex,
	}

	if managed {
		h.mediaVersion = rand.Text()
	}

	mux := http.NewServeMux()

	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	mux.HandleFunc("GET /{$}", h.archive)
	mux.HandleFunc("GET /friends", h.details)
	mux.HandleFunc("GET /snaps", h.archive)
	mux.HandleFunc("GET /calls", h.details)
	mux.HandleFunc("GET /account", h.details)
	mux.HandleFunc("POST /theme", h.setTheme)
	mux.HandleFunc("GET /media/{filename}", h.serveMedia)

	return secure(mux), nil
}

func secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; media-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")

		next.ServeHTTP(w, r)
	})
}

type handler struct {
	managed        bool
	store          Store
	templates      *template.Template
	mediaDirectory string
	mediaIndex     parser.MediaIndex
	mediaVersion   string
}

type page struct {
	Setup              *setupPage
	CanManage          bool
	Section            string
	Title              string
	Filter             string
	Friends            []db.Friend
	Calls              []db.Call
	Changes            []db.AccountChange
	Details            []detailGroup
	Account            *parser.Account
	Profile            *parser.UserProfile
	MoreRecords        bool
	Theme              string
	Query              string
	ConversationID     int64
	ConversationOffset int
	Offset             int
	MessageID          int64
	Conversations      []db.Conversation
	MoreConversations  bool
	Messages           []message
	MoreMessages       bool
	Selected           *db.Conversation
	AccountName        string
	AccountUsername    string
	Error              string
	ErrorTitle         string
}

type message struct {
	ID               int64
	From             string
	Text             string
	MediaType        string
	Saved            bool
	Outgoing         bool
	Created          string
	DateTime         string
	Day              string
	NewDay           bool
	ConversationName string
	ConversationURL  string
	Attachments      []attachment
	ContextURL       string
	Highlighted      bool
}

func (p page) URL(conversationID int64, offset, conversationOffset int) string {
	v := url.Values{}

	if p.Query != "" {
		v.Set("q", p.Query)
	}

	if conversationID > 0 {
		v.Set("conversation", strconv.FormatInt(conversationID, 10))
	}

	if offset > 0 {
		v.Set("offset", strconv.Itoa(offset))
	}

	if conversationOffset > 0 {
		v.Set("conversations_offset", strconv.Itoa(conversationOffset))
	}

	if len(v) == 0 {
		return p.MessagesPath()
	}

	return p.MessagesPath() + "?" + v.Encode()
}

func (p page) IsHistory() bool {
	return p.Section == "" || p.Section == "snaps"
}

func (p page) MessagesPath() string {
	if p.Section == "snaps" {
		return "/snaps"
	}

	return "/"
}

func (p page) HistoryLabel() string {
	if p.Section == "snaps" {
		return "snaps"
	}

	return "messages"
}

func (p page) PreviousMessages() string {
	return p.URL(p.ConversationID, max(0, p.Offset-messagePageSize), p.ConversationOffset)
}

func (p page) NextMessages() string {
	return p.URL(p.ConversationID, p.Offset+messagePageSize, p.ConversationOffset)
}

func (p page) PreviousConversations() string {
	return p.URL(p.ConversationID, p.Offset, max(0, p.ConversationOffset-conversationPageSize))
}

func (p page) NextConversations() string {
	return p.URL(p.ConversationID, p.Offset, p.ConversationOffset+conversationPageSize)
}

func (p page) ClearSearchURL() string {
	p.Query = ""

	return p.URL(p.ConversationID, 0, p.ConversationOffset)
}

func (p page) MessageContextURL(conversationID, messageID int64) string {
	v := url.Values{
		"conversation": {strconv.FormatInt(conversationID, 10)},
		"message":      {strconv.FormatInt(messageID, 10)},
	}

	if p.ConversationOffset > 0 {
		v.Set("conversations_offset", strconv.Itoa(p.ConversationOffset))
	}

	u := url.URL{Path: p.MessagesPath(), RawQuery: v.Encode(), Fragment: fmt.Sprintf("message-%d", messageID)}

	return u.String()
}

func (p page) Start() int { return p.Offset + 1 }

func (p page) End() int { return p.Offset + len(p.Messages) }

func (h *handler) archive(w http.ResponseWriter, r *http.Request) {
	p := page{Theme: requestTheme(r), Query: strings.TrimSpace(r.URL.Query().Get("q"))}

	if r.URL.Path == "/snaps" {
		p.Section = "snaps"
	}

	status := http.StatusOK

	var err error

	p.ConversationID, err = nonnegative(r.URL.Query().Get("conversation"))
	if err == nil {
		p.MessageID, err = nonnegative(r.URL.Query().Get("message"))
	}

	if err == nil {
		var offset int64

		offset, err = nonnegative(r.URL.Query().Get("offset"))

		p.Offset = int(offset)
	}

	if err == nil {
		var offset int64

		offset, err = nonnegative(r.URL.Query().Get("conversations_offset"))

		p.ConversationOffset = int(offset)
	}

	if err != nil || utf8.RuneCountInString(p.Query) > 500 {
		status = http.StatusBadRequest

		p.ErrorTitle, p.Error = "Invalid request", "Use a search of 500 characters or fewer and valid conversation and page numbers."
	} else if err := h.load(r.Context(), &p); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound

			p.ErrorTitle, p.Error = "Conversation not found", "The archive may have been replaced. Select another conversation."

			if p.MessageID != 0 {
				p.ErrorTitle, p.Error = "Message not found", "The archive may have been replaced. Search again or select another conversation."

				if p.Section == "snaps" {
					p.ErrorTitle = "Snap not found"
				}
			}
		} else {
			status = http.StatusInternalServerError

			p.ErrorTitle, p.Error = "Unable to load messages", "Please try again in a moment."

			slog.ErrorContext(r.Context(), "load archive", "error", err)
		}
	}

	h.render(w, r, p, status)
}

func (h *handler) render(w http.ResponseWriter, r *http.Request, p page, status int) {
	p.CanManage = h.managed

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request, HX-Target, HX-History-Restore-Request, Cookie")

	name := "page"

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-History-Restore-Request") != "true" {
		name = "workspace"

		if r.Header.Get("HX-Target") == "load-older" && p.Query == "" && p.IsHistory() {
			name = "history"
		}
	}

	var body bytes.Buffer

	if err := h.templates.ExecuteTemplate(&body, name, p); err != nil {
		slog.ErrorContext(r.Context(), "render archive", "error", err)

		http.Error(w, "Unable to render the archive. Please reload to try again.", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(status)

	_, _ = w.Write(body.Bytes())
}

func requestTheme(r *http.Request) string {
	if cookie, err := r.Cookie(themeCookieName); err == nil && (cookie.Value == "light" || cookie.Value == "dark") {
		return cookie.Value
	}

	return "system"
}

func (h *handler) setTheme(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid theme preference", http.StatusBadRequest)

		return
	}

	theme := r.PostForm.Get("theme")

	if theme != "system" && theme != "light" && theme != "dark" {
		http.Error(w, "Choose System, Light, or Dark", http.StatusBadRequest)

		return
	}

	http.SetCookie(w, &http.Cookie{
		Name: themeCookieName, Value: theme, Path: "/", MaxAge: 365 * 24 * 60 * 60,
		HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode,
	})

	if r.Header.Get("HX-Request") == "true" {
		// CSS applies the selected option immediately. Persist it without swapping
		// any HTML, preserving the active search, conversation, and scroll position.
		w.WriteHeader(http.StatusNoContent)

		return
	}

	// A native form submission returns to the same view without JavaScript.
	target := "/"

	if u, err := url.Parse(r.PostForm.Get("return_to")); err == nil && !u.IsAbs() && u.Host == "" && archivePath(u.Path) {
		target = u.RequestURI()
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (h *handler) load(ctx context.Context, p *page) error {
	if p.MessageID != 0 {
		getPosition := h.store.GetChatPosition

		if p.Section == "snaps" {
			getPosition = h.store.GetSnapPosition
		}

		position, err := getPosition(ctx, p.MessageID)
		if err != nil {
			return err
		}

		p.Query = ""

		p.ConversationID = position.ConversationID

		p.Offset = max(0, position.Offset-messagePageSize/2)
	}

	account, err := h.store.GetAccount(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if account != nil {
		p.AccountName = first(account.BasicInformation.Name, account.BasicInformation.Username)

		p.AccountUsername = account.BasicInformation.Username
	}

	listConversations := h.store.ListConversations

	if p.Section == "snaps" {
		listConversations = h.store.ListSnapConversations
	}

	conversations, err := listConversations(ctx, db.Pagination{Limit: conversationPageSize + 1, Offset: p.ConversationOffset})
	if err != nil {
		return err
	}

	p.MoreConversations = len(conversations) > conversationPageSize

	p.Conversations = conversations[:min(len(conversations), conversationPageSize)]

	names := make(map[int64]string, len(p.Conversations))

	for _, c := range p.Conversations {
		names[c.ID] = c.Key
	}

	if p.ConversationID != 0 {
		selected, err := h.store.GetConversation(ctx, p.ConversationID)
		if err != nil {
			return err
		}

		p.Selected = &selected

		names[selected.ID] = selected.Key
	}

	chats, err := h.listHistory(ctx, p, db.MessageOptions{
		Pagination:     db.Pagination{Limit: messagePageSize + 1, Offset: p.Offset},
		ConversationID: p.ConversationID,
		Query:          searchQuery(p.Query),
	})
	if err != nil {
		return err
	}

	p.MoreMessages = len(chats) > messagePageSize

	chats = chats[:min(len(chats), messagePageSize)]

	if p.Query == "" {
		// Fetch the most recent window, then render it in reading order. Older
		// windows can be prepended without moving or replacing existing messages.
		slices.Reverse(chats)
	}

	previousDay := ""

	for _, c := range chats {
		created, day, datetime := messageDate(c.Created, c.CreatedMicrosecond)

		text := ""

		if c.Content != nil {
			text = *c.Content
		}

		conversationName := names[c.ConversationID]

		if c.ConversationTitle != nil && strings.TrimSpace(*c.ConversationTitle) != "" {
			conversationName = *c.ConversationTitle
		}

		p.Messages = append(p.Messages, message{
			ID: c.ID, From: first(c.From, "Unknown sender"), Text: text,
			MediaType: first(c.MediaType, "Media"), Saved: c.IsSaved, Outgoing: c.IsSender,
			Created: created, Day: day, DateTime: datetime, NewDay: day != previousDay,
			ConversationName: first(conversationName, fmt.Sprintf("Conversation %d", c.ConversationID)),
			ConversationURL:  p.URL(c.ConversationID, 0, p.ConversationOffset),
			Attachments:      h.attachments(c.Chat),
			ContextURL:       p.MessageContextURL(c.ConversationID, c.ID),
			Highlighted:      c.ID == p.MessageID,
		})

		previousDay = day
	}

	return nil
}

func (h *handler) listHistory(ctx context.Context, p *page, options db.MessageOptions) ([]db.Chat, error) {
	if p.Section != "snaps" {
		return h.store.ListChats(ctx, options)
	}

	snaps, err := h.store.ListSnaps(ctx, options)
	if err != nil {
		return nil, err
	}

	items := make([]db.Chat, 0, len(snaps))

	for _, snap := range snaps {
		title := snap.ConversationTitle

		if title == nil {
			title = &snap.ConversationKey
		}

		items = append(items, db.Chat{
			ID: snap.ID, ConversationID: snap.ConversationID,
			Chat: parser.Chat{
				From: snap.From, MediaType: snap.MediaType, Created: snap.Created,
				ConversationTitle: title, IsSender: snap.IsSender, CreatedMicrosecond: snap.CreatedMicroseconds,
			},
		})
	}

	return items, nil
}

// Treat search as words, never as raw FTS syntax. Each word is a quoted prefix,
// so punctuation and quotes cannot turn a normal search into a database error.
func searchQuery(q string) string {
	words := strings.Fields(q)

	for i, word := range words {
		words[i] = `"` + strings.ReplaceAll(word, `"`, `""`) + `"*`
	}

	return strings.Join(words, " AND ")
}

func nonnegative(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}

	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil || n < 0 {
		return 0, errors.New("invalid nonnegative integer")
	}

	return n, nil
}

func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func number(n int64) string {
	s := strconv.FormatInt(n, 10)

	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}

	return s
}

func date(microseconds int64) string {
	if microseconds == 0 {
		return "No date"
	}

	return time.UnixMicro(microseconds).UTC().Format("Jan 2, 2006")
}

func messageDate(raw string, microseconds int64) (string, string, string) {
	var t time.Time

	if microseconds != 0 {
		t = time.UnixMicro(microseconds).UTC()
	} else {
		for _, layout := range []string{"2006-01-02 15:04:05 MST", time.RFC3339, "2006-01-02 15:04:05"} {
			parsed, err := time.Parse(layout, raw)
			if err == nil {
				t = parsed.UTC()

				break
			}
		}
	}

	if t.IsZero() {
		return first(raw, "Time unknown"), "Date unknown", ""
	}

	return t.Format("15:04:05"), t.Format("2006-01-02"), t.Format(time.RFC3339)
}
