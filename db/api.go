package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vanillaiice/snaparser/parser"
)

// Import atomically replaces application data with the supplied parsed backup.
// All files must be parsed and validated before calling Import. Omitted fields
// are cleared along with the previous backup. The source export root is recorded
// in the same transaction; media filenames and bytes are resolved by the viewer.
func (d *DB) Import(ctx context.Context, backup parser.Backup) error {
	files := []struct {
		name    string
		present bool
		load    func(*backupWriter) error
	}{
		{"account.json", backup.Account != nil, func(w *backupWriter) error { return w.account(backup.Account) }},
		{"account_history.json", backup.AccountHistory != nil, func(w *backupWriter) error { return w.accountHistory(backup.AccountHistory) }},
		{"chat_history.json", backup.ChatHistory != nil, func(w *backupWriter) error { return w.chats(*backup.ChatHistory) }},
		{"friends.json", backup.Friends != nil, func(w *backupWriter) error { return w.friends(backup.Friends) }},
		{"snap_history.json", backup.SnapHistory != nil, func(w *backupWriter) error { return w.snaps(*backup.SnapHistory) }},
		{"snap_pro.json", backup.SnapProfile != nil, func(w *backupWriter) error { return w.snapProfile(backup.SnapProfile) }},
		{"talk_history.json", backup.TalkHistory != nil, func(w *backupWriter) error { return w.calls(backup.TalkHistory) }},
		{"user_profile.json", backup.UserProfile != nil, func(w *backupWriter) error { return w.userProfile(backup.UserProfile) }},
	}

	hasFiles := false

	for _, file := range files {
		hasFiles = hasFiles || file.present
	}

	if !hasFiles {
		return errors.New("import backup: no parsed files supplied")
	}

	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin backup import: %w", err)
	}

	defer tx.Rollback()

	// Delete only source rows. FTS triggers run inside this same transaction.
	for _, table := range []string{
		"chats", "snaps", "conversations", "friends", "device_history",
		"login_history", "account_changes", "calls", "snap_profiles",
		"user_profiles", "engagements", "discover_channels", "ad_interactions",
		"user_profile_entries", "import_history", "accounts", "archive_source",
	} {
		if _, err := tx.ExecContext(ctx, "delete from "+table); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}

	w := &backupWriter{ctx: ctx, tx: tx, statements: make(map[string]*sql.Stmt), conversations: make(map[string]int64)}

	defer w.close()

	for _, file := range files {
		if !file.present {
			continue
		}

		before := w.inserted

		if err := file.load(w); err != nil {
			return fmt.Errorf("import %s: %w", file.name, err)
		}

		count := w.inserted - before

		name := file.name

		for _, supplied := range backup.Filenames {
			if filepath.Base(supplied) == file.name {
				name = supplied

				break
			}
		}

		if err := w.insert("import_history", "filename, inserted_count", name, count); err != nil {
			return fmt.Errorf("record import metadata for %s: %w", name, err)
		}
	}

	if backup.ExportDirectory != "" {
		if _, err := tx.ExecContext(ctx, "insert into archive_source (export_directory) values (?)", backup.ExportDirectory); err != nil {
			return fmt.Errorf("record export directory: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit backup import: %w", err)
	}

	return nil
}

// backupWriter caches statements for the duration of one import transaction.
// Table and column names below are internal constants; values are parameters.
type backupWriter struct {
	ctx           context.Context
	tx            *sql.Tx
	statements    map[string]*sql.Stmt
	conversations map[string]int64
	inserted      int64
}

func (w *backupWriter) close() {
	for _, stmt := range w.statements {
		stmt.Close()
	}
}

func (w *backupWriter) insert(table, columns string, values ...any) error {
	query := "insert into " + table + " (" + columns + ") values (" + strings.TrimSuffix(strings.Repeat("?,", len(values)), ",") + ")"

	stmt := w.statements[query]

	if stmt == nil {
		var err error

		stmt, err = w.tx.PrepareContext(w.ctx, query)
		if err != nil {
			return fmt.Errorf("prepare %s: %w", table, err)
		}

		w.statements[query] = stmt
	}

	if _, err := stmt.ExecContext(w.ctx, values...); err != nil {
		return fmt.Errorf("insert %s: %w", table, err)
	}

	w.inserted++

	return nil
}

func (w *backupWriter) account(a *parser.Account) error {
	b, d := a.BasicInformation, a.DeviceInformation

	if err := w.insert("accounts", "username, name, creation_date, registration_ip, country, last_active, device_make, device_model_id, device_model_name, device_language, device_os_type, device_os_version, device_connection_type",
		b.Username, b.Name, b.CreationDate, b.RegistrationIP, b.Country, b.LastActive,
		d.Make, d.ModelID, d.ModelName, d.Language, d.OSType, d.OSVersion, d.ConnectionType); err != nil {
		return err
	}

	for _, d := range a.DeviceHistory {
		if err := w.insert("device_history", "make, model, start_time, device_type", d.Make, d.Model, d.StartTime, d.DeviceType); err != nil {
			return err
		}
	}

	for _, l := range a.LoginHistory {
		if err := w.insert("login_history", "ip, country, created, status, device", l.IP, l.Country, l.Created, l.Status, l.Device); err != nil {
			return err
		}
	}

	return nil
}

