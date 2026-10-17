package parser

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveMediaPaths(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{
		"2025-09-15_b~photo.jpg",
		"2025-09-15_b~video.mp4",
		"2025-09-15_b~variant.png",
		"2025-09-15_b~variant.webp",
		"2025-09-14_b~photo.png",
		"2025-09-14_b~other-date.jpg",
		"2025-09-15_b~under_score.gif",
		"2025-09-15_b~opaque.unknown",
		"2025-02-30_b~invalid.jpg",
		"2025-09-15_.jpg",
		"2025-09-15_b~no-extension",
		"2025-09-15_b~empty-extension.",
		"notes.txt",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(directory, "2025-09-15_b~directory.jpg"), 0700); err != nil {
		t.Fatal(err)
	}
	index, err := NewMediaIndex(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		created string
		ids     string
		want    []string
	}{
		{name: "actual extension", created: "2025-09-15 19:32:37 UTC", ids: "b~photo", want: []string{"2025-09-15_b~photo.jpg"}},
		{name: "same ID on another date", created: "2025-09-14 23:59:59 UTC", ids: "b~photo", want: []string{"2025-09-14_b~photo.png"}},
		{name: "multiple IDs and missing file", created: "2025-09-15 00:00:00 UTC", ids: " b~video | missing | b~photo ", want: []string{"2025-09-15_b~video.mp4", "2025-09-15_b~photo.jpg"}},
		{name: "empty and duplicate IDs", created: "2025-09-15 12:00:00 UTC", ids: " | b~photo || b~photo | ", want: []string{"2025-09-15_b~photo.jpg"}},
		{name: "multiple extensions", created: "2025-09-15 12:00:00 UTC", ids: "b~variant", want: []string{"2025-09-15_b~variant.png", "2025-09-15_b~variant.webp"}},
		{name: "ID containing underscore", created: "2025-09-15 12:00:00 UTC", ids: "b~under_score", want: []string{"2025-09-15_b~under_score.gif"}},
		{name: "unknown extension", created: "2025-09-15 12:00:00 UTC", ids: "b~opaque", want: []string{"2025-09-15_b~opaque.unknown"}},
		{name: "no match across dates", created: "2025-09-15 12:00:00 UTC", ids: "b~other-date"},
		{name: "missing attachment", created: "2025-09-15 12:00:00 UTC", ids: "missing"},
		{name: "invalid date", created: "2025-02-30 12:00:00 UTC", ids: "b~invalid"},
		{name: "invalid time", created: "2025-09-15 25:00:00 UTC", ids: "b~photo"},
		{name: "missing date", ids: "b~photo"},
		{name: "non-UTC date", created: "2025-09-15 12:00:00 PST", ids: "b~photo"},
		{name: "empty IDs", created: "2025-09-15 12:00:00 UTC", ids: " | "},
		{name: "directory is not an attachment", created: "2025-09-15 12:00:00 UTC", ids: "b~directory"},
		{name: "no extension", created: "2025-09-15 12:00:00 UTC", ids: "b~no-extension | b~empty-extension"},
		{name: "literal ID lookup", created: "2025-09-15 12:00:00 UTC", ids: "b~* | ../b~photo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var want []string
			for _, name := range tc.want {
				want = append(want, filepath.Join(directory, name))
			}
			chat := Chat{Created: tc.created, MediaIDS: tc.ids}
			if got := chat.ResolveMediaPaths(index); !reflect.DeepEqual(got, want) {
				t.Fatalf("resolved paths = %v, want %v", got, want)
			}
		})
	}
	chat := Chat{Created: "2025-09-15 19:32:37 UTC", MediaIDS: "b~photo"}
	if got := chat.ResolveMediaPaths(MediaIndex{}); len(got) != 0 {
		t.Fatalf("zero index resolved paths: %v", got)
	}
}

func TestNewMediaIndexDirectoryErrors(t *testing.T) {
	directory := t.TempDir()
	if _, err := NewMediaIndex(filepath.Join(directory, "missing")); err == nil {
		t.Fatal("missing media directory must report an error")
	}
	file := filepath.Join(directory, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewMediaIndex(file); err == nil {
		t.Fatal("a file cannot be used as a media directory")
	}
	index, err := NewMediaIndex(directory)
	if err != nil || len(index.paths) != 0 {
		t.Fatalf("directory without matching files: index = %v, error = %v", index, err)
	}
}
