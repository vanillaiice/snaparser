package snaparser

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/vanillaiice/snaparser/parser"
)

// MediaTypeText represents a media that is text
const MediaTypeText = "TEXT"

// replaceSlash checks if a string contains '/' characters.
// If yes, they are replaced with '-'.
func replaceSlash(s *string) {
	if strings.Contains(*s, "/") {
		*s = strings.ReplaceAll(*s, "/", "-")
	}
}

// createUserFile creates a file where the chats with a user will be written
func createUserFile(user *string) (f *os.File, err error) {
	replaceSlash(user)
	return os.Create(*user + ".txt")
}

// writeContent writes the content of a parsed message to a writer.
func writeContent(w io.Writer, c []*parser.Content, withColor bool) (err error) {
	for i := len(c) - 1; i >= 0; i-- {
		if c[i].MediaType != MediaTypeText {
			continue
		}

		if withColor {
			c[i].Created = color.GreenString(c[i].Created)
			if i%2 == 0 {
				c[i].From = color.YellowString(c[i].From)
			} else {
				c[i].From = color.RedString(c[i].From)
			}
		}

		s := fmt.Sprintf("%s (%s): %s\n", c[i].From, c[i].Created, c[i].Content)

		if _, err = io.WriteString(w, s); err != nil {
			return
		}
	}

	return
}