func (w *backupWriter) accountHistory(h *parser.AccountHistory) error {
	for _, c := range h.DisplayNameChange {
		if err := w.insert("account_changes", "change_type, date, status, value", "display_name", c.Date, "", c.DisplayName); err != nil {
			return err
		}
	}

	for _, c := range h.EmailChange {
		if err := w.insert("account_changes", "change_type, date, status, value", "email", c.Date, "", c.EmailAddress); err != nil {
			return err
		}
	}

	for _, c := range h.MobileNumberChange {
		if err := w.insert("account_changes", "change_type, date, status, value", "mobile_number", c.Date, "", c.MobileNumber); err != nil {
			return err
		}
	}

	for _, c := range h.PasswordChange {
		if err := w.insert("account_changes", "change_type, date, status, value", "password", c.Date, "", ""); err != nil {
			return err
		}
	}

	for _, c := range h.SnapchatLinkedToBitmoji {
		if err := w.insert("account_changes", "change_type, date, status, value", "bitmoji_linked", c.Date, "", ""); err != nil {
			return err
		}
	}

	for _, c := range h.DownloadMyDataReports {
		if err := w.insert("account_changes", "change_type, date, status, value", "data_report", c.Date, c.Status, c.EmailAddress); err != nil {
			return err
		}
	}

	return nil
}

func (w *backupWriter) friends(f *parser.Friends) error {
	for _, group := range []struct {
		category string
		friends  []parser.Friend
	}{
		{"friends", f.Friends},
		{"requests_sent", f.FriendsRequestsSent},
		{"blocked", f.BlockedUsers},
		{"deleted", f.DeletedFriends},
		{"hidden_suggestions", f.HiddenFriendSuggestions},
		{"ignored", f.IgnoredSnapchatters},
		{"pending", f.PendingRequests},
	} {
		for _, f := range group.friends {
			if err := w.insert("friends", "username, display_name, creation_timestamp, last_modified_timestamp, source, category",
				f.Username, f.DisplayName, f.CreationTimestamp, f.LastModifiedTimestamp, f.Source, group.category); err != nil {
				return err
			}
		}
	}

	return nil
}

// Chats and snaps with the same key share one conversation within this backup.
func (w *backupWriter) conversation(key string) (int64, error) {
	if id, ok := w.conversations[key]; ok {
		return id, nil
	}

	id := int64(len(w.conversations) + 1)

	if err := w.insert("conversations", "id, conversation_key", id, key); err != nil {
		return 0, err
	}

	w.conversations[key] = id

	return id, nil
}

func (w *backupWriter) chats(history parser.ChatHistory) error {
	keys := make([]string, 0, len(history))

	for key := range history {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	for _, key := range keys {
		id, err := w.conversation(key)
		if err != nil {
			return err
		}

		for _, c := range history[key] {
			if err := w.insert("chats", "conversation_id, sender, media_type, created, content, conversation_title, is_sender, created_microseconds, is_saved, media_ids",
				id, c.From, c.MediaType, c.Created, c.Content, c.ConversationTitle, c.IsSender, snapTimestampMicroseconds(c.CreatedMicrosecond), c.IsSaved, c.MediaIDS); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *backupWriter) snaps(history parser.SnapHistory) error {
	keys := make([]string, 0, len(history))

	for key := range history {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	for _, key := range keys {
		id, err := w.conversation(key)
		if err != nil {
			return err
		}

		for _, s := range history[key] {
			if err := w.insert("snaps", "conversation_id, sender, media_type, created, conversation_title, is_sender, created_microseconds",
				id, s.From, s.MediaType, s.Created, s.ConversationTitle, s.IsSender, snapTimestampMicroseconds(s.CreatedMicroseconds)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *backupWriter) snapProfile(s *parser.SnapProfile) error {
	p := s.Profile

	return w.insert("snap_profiles", "created, profile_title, location, profile_website, profile_photo, hero_image",
		p.Created, p.ProfileTitle, p.Location, p.ProfileWebsite, p.ProfilePhoto, p.HeroImage)
}

func (w *backupWriter) calls(h *parser.TalkHistory) error {
	for _, group := range []struct {
		kind  string
		calls []parser.Call
	}{
		{"outgoing", h.OutgoingCalls},
		{"incoming", h.IncomingCalls},
	} {
		for _, c := range group.calls {
			if err := w.insert("calls", "call_type, date_time, type, people_in_chat, result, city, country, length_seconds, network",
				group.kind, c.DateTime, c.Type, c.PeopleInChat, c.Result, c.City, c.Country, c.Length, c.Network); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *backupWriter) userProfile(u *parser.UserProfile) error {
	a, d := u.AppProfile, u.Demographics

	if err := w.insert("user_profiles", "country, creation_time, account_creation_country, platform_version, in_app_language, cohort_age, derived_ad_demographic, mobile_ad_id",
		a.Country, a.CreationTime, a.AccountCreationCountry, a.PlatformVersion, a.InAppLanguage, d.CohortAge, d.DerivedAdDemographic, u.MobileAdID); err != nil {
		return err
	}

	for _, e := range u.Engagement {
		if err := w.insert("engagements", "event, occurrences", e.Event, e.Occurrences); err != nil {
			return err
		}
	}

	for _, c := range u.DiscoverChannelsViewed {
		if err := w.insert("discover_channels", "channel_name", c.ChannelName); err != nil {
			return err
		}
	}

	for _, a := range u.AdsYouInteractedWith {
		if err := w.insert("ad_interactions", "advertiser_name, date", a.AdvertiserName, a.Date); err != nil {
			return err
		}
	}

	for _, group := range []struct {
		category string
		values   []string
	}{
		{"time_spent", u.BreakdownOfTimeSpentOnApp},
		{"content", u.ContentCategories},
		{"web_interactions", u.Interactions.WebInteractions},
		{"app_interactions", u.Interactions.AppInteractions},
	} {
		for _, value := range group.values {
			if err := w.insert("user_profile_entries", "category, value", group.category, value); err != nil {
				return err
			}
		}
	}

	return nil
}
