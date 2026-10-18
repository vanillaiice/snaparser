package db

import (
	"context"
	"database/sql"

	"github.com/vanillaiice/snaparser/v2/parser"
)

// GetAccount returns the account and its device/login histories from one
// consistent snapshot. It returns sql.ErrNoRows when no account was imported.
func (d *DB) GetAccount(ctx context.Context) (*parser.Account, error) {
	tx, err := d.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	a := &parser.Account{}

	b, device := &a.BasicInformation, &a.DeviceInformation

	err = tx.QueryRowContext(ctx, `select username, name, creation_date, registration_ip, country, last_active,
        device_make, device_model_id, device_model_name, device_language, device_os_type, device_os_version,
        device_connection_type from accounts where id = 1`).Scan(
		&b.Username, &b.Name, &b.CreationDate, &b.RegistrationIP, &b.Country, &b.LastActive,
		&device.Make, &device.ModelID, &device.ModelName, &device.Language, &device.OSType, &device.OSVersion, &device.ConnectionType)
	if err != nil {
		return nil, err
	}

	a.DeviceHistory, err = readRows(ctx, tx, "select make, model, start_time, device_type from device_history order by id", nil, func(row rowScanner) (parser.DeviceHistory, error) {
		var h parser.DeviceHistory

		err := row.Scan(&h.Make, &h.Model, &h.StartTime, &h.DeviceType)

		return h, err
	})
	if err != nil {
		return nil, err
	}

	a.LoginHistory, err = readRows(ctx, tx, "select ip, country, created, status, device from login_history order by id", nil, func(row rowScanner) (parser.LoginHistory, error) {
		var h parser.LoginHistory

		err := row.Scan(&h.IP, &h.Country, &h.Created, &h.Status, &h.Device)

		return h, err
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return a, nil
}

// GetSnapProfile returns the stored profile. Sections represented by parser
// TODO types are not stored. A missing profile returns sql.ErrNoRows.
func (d *DB) GetSnapProfile(ctx context.Context) (*parser.SnapProfile, error) {
	s := &parser.SnapProfile{}

	p := &s.Profile

	err := d.conn.QueryRowContext(ctx, `select created, profile_title, location, profile_website,
        profile_photo, hero_image from snap_profiles where id = 1`).Scan(
		&p.Created, &p.ProfileTitle, &p.Location, &p.ProfileWebsite, &p.ProfilePhoto, &p.HeroImage)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// GetUserProfile returns all currently modeled profile fields and collections
// from one consistent snapshot. A missing profile returns sql.ErrNoRows.
func (d *DB) GetUserProfile(ctx context.Context) (*parser.UserProfile, error) {
	tx, err := d.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	u := &parser.UserProfile{}

	a, demo := &u.AppProfile, &u.Demographics

	err = tx.QueryRowContext(ctx, `select country, creation_time, account_creation_country, platform_version,
        in_app_language, cohort_age, derived_ad_demographic, mobile_ad_id from user_profiles where id = 1`).Scan(
		&a.Country, &a.CreationTime, &a.AccountCreationCountry, &a.PlatformVersion, &a.InAppLanguage,
		&demo.CohortAge, &demo.DerivedAdDemographic, &u.MobileAdID)
	if err != nil {
		return nil, err
	}

	u.Engagement, err = readRows(ctx, tx, "select event, occurrences from engagements order by id", nil, func(row rowScanner) (parser.Engagement, error) {
		var e parser.Engagement

		err := row.Scan(&e.Event, &e.Occurrences)

		return e, err
	})
	if err != nil {
		return nil, err
	}

	u.DiscoverChannelsViewed, err = readRows(ctx, tx, "select channel_name from discover_channels order by id", nil, func(row rowScanner) (parser.DiscoverChannelsViewed, error) {
		var c parser.DiscoverChannelsViewed

		err := row.Scan(&c.ChannelName)

		return c, err
	})
	if err != nil {
		return nil, err
	}

	u.AdsYouInteractedWith, err = readRows(ctx, tx, "select advertiser_name, date from ad_interactions order by id", nil, func(row rowScanner) (parser.AdsYouInteractedWith, error) {
		var a parser.AdsYouInteractedWith

		err := row.Scan(&a.AdvertiserName, &a.Date)

		return a, err
	})
	if err != nil {
		return nil, err
	}

	type entry struct{ category, value string }

	entries, err := readRows(ctx, tx, "select category, value from user_profile_entries order by id", nil, func(row rowScanner) (entry, error) {
		var e entry

		err := row.Scan(&e.category, &e.value)

		return e, err
	})
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		switch e.category {
		case "time_spent":
			u.BreakdownOfTimeSpentOnApp = append(u.BreakdownOfTimeSpentOnApp, e.value)

		case "content":
			u.ContentCategories = append(u.ContentCategories, e.value)

		case "web_interactions":
			u.Interactions.WebInteractions = append(u.Interactions.WebInteractions, e.value)

		case "app_interactions":
			u.Interactions.AppInteractions = append(u.Interactions.AppInteractions, e.value)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return u, nil
}
