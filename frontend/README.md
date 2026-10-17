# Embedded archive browser

Start directly from an unzipped Snapchat export; this creates the SQLite archive
and starts the viewer in one command:

```sh
go run ./cmd/snaparser serve --data ./data
# Open http://127.0.0.1:8080

# On subsequent runs, reuse the imported database without importing again:
go run ./cmd/snaparser serve

# Or choose an existing database:
go run ./cmd/snaparser serve --db /path/to/archive.db
```

`--data` requires the unchanged export root containing `json/` and `chat_media/`.
It parses every supported JSON file and validates additional JSON files before
atomically replacing the selected archive using `db.Import`. Missing directories,
missing supported JSON files, or malformed JSON abort without changing an
existing archive. Additional JSON contents are not imported. The absolute export
root is stored in the same transaction as the parsed data. The default database is
`snaparser.db`; `--db` selects another path. Import without starting the viewer
with `go run ./cmd/snaparser import --data ./data --db snaparser.db`.

`--listen` changes the default `127.0.0.1:8080` address. Without `--data`, the
command requires an existing database file. Opening it applies the existing
database package's migrations. Keep the source export at its imported location;
viewer startup aborts if its recorded media directory cannot be read. Archives
without a recorded source still display chats; reimport from the export root to
enable attachments. An empty archive gets a welcome state. There is
no demo data, and the viewer has no HTTP import endpoint.

To integrate with the final server, mount the handler at `/`:

```go
viewer, err := frontend.New(database) // *db.DB satisfies frontend.Store
if err != nil {
    return err
}
mux.Handle("/", viewer)
```

The caller owns the database and HTTP server lifecycle. The existing database
package remains the only data access layer. The frontend doesn't access its
connection, modify its schema, or perform imports. Its Store interface also lets
the future server supply its own adapter.

The handler serves complete Go `html/template` documents for normal requests
and `workspace` fragments for HTMX. Search replaces only `#archive`, preserving
input focus while typing. Links and forms also work without JavaScript. URLs
preserve search, selected conversation, and separate message/conversation pages;
browser history restoration receives a complete document.

Search treats input as literal word prefixes combined with AND, searching the
existing chats FTS index (message content, sender, and conversation title).
Quoted input is escaped rather than interpreted as FTS operators. Each request
fetches at most 41 chats and 26 conversation summaries to detect the next page.
Conversations render oldest-to-newest and open at the latest message. Scrolling
to the top automatically prepends earlier history without replacing the current
messages. Load older messages also works manually; failed loads offer a Retry
button. Latest messages returns to the bottom. The message pane
scrolls independently of the fixed page. Native links page through older and
newer history when HTMX is unavailable. Search results remain ranked by relevance
and use ordinary pagination. Times are UTC.
Click a search result's text or its View in conversation link to open the full
conversation around that message. The viewer clears the search, loads a window
of surrounding messages, scrolls to the result, and highlights its bubble. Older
and newer history remain available. Context links include a message ID and
fragment so they also work without JavaScript, on refresh, and in a new tab.
Browser Back returns to the search results.
Exports with milliseconds mislabeled as microseconds are normalized on import;
existing databases are corrected automatically when opened, without reimporting.
Messages use pale yellow incoming bubbles on the left and light blue outgoing
bubbles on the right, with sender names, timestamps, and saved status.
The desktop conversation sidebar becomes a native selector on phones. The
header's Theme selector offers System, Light, and Dark. CSS applies the choice
immediately; HTMX persists it in a one-year cookie without replacing the page
or losing the active search or scroll position. With JavaScript disabled, use
the Apply button. System follows the device's appearance setting. All UI palette values live
in the two theme blocks at the top of `assets/app.css`; component rules only
reference those shared color variables.

After JSON import, the viewer builds one media filename index at startup and
resolves attachments for each displayed chat using its UTC `Created` date and
pipe-separated `Media IDs`. Matching requires the exact `YYYY-MM-DD_<ID>` stem;
the actual filename supplies the extension. Images appear inline, video and audio
use native controls, and other types get download links. Captions and multiple
attachments are supported. Snapchat `NOTE` messages use audio controls even when
their files have an `.mp4` extension. Missing files retain the unavailable state.
`/media/{filename}` reads only regular files inside the recorded `chat_media`
directory, supports HEAD and byte ranges for playback, and disables caching.
Restart the viewer after importing another export. Media files are not embedded
or copied to the database.

All application assets use `go:embed`: plain CSS, an SVG icon, and the unmodified
[HTMX 2.0.11 distribution](https://htmx.org/docs/#installing), with its license
in `assets/HTMX-LICENSE`. No custom JavaScript, inline scripts, CDN requests,
remote fonts, Node, or frontend build step. Archive responses use `no-store`,
and HTMX history snapshots are disabled to keep messages out of localStorage.

```sh
go test ./frontend ./cmd/snaparser
go build ./cmd/snaparser
```
