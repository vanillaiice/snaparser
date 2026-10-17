package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MediaIndex maps chat media's date-and-ID filenames to their existing paths.
// Its zero value is an empty index. Construct it once per export and reuse it
// for all chats; lookups do not access the filesystem.
type MediaIndex struct {
	paths map[string][]string
}

// NewMediaIndex scans the supplied chat_media directory. Only regular files
// named YYYY-MM-DD_<media ID>.<extension> are indexed. Paths retain the supplied
// directory prefix and actual extensions; multiple files for one key are kept
// in filename order. A missing or unreadable directory returns an error.
func NewMediaIndex(directory string) (MediaIndex, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return MediaIndex{}, fmt.Errorf("index chat media: %w", err)
	}

	index := MediaIndex{paths: make(map[string][]string)}

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		name := entry.Name()

		ext := filepath.Ext(name)

		if ext == "" || ext == "." {
			continue
		}

		key := strings.TrimSuffix(name, ext)

		date, id, ok := strings.Cut(key, "_")

		if !ok || id == "" {
			continue
		}

		if _, err := time.Parse(time.DateOnly, date); err != nil {
			continue
		}

		index.paths[key] = append(index.paths[key], filepath.Join(directory, name))
	}

	return index, nil
}

// ResolveMediaPaths returns indexed attachments for the chat's UTC Created date
// and pipe-separated Media IDs, in ID order, without duplicate paths. Missing
// files are omitted. Invalid or missing Created dates return no paths; IDs are
// never matched across dates because an ID can occur on more than one date.
func (c Chat) ResolveMediaPaths(index MediaIndex) []string {
	if strings.TrimSpace(c.MediaIDS) == "" {
		return nil
	}

	created, err := time.Parse("2006-01-02 15:04:05 UTC", c.Created)
	if err != nil {
		return nil
	}

	prefix := created.Format(time.DateOnly) + "_"

	var paths []string

	seen := make(map[string]bool)

	for _, id := range strings.Split(c.MediaIDS, "|") {
		id = strings.TrimSpace(id)

		if id == "" {
			continue
		}

		for _, path := range index.paths[prefix+id] {
			if !seen[path] {
				paths = append(paths, path)
				seen[path] = true
			}
		}
	}

	return paths
}
