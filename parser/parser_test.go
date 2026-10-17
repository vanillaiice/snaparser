package parser

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// errorReader is a custom io.Reader that always returns an error.
type errorReader struct{}

func (er *errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("simulated read error")
}

func TestParse(t *testing.T) {
	files := []string{
		"account.json",
		"account_history.json",
		"chat_history.json",
		"friends.json",
		"snap_history.json",
		"snap_pro.json",
		"talk_history.json",
		"user_profile.json",
	}

	for _, f := range files {
		path := filepath.Join("../data/json/", f)

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		r := bytes.NewReader(content)

		if _, err := Parse(path, r); err != nil {
			t.Error(err)
		}
	}
}
