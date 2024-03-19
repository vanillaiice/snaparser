package snaparser

import "strings"

// checkIllegalString checks if a string contains '/' characters.
// If yes, they are replaced with '-'
func CheckIllegalString(str *string) {
	if strings.Contains(*str, "/") == true {
		*str = strings.ReplaceAll(*str, "/", "-")
	}
}
