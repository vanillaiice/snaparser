package frontend

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/vanillaiice/snaparser/db"
)

const recordPageSize = 40

type detailField struct {
	Label string
	Value string
}

type detailGroup struct {
	Title  string
	Fields []detailField
}

func label(value string) string {
	labels := map[string]string{
		"friends": "Friends", "requests_sent": "Requests sent", "blocked": "Blocked",
		"deleted": "Deleted", "hidden_suggestions": "Hidden suggestions", "ignored": "Ignored", "pending": "Pending requests",
		"incoming": "Incoming", "outgoing": "Outgoing", "display_name": "Display name changed",
		"email": "Email changed", "mobile_number": "Mobile number changed", "password": "Password changed",
		"bitmoji_linked": "Bitmoji linked", "data_report": "Data report requested",
	}

	return first(labels[value], value)
}

func archivePath(path string) bool {
	switch path {
	case "/", "/friends", "/snaps", "/calls", "/account":
		return true
	}

	return false
}

func (p page) RecordsURL(offset int) string {
	v := url.Values{}

	if p.Query != "" {
		v.Set("q", p.Query)
	}

	if p.Filter != "" {
		v.Set("filter", p.Filter)
	}

	if p.ConversationID != 0 {
		v.Set("conversation", strconv.FormatInt(p.ConversationID, 10))
	}

	if offset > 0 {
		v.Set("offset", strconv.Itoa(offset))
	}

	path := "/" + p.Section

	if len(v) != 0 {
		path += "?" + v.Encode()
	}

	return path
}

func (p page) CurrentURL() string {
	if !p.IsHistory() {
		return p.RecordsURL(p.Offset)
	}

	return p.URL(p.ConversationID, p.Offset, p.ConversationOffset)
}

func (p page) PreviousRecords() string {
	return p.RecordsURL(max(0, p.Offset-recordPageSize))
}

func (p page) NextRecords() string {
	return p.RecordsURL(p.Offset + recordPageSize)
}

func (p page) RecordCount() int {
	switch p.Section {
	case "friends":
		return len(p.Friends)

	case "calls":
		return len(p.Calls)

	default:
		return len(p.Changes)
	}
}

func (p page) RecordEnd() int {
	return p.Offset + p.RecordCount()
}

func (h *handler) details(w http.ResponseWriter, r *http.Request) {
	p := page{
		Section: strings.TrimPrefix(r.URL.Path, "/"),
		Theme:   requestTheme(r), Query: strings.TrimSpace(r.URL.Query().Get("q")),
		Filter: r.URL.Query().Get("filter"),
	}

	p.Title = map[string]string{"friends": "Friends", "calls": "Calls", "account": "Account"}[p.Section]

	offset, err := nonnegative(r.URL.Query().Get("offset"))

	p.Offset = int(offset)

	if err == nil {
		p.ConversationID, err = nonnegative(r.URL.Query().Get("conversation"))
	}

	validFilter := p.Filter == ""

	switch p.Section {
	case "friends":
		for _, category := range []string{"friends", "requests_sent", "blocked", "deleted", "hidden_suggestions", "ignored", "pending"} {
			validFilter = validFilter || p.Filter == category
		}

	case "calls":
		validFilter = validFilter || p.Filter == "incoming" || p.Filter == "outgoing"
	}

	status := http.StatusOK

	if err != nil || !validFilter || utf8.RuneCountInString(p.Query) > 500 {
		status = http.StatusBadRequest

		p.ErrorTitle, p.Error = "Invalid request", "Use valid filters, page numbers, and a search of 500 characters or fewer."
	} else if err := h.loadDetails(r.Context(), &p); err != nil {
		status = http.StatusInternalServerError

		p.ErrorTitle, p.Error = "Unable to load archive", "Please try again in a moment."

		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound

			p.ErrorTitle, p.Error = "Conversation not found", "The archive may have been replaced. Select another conversation."
		} else {
			slog.ErrorContext(r.Context(), "load archive details", "section", p.Section, "error", err)
		}
	}

	h.render(w, r, p, status)
}

func (h *handler) loadDetails(ctx context.Context, p *page) error {
	account, err := h.store.GetAccount(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	p.Account = account

	if account != nil {
		p.AccountName = first(account.BasicInformation.Name, account.BasicInformation.Username)

		p.AccountUsername = account.BasicInformation.Username
	}

	pagination := db.Pagination{Limit: recordPageSize + 1, Offset: p.Offset}

	switch p.Section {
	case "friends":
		friends, err := h.store.ListFriends(ctx, db.FriendOptions{Pagination: pagination, Category: p.Filter, Query: searchQuery(p.Query)})
		if err != nil {
			return err
		}

		p.MoreRecords = len(friends) > recordPageSize

		p.Friends = friends[:min(len(friends), recordPageSize)]

	case "calls":
		calls, err := h.store.ListCalls(ctx, db.CallOptions{Pagination: pagination, Direction: p.Filter, Query: p.Query})
		if err != nil {
			return err
		}

		p.MoreRecords = len(calls) > recordPageSize

		p.Calls = calls[:min(len(calls), recordPageSize)]

	case "account":
		return h.loadAccount(ctx, p, pagination)
	}

	return nil
}

func (h *handler) loadAccount(ctx context.Context, p *page, pagination db.Pagination) error {
	if p.Account != nil {
		a, d := p.Account.BasicInformation, p.Account.DeviceInformation

		p.Details = append(p.Details,
			detailGroup{"Account details", []detailField{
				{"Username", a.Username}, {"Name", a.Name}, {"Created", a.CreationDate},
				{"Registration IP", a.RegistrationIP}, {"Country", a.Country}, {"Last active", a.LastActive},
			}},
			detailGroup{"Device", []detailField{
				{"Make", d.Make}, {"Model ID", d.ModelID}, {"Model", d.ModelName}, {"Language", d.Language},
				{"Operating system", d.OSType}, {"OS version", d.OSVersion}, {"Connection", d.ConnectionType},
			}},
		)
	}

	snap, err := h.store.GetSnapProfile(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if snap != nil {
		s := snap.Profile

		p.Details = append(p.Details, detailGroup{"Public profile", []detailField{
			{"Title", s.ProfileTitle}, {"Created", s.Created}, {"Location", s.Location},
			{"Website", s.ProfileWebsite}, {"Profile photo", s.ProfilePhoto}, {"Hero image", s.HeroImage},
		}})
	}

	profile, err := h.store.GetUserProfile(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	p.Profile = profile

	if profile != nil {
		a, d := profile.AppProfile, profile.Demographics

		p.Details = append(p.Details, detailGroup{"App profile and demographics", []detailField{
			{"Country", a.Country}, {"Created", a.CreationTime}, {"Account creation country", a.AccountCreationCountry},
			{"Platform version", a.PlatformVersion}, {"Language", a.InAppLanguage},
			{"Age cohort", d.CohortAge}, {"Advertising demographic", d.DerivedAdDemographic}, {"Mobile advertising ID", profile.MobileAdID},
		}})
	}

	changes, err := h.store.ListAccountChanges(ctx, pagination)
	if err != nil {
		return err
	}

	p.MoreRecords = len(changes) > recordPageSize

	p.Changes = changes[:min(len(changes), recordPageSize)]

	return nil
}
