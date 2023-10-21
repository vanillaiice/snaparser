package parser

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Data struct {
	DataMap map[string][]Content
}

type Content struct {
	From               string
	MediaType          string `json:"Media Type"`
	Created            string
	Content            string
	ConversationTitle  string `json:"Conversation Title"`
	IsSender           bool   `json:"isSender"`
	CreatedMicrosecond int64  `json:"Created(microseconds)"`
}

func ParseAll(file *os.File) ([]string, map[string][]Content, error) {
	var data map[string][]Content
	var users []string

	fileByte, err := ioutil.ReadAll(file)
	if err != nil {
		return users, data, err
	}

	err = json.Unmarshal(fileByte, &data)
	if err != nil {
		return users, data, err
	}

	for k, _ := range data {
		users = append(users, k)
	}

	return users, data, nil
}

func ParseUser(file *os.File, user string) ([]Content, error) {
	var data map[string][]Content

	fileByte, err := ioutil.ReadAll(file)
	if err != nil {
		return data[user], err
	}

	err = json.Unmarshal(fileByte, &data)
	if err != nil {
		return data[user], err
	}

	return data[user], nil
}
