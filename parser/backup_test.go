package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func backupDirectory(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	for _, name := range []string{"json", "chat_media"} {
		if err := os.Mkdir(filepath.Join(directory, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for name := range parsers {
		if err := os.WriteFile(filepath.Join(directory, "json", name), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}

func TestLoadBackup(t *testing.T) {
	directory := backupDirectory(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, directory)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := LoadBackup(relative)
	if err != nil {
		t.Fatal(err)
	}
	if len(backup.Filenames) != 8 || backup.Account == nil || backup.AccountHistory == nil || backup.ChatHistory == nil || backup.Friends == nil || backup.SnapHistory == nil || backup.SnapProfile == nil || backup.TalkHistory == nil || backup.UserProfile == nil {
		t.Fatal("all supported backup types must be collected")
	}
	if backup.ExportDirectory != directory {
		t.Fatalf("source directory = %q, want absolute path %q", backup.ExportDirectory, directory)
	}
}

func TestLoadBackupRequiresIntactExport(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(string) error
	}{
		{"missing supported JSON", func(root string) error { return os.Remove(filepath.Join(root, "json", "friends.json")) }},
		{"missing media directory", func(root string) error { return os.Remove(filepath.Join(root, "chat_media")) }},
		{"malformed supported JSON", func(root string) error {
			return os.WriteFile(filepath.Join(root, "json", "friends.json"), []byte(`{"bad":`), 0600)
		}},
		{"malformed additional JSON", func(root string) error {
			return os.WriteFile(filepath.Join(root, "json", "bitmoji.json"), []byte(`{} {}`), 0600)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := backupDirectory(t)
			if err := tc.change(root); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadBackup(root); err == nil {
				t.Fatal("damaged export was accepted")
			}
		})
	}
	root := backupDirectory(t)
	if _, err := LoadBackup(filepath.Join(root, "json")); err == nil {
		t.Fatal("the json directory must not be accepted as the export root")
	}
	if err := os.WriteFile(filepath.Join(root, "json", "bitmoji.json"), []byte(`{"valid":"additional data"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBackup(root); err != nil {
		t.Fatalf("valid additional JSON should be accepted: %v", err)
	}
}

func TestLoadBackupMissingAndInvalidSources(t *testing.T) {
	if _, err := LoadBackup(t.TempDir()); err == nil {
		t.Fatal("an empty directory must not be treated as a replacement backup")
	}
	if _, err := LoadBackup(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("a missing directory should fail")
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "account.json")
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBackup(path); err == nil {
		t.Fatal("the source must be a directory")
	}
}

func TestParseRejectsIncompleteOrAdditionalDocuments(t *testing.T) {
	for _, input := range []string{`null`, `{} {}`, `{} garbage`, `{"broken":`, `[]`, ``} {
		t.Run(input, func(t *testing.T) {
			if _, err := Parse("account.json", strings.NewReader(input)); err == nil {
				t.Fatalf("invalid backup document accepted: %s", input)
			}
		})
	}
	parsed, err := Parse("account.json", strings.NewReader("{\"Basic Information\":{\"Username\":\"owner\"}}\n  "))
	if err != nil {
		t.Fatal(err)
	}
	if account, ok := parsed.(*Account); !ok || account.BasicInformation.Username != "owner" {
		t.Fatal("strict decoding must preserve the typed account result")
	}
}
