package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
)

type Parser func() any

var parsers = map[string]Parser{
	"account.json":         func() any { return &Account{} },
	"account_history.json": func() any { return &AccountHistory{} },
	// "bitmoji.json": func() any { return &Bitmoji{},
	"chat_history.json": func() any { return &ChatHistory{} },
	// "connected_apps.json": func() any { return &ConnectedApps{} },
	// "feature_emails.json": func() any { return &FeatureEmails{} },
	"friends.json":      func() any { return &Friends{} },
	"snap_history.json": func() any { return &SnapHistory{} },
	"snap_pro.json":     func() any { return &SnapProfile{} },
	"talk_history.json": func() any { return &TalkHistory{} },
	"user_profile.json": func() any { return &UserProfile{} },
}

func Parse(filename string, r io.Reader) (any, error) {
	base := filepath.Base(filename)
	parser, ok := parsers[base]
	if !ok {
		return nil, fmt.Errorf("unsupported file: %s", base)
	}

	data := parser()

	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	if data == nil {
		return nil, fmt.Errorf("%s: expected a JSON object, got null", base)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("%s: expected one JSON document", base)
		}
		return nil, fmt.Errorf("%s: trailing data: %w", base, err)
	}

	return data, nil
}
