package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Backup collects parsed files and the source root of one Snapchat export.
// Nil fields support programmatic partial imports; LoadBackup requires a complete
// export. This is an application type, not a JSON format.
type Backup struct {
	Account         *Account
	AccountHistory  *AccountHistory
	ChatHistory     *ChatHistory
	Friends         *Friends
	SnapHistory     *SnapHistory
	SnapProfile     *SnapProfile
	TalkHistory     *TalkHistory
	UserProfile     *UserProfile
	Filenames       []string
	ExportDirectory string
}

// LoadBackup loads an unchanged export root containing json/ and chat_media/.
// Every supported JSON file is required; additional JSON files are validated
// but not mapped to application types. Any read or parse error aborts the load
// before a database is opened or changed. Media paths are resolved by the viewer.
func LoadBackup(directory string) (Backup, error) {
	var backup Backup

	directory, err := filepath.Abs(directory)
	if err != nil {
		return backup, fmt.Errorf("resolve export directory: %w", err)
	}

	backup.ExportDirectory = directory

	jsonDirectory := filepath.Join(directory, "json")

	entries, err := os.ReadDir(jsonDirectory)

	if err != nil {
		return Backup{}, fmt.Errorf("read export json directory (provide the export root containing json/ and chat_media/): %w", err)
	}

	if _, err := os.ReadDir(filepath.Join(directory, "chat_media")); err != nil {
		return Backup{}, fmt.Errorf("read export chat_media directory: %w", err)
	}

	filenames := make([]string, 0, len(parsers))

	for name := range parsers {
		filenames = append(filenames, name)
	}

	slices.Sort(filenames)

	for _, name := range filenames {
		path := filepath.Join(jsonDirectory, name)

		file, err := os.Open(path)
		if err != nil {
			return Backup{}, fmt.Errorf("open required %s: %w", name, err)
		}

		parsed, parseErr := Parse(name, file)

		closeErr := file.Close()
		if parseErr != nil {
			return Backup{}, fmt.Errorf("parse %s: %w", name, parseErr)
		}

		if closeErr != nil {
			return Backup{}, fmt.Errorf("close %s: %w", name, closeErr)
		}

		switch value := parsed.(type) {
		case *Account:
			backup.Account = value
		case *AccountHistory:
			backup.AccountHistory = value
		case *ChatHistory:
			backup.ChatHistory = value
		case *Friends:
			backup.Friends = value
		case *SnapHistory:
			backup.SnapHistory = value
		case *SnapProfile:
			backup.SnapProfile = value
		case *TalkHistory:
			backup.TalkHistory = value
		case *UserProfile:
			backup.UserProfile = value
		default:
			return Backup{}, fmt.Errorf("unsupported parsed backup type for %s", name)
		}

		backup.Filenames = append(backup.Filenames, path)
	}

	for _, entry := range entries {
		name := entry.Name()

		if _, supported := parsers[name]; supported || !strings.EqualFold(filepath.Ext(name), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(jsonDirectory, name))
		if err != nil {
			return Backup{}, fmt.Errorf("read %s: %w", name, err)
		}

		if !json.Valid(data) {
			return Backup{}, fmt.Errorf("parse %s: invalid JSON", name)
		}
	}

	return backup, nil
}
