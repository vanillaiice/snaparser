package snaparser

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vanillaiice/snaparser/parser"
)

const MediaTypeText = "TEXT"

// checkIllegalString checks if a string contains '/' characters.
// If yes, they are replaced with '-'
func checkIllegalString(s *string) {
	if strings.Contains(*s, "/") {
		*s = strings.ReplaceAll(*s, "/", "-")
	}
}

// createUserFile creates a file where the chats with a user will be written
func createUserFile(user *string) (f *os.File, err error) {
	checkIllegalString(user)
	return os.Create(*user + ".txt")
}

// writeData writes the content of a parsed message to a writer
func writeData(content *parser.Content, w io.Writer) (err error) {
	if content.MediaType != MediaTypeText {
		return
	}
	s := fmt.Sprintf("%s (%s): %s\n", content.From, content.Created, content.Content)
	_, err = io.WriteString(w, s)
	return
}
