// Package parser parses a snapchat chat history json file to a struct
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
	Content            string //`json:",omitempty"`
	ConversationTitle  string `json:"Conversation Title"`
	IsSender           bool   `json:"isSender"`
	CreatedMicrosecond int64  `json:"Created(microseconds)"`
}

// ParseAll parses the chat history.
// It returns an array with all usernames, a map of usernames and Content array,
// and any error encountered.
func ParseAll(in io.Reader) (data map[string][]Content, err error) {
	inByte, err := io.ReadAll(in)
	if err != nil {
		return
	}

	if err = json.Unmarshal(inByte, &data); err != nil {
		return
	}

	return
}

// ParseUser parses the chat history with one user.
// It returns a Content array, and any encountered error.
func ParseUser(in io.Reader, user string) (content []Content, err error) {
	var data map[string][]Content

	inByte, err := io.ReadAll(in)
	if err != nil {
		return
	}

	if err = json.Unmarshal(inByte, &data); err != nil {
		return
	}

	for k := range data {
		if k == user {
			return data[k], nil
		}
	}

	return nil, fmt.Errorf("no chats with user: %s", user)
}
