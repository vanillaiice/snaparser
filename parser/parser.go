// Package parser parses a snapchat chat history json file to a struct
package parser

import (
	"encoding/json"
	"io"
)

// Content contains the fields in the chat history json file
type Content struct {
	From               string
	MediaType          string `json:"Media Type"`
	Created            string
	Content            string
	ConversationTitle  string `json:"Conversation Title"`
	IsSender           bool   `json:"isSender"`
	CreatedMicrosecond int64  `json:"Created(microseconds)"`
}

// ParseAll parses the chat history.
// It returns an array with all usernames, a map of usernames and Content array, and any error encountered.
func ParseAll(in io.Reader) ([]string, map[string][]Content, error) {
	var data map[string][]Content
	var users []string

	inByte, err := io.ReadAll(in)
	if err != nil {
		return users, data, err
	}

	err = json.Unmarshal(inByte, &data)
	if err != nil {
		return users, data, err
	}

	for k, _ := range data {
		users = append(users, k)
	}

	return users, data, nil
}

// ParseUser parses the chat history with one user.
// Internally, it calls ParseAll.
// It returns a Content arrat, and any encountered error.
func ParseUser(in io.Reader, user string) ([]Content, error) {
	_, data, err := ParseAll(in)
	if err != nil {
		return []Content{}, err
	}

	return data[user], nil
}
