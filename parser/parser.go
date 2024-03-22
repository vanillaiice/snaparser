// Package parser parses snapchat chat history from a json file to a struct.
package parser

import (
	"encoding/json"
	"fmt"
	"io"
)

// Content contains the fields in the chat history json file
type Content struct {
	From               string
	MediaType          string `json:"Media Type"`
	Created            string
	Content            string `json:",omitempty"`
	ConversationTitle  string `json:"Conversation Title"`
	IsSender           bool   `json:"isSender"`
	CreatedMicrosecond int64  `json:"Created(microseconds)"`
}

// ParseAll parses the chat history.It returns a map of usernames
// a slice of Content, and any error encountered.
func ParseAll(r io.Reader) (data map[string][]*Content, err error) {
	in, err := io.ReadAll(r)
	if err != nil {
		return
	}
	err = json.Unmarshal(in, &data)
	return
}

// ParseUser parses the chat history with one user.
// It returns a Content array, and any encountered error.
func ParseUser(r io.Reader, user string) (content []*Content, err error) {
	var data map[string][]*Content

	in, err := io.ReadAll(r)
	if err != nil {
		return
	}

	if err = json.Unmarshal(in, &data); err != nil {
		return
	}

	for k := range data {
		if k == user {
			return data[k], nil
		}
	}

	return nil, fmt.Errorf("no chats with user: %s", user)
}
