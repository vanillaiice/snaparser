# Snaparser

Snaparser parses snapchat chat history json files to human friendly format.
Snaparser has three main components, the parser, sqlite database, and htmx frontend, all in one go binary.

Create the database from an unzipped Snapchat export and open the embedded chat
browser in one command:

```sh
go run ./cmd/snaparser serve --data ./data
# http://127.0.0.1:8080

# Later, reuse the database without reimporting:
go run ./cmd/snaparser serve
```

`--data` requires the unchanged export root containing `json/` and `chat_media/`.
Keep this directory in place: the viewer reads attachments from it. `--db` selects the
SQLite path (default `snaparser.db`). To import without serving, use
`snaparser import --data ./data --db snaparser.db`. Importing replaces the current
backup only after all required JSON files parse successfully and additional JSON
files validate. Missing directories, missing supported files, or invalid JSON
abort the import without changing an existing archive.

See [frontend/README.md](frontend/README.md) for server integration and the current UI scope.

# Architecture

The database represents one complete Snapchat backup at a time. Importing a new
backup replaces all application data. There is no account matching, UPSERT,
deduplication, or incremental synchronization.

The database API is:

```go
func Open(path string) (*DB, error)
func (db *DB) Close() error
func (db *DB) Import(ctx context.Context, backup parser.Backup) error
```

`parser.Backup` collects the typed results from parsing the supplied JSON files,
along with their filenames and absolute export root for import metadata. It is
not a new JSON format. `LoadBackup` requires every supported JSON file under
`json/`; it validates additional JSON files without importing their contents.
The lower-level `db.Import` API also accepts programmatically supplied partial
backups. Omitted fields clear the corresponding previous data.

The WebUI reads the saved export root and builds one index of its `chat_media`
directory at startup. It resolves attachments as chats are rendered. The same
resolver is available to other callers:

```go
index, err := parser.NewMediaIndex(filepath.Join(exportDirectory, "chat_media"))
if err != nil {
    return err
}
paths := chat.ResolveMediaPaths(index) // chat is a parser.Chat
```

Resolution matches the UTC date in `Created` and each pipe-separated `Media IDs`
value against `YYYY-MM-DD_<ID>.<extension>`. Returned filesystem paths include
the supplied directory and actual file extensions. Missing attachments are
omitted; IDs are never matched across dates. The index is a snapshot of filenames;
rebuild it if the media directory changes. The WebUI displays images, video and
audio controls, and download links for other file types. Missing media displays
an unavailable placeholder. Media bytes stay in the export; paths and files are
not copied into SQLite. Restart the viewer after importing another export.
Archives imported before source tracking need reimporting to enable media.

The import implementation follows this sequence:

1. Parse and validate every supplied file before calling `Import`.
2. Begin one database transaction.
3. Delete application data, removing chats and snaps before conversations.
4. Insert the new account, profiles, friends, conversations, messages, histories,
   and other supported parsed data.
5. Record import metadata and the absolute export root for the new backup.
6. Commit once every operation succeeds; roll back on any insertion or commit
   failure. A parsing failure must never start replacement.

The previous backup must remain intact if parsing or insertion fails. Keep the
schema, migration history (`schema_migrations`), indexes, and triggers between
imports. Delete source rows only; FTS5 triggers maintain the search indexes
within the same transaction.

Account and profile tables are singletons, enforced by a primary key constrained
to `1`. Other tables do not need account ownership foreign keys. Chats and snaps
retain their foreign keys to conversations. The initial schema and FTS migration
describe this layout for new databases; an existing database that has already
applied them would require a separate schema migration.

# Existing solution

Snapchat provides a static website when downloading your [data](https://help.snapchat.com/hc/en-us/articles/7012305371156-How-do-I-download-my-data-from-Snapchat-). It can be used
to view your login history, account info, snap & chat history metadata and a
bunch of other useful stuff. However, there is no search or filtering, and the
ui is kinda boring.

If you still want to use it, you can run the command below:

```
$ python3 -m http.server --directory data/
```
> with data holding the content of your unzipped data

Else, continue reading...

# Installation

```
$ go install github.com/vanillaiice/snaparser/cmd/snaparser@latest
```

# Usage

To download your chat history data, follow the guide available on snapchat's 
[website](https://help.snapchat.com/hc/en-us/articles/7012305371156-How-do-I-download-my-data-from-Snapchat-). 
You can then do the following:

```sh
# Import the unchanged export and start the viewer
snaparser serve --data /path/to/snapchat-export

# Import without starting the viewer
snaparser import --data /path/to/snapchat-export --db archive.db

# Reuse the imported archive and its recorded media directory
snaparser serve --db archive.db
```

# Flags

- `--data PATH`: export root containing `json/` and `chat_media/`; required for
  `import`, optional for `serve`.
- `--db PATH`: SQLite archive, default `snaparser.db`.
- `--listen ADDRESS`: viewer address, default `127.0.0.1:8080` (`serve` only).
- `--help`: show command-specific help.

# Author

vanillaiice

# License

BSD-3-Clause
