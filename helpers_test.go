package snaparser

import (
	"testing"
)

func TestCheckIllegalString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "NoSlashInString",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "SingleSlashInString",
			input:    "hello/world",
			expected: "hello-world",
		},
		{
			name:     "MultipleSlashesInString",
			input:    "hello/world/foo/bar",
			expected: "hello-world-foo-bar",
		},
		{
			name:     "EmptyString",
			input:    "",
			expected: "",
		},
		{
			name:     "StringWithLeadingSlash",
			input:    "/hello/world",
			expected: "-hello-world",
		},
		{
			name:     "StringWithTrailingSlash",
			input:    "hello/world/",
			expected: "hello-world-",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			checkIllegalString(&input)
			if input != tc.expected {
				t.Errorf("CheckIllegalString(%q) = %q, expected %q", tc.input, input, tc.expected)
			}
		})
	}
}
