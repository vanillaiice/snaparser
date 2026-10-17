package parser

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

// errorReader is a custom io.Reader that always returns an error.
type errorReader struct{}

func (er *errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("simulated read error")
}

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		filename string
		input    string
		want     any
	}{
		{
			"account.json",
			`{"Basic Information":{"Username":"owner"}}`,
			&Account{BasicInformation: BasicInformation{Username: "owner"}},
		},
		{
			"account_history.json",
			`{"Display Name Change":[{"Date":"date","Display Name":"Owner"}]}`,
			&AccountHistory{DisplayNameChange: []DisplayNameChange{{Date: "date", DisplayName: "Owner"}}},
		},
		{
			"chat_history.json",
			`{"friend":[{"From":"friend","Media Type":"PHOTO","isSender":true,"Media IDs":"image"}]}`,
			&ChatHistory{"friend": {{From: "friend", MediaType: "PHOTO", IsSender: true, MediaIDS: "image"}}},
		},
		{
			"friends.json",
			`{"Friend Requests Sent":[{"Username":"friend","Display Name":"Friend"}]}`,
			&Friends{FriendsRequestsSent: []Friend{{Username: "friend", DisplayName: "Friend"}}},
		},
		{
			"snap_history.json",
			`{"friend":[{"From":"owner","IsSender":true,"Created(microseconds)":1700000000000000}]}`,
			&SnapHistory{"friend": {{From: "owner", IsSender: true, CreatedMicroseconds: 1_700_000_000_000_000}}},
		},
		{
			"snap_pro.json",
			`{"Profile":{"Profile Title":"Owner","Profile Website":"https://example.test"}}`,
			&SnapProfile{Profile: Profile{ProfileTitle: "Owner", ProfileWebsite: "https://example.test"}},
		},
		{
			"talk_history.json",
			`{"Incoming Calls":[{"Date & Time":"date","People in Chat":2,"Length (sec)":42}]}`,
			&TalkHistory{IncomingCalls: []Call{{DateTime: "date", PeopleInChat: 2, Length: 42}}},
		},
		{
			"user_profile.json",
			`{"App Profile":{"In-app Language":"en"},"Interactions":{"Web Interactions":["example.test"]}}`,
			&UserProfile{AppProfile: AppProfile{InAppLanguage: "en"}, Interactions: Interactions{WebInteractions: []string{"example.test"}}},
		},
	} {
		t.Run(tc.filename, func(t *testing.T) {
			parsed, err := Parse("export/json/"+tc.filename, bytes.NewBufferString(tc.input))
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(parsed, tc.want) {
				t.Fatalf("parsed = %#v, want %#v", parsed, tc.want)
			}
		})
	}
}

func TestParseUnsupportedFileAndReadError(t *testing.T) {
	if _, err := Parse("unknown.json", bytes.NewBufferString(`{}`)); err == nil {
		t.Fatal("unsupported file accepted")
	}

	if _, err := Parse("account.json", &errorReader{}); err == nil {
		t.Fatal("read error ignored")
	}
}
