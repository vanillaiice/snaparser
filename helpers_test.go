package snaparser

import (
	"bytes"
	"testing"

	"github.com/vanillaiice/snaparser/parser"
)

func TestReplaceSlash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"NoSlashes", "noslashes", "noslashes"},
		{"SingleSlash", "single/slash", "single-slash"},
		{"MultipleSlashes", "multiple//slashes", "multiple--slashes"},
		{"NoSlash", "noslash", "noslash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.input
			replaceSlash(&s)
			if s != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, s)
			}
		})
	}
}

func TestCreateUserFile(t *testing.T) {
	user := "testuser"
	f, err := createUserFile(&user)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	if f.Name() != "testuser.txt" {
		t.Errorf("Expected filename testuser.txt, got %s", f.Name())
	}
}

func TestWriteContent(t *testing.T) {
	var buf bytes.Buffer
	contents := []*parser.Content{
		{MediaType: "TEXT", From: "Alice", Created: "2023-01-01", Content: "Hello"},
		{MediaType: "IMAGE", From: "Bob", Created: "2023-01-02", Content: "image.jpg"}, // Not TEXT media
		{MediaType: "TEXT", From: "Alice", Created: "2023-01-03", Content: "World"},
	}

	// Test without color
	err := writeContent(&buf, contents, false)
	if err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}
	expected := "Alice (2023-01-03): World\nAlice (2023-01-01): Hello\n"
	if buf.String() != expected {
		t.Errorf("Expected output:\n%s\nGot:\n%s", expected, buf.String())
	}

	// TODO: Test with color
}
