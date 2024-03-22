package parser_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/vanillaiice/snaparser/parser"
)

// errorReader is a custom io.Reader that always returns an error.
type errorReader struct{}

func (er *errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("simulated read error")
}

func TestParseAll(t *testing.T) {
	// Test case: valid JSON input
	validJSON := `{
        "user1": [
            {
                "From": "user1",
                "Media Type": "Text",
                "Created": "2023-03-19T12:34:56Z",
                "Content": "Hello",
                "Conversation Title": "Chat 1",
                "isSender": true,
                "Created(microseconds)": 1679234096000000
            }
        ],
        "user2": [
            {
                "From": "user2",
                "Media Type": "Text",
                "Created": "2023-03-19T14:56:78Z",
                "Content": "Hi",
                "Conversation Title": "Chat 2",
                "isSender": false,
                "Created(microseconds)": 1679246238000000
            }
        ]
    }`

	expected := map[string][]*parser.Content{
		"user1": {{
			From:               "user1",
			MediaType:          "Text",
			Created:            "2023-03-19T12:34:56Z",
			Content:            "Hello",
			ConversationTitle:  "Chat 1",
			IsSender:           true,
			CreatedMicrosecond: 1679234096000000,
		}},
		"user2": {{
			From:               "user2",
			MediaType:          "Text",
			Created:            "2023-03-19T14:56:78Z",
			Content:            "Hi",
			ConversationTitle:  "Chat 2",
			IsSender:           false,
			CreatedMicrosecond: 1679246238000000,
		}},
	}

	data, err := parser.ParseAll(bytes.NewBufferString(validJSON))
	if err != nil {
		t.Errorf("ParseAll() error = %v", err)
		return
	}

	if diff := cmp.Diff(data, expected); diff != "" {
		t.Errorf("ParseAll() data mismatch (-got +want):\n%s", diff)
	}

	// Test case: invalid JSON input
	invalidJSON := `invalid json`
	_, err = parser.ParseAll(bytes.NewBufferString(invalidJSON))
	if err == nil {
		t.Error("ParseAll() expected error for invalid JSON input, got nil")
	}

	errReader := &errorReader{}
	_, err = parser.ParseAll(errReader)
	if err == nil {
		t.Error("ParseAll() expected error for failed io.Reader, got nil")
	}
}

func TestParseUser(t *testing.T) {
	jsonData := `{
        "user1": [
            {
                "From": "user1",
                "Media Type": "Text",
                "Created": "2023-03-19T12:34:56Z",
                "Content": "Hello",
                "Conversation Title": "Chat 1",
                "isSender": true,
                "Created(microseconds)": 1679234096000000
            }
        ],
        "user2": [
            {
                "From": "user2",
                "Media Type": "Text",
                "Created": "2023-03-19T14:56:78Z",
                "Content": "Hi",
                "Conversation Title": "Chat 2",
                "isSender": false,
                "Created(microseconds)": 1679246238000000
            }
        ]
    }`

	expected := []*parser.Content{{
		From:               "user1",
		MediaType:          "Text",
		Created:            "2023-03-19T12:34:56Z",
		Content:            "Hello",
		ConversationTitle:  "Chat 1",
		IsSender:           true,
		CreatedMicrosecond: 1679234096000000,
	}}

	content, err := parser.ParseUser(bytes.NewBufferString(jsonData), "user1")
	if err != nil {
		t.Errorf("ParseUser() error = %v", err)
		return
	}

	if diff := cmp.Diff(content, expected); diff != "" {
		t.Errorf("ParseUser() content mismatch (-got +want):\n%s", diff)
	}

	// Test case: non-existent user
	_, err = parser.ParseUser(bytes.NewBufferString(jsonData), "user3")
	if err == nil {
		t.Error("ParseUser() expected error for non-existent user, got nil")
	}

	// Test case: invalid JSON input
	invalidJSON := `invalid json`
	_, err = parser.ParseUser(bytes.NewBufferString(invalidJSON), "user1")
	if err == nil {
		t.Error("ParseUser() expected error for invalid JSON input, got nil")
	}

	errReader := &errorReader{}
	_, err = parser.ParseUser(errReader, "user1")
	if err == nil {
		t.Error("ParseUser() expected error for failed io.Reader, got nil")
	}
}
